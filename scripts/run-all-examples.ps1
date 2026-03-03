param(
  [string]$Provider = $env:PROVIDER
)

$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $repoRoot
if ([string]::IsNullOrWhiteSpace($Provider)) {
  $Provider = 'aws'
}
$Provider = $Provider.ToLowerInvariant()
$examplesRoot = Join-Path $repoRoot (Join-Path 'examples' $Provider)
if (-not (Test-Path $examplesRoot -PathType Container)) {
  if ($env:ALLOW_MISSING_PROVIDER_DIR -eq '1') {
    Write-Host "Skipping provider '$Provider': examples directory not found at $examplesRoot"
    exit 0
  }
  $availableProviders = @(Get-ChildItem -Path (Join-Path $repoRoot 'examples') -Directory | Select-Object -ExpandProperty Name | Sort-Object)
  $availableText = if ($availableProviders.Count -gt 0) { $availableProviders -join ', ' } else { '(none found)' }
  throw "Provider examples directory not found: $examplesRoot`nAvailable providers: $availableText"
}

function Wait-StackyardReady {
  param(
    [string[]]$ComposeArgs,
    [int]$Attempts = 90,
    [int]$DelaySec = 1
  )

  for ($i = 0; $i -lt $Attempts; $i++) {
    try {
      $logs = docker compose @ComposeArgs logs --no-color stackyard 2>$null
      if ($logs -match 'stackyard listening on') {
        return $true
      }
    } catch {
      # keep waiting
    }
    Start-Sleep -Seconds $DelaySec
  }
  return $false
}

function New-StackyardOverrideFile {
  $overridePath = [System.IO.Path]::GetTempFileName()
  $yaml = @"
services:
  stackyard:
    container_name: !reset null
    ports: !reset []
"@
  Set-Content -Path $overridePath -Value $yaml -Encoding utf8
  return $overridePath
}

$composeFiles = @()
if (-not [string]::IsNullOrWhiteSpace($env:EXAMPLE_COMPOSE)) {
  $singleCompose = $env:EXAMPLE_COMPOSE
  if (-not (Test-Path $singleCompose)) {
    throw "EXAMPLE_COMPOSE file not found: $singleCompose"
  }
  $composeFiles = @(Get-Item $singleCompose)
} elseif ($env:RUN_ALL_EXAMPLES -eq '1') {
  $composeFiles = Get-ChildItem -Path $examplesRoot -Filter 'docker-compose.yml' -Recurse -File |
    Sort-Object FullName
} else {
  $serviceDirs = Get-ChildItem -Path $examplesRoot -Directory | Sort-Object FullName
  foreach ($serviceDir in $serviceDirs) {
    $firstCompose = Get-ChildItem -Path $serviceDir.FullName -Filter 'docker-compose.yml' -Recurse -File |
      Sort-Object FullName |
      Select-Object -First 1
    if ($firstCompose) {
      $composeFiles += $firstCompose
    }
  }
}

if (-not $composeFiles -or $composeFiles.Count -eq 0) {
  Write-Host "No example docker-compose files found under $examplesRoot/*/*/docker-compose.yml"
  exit 0
}

foreach ($compose in $composeFiles) {
  $composePath = $compose.FullName
  Write-Host "==> Running example compose: $composePath"

  $baseComposeArgs = @('-f', $composePath)
  $composeImages = @(docker compose @baseComposeArgs config --images 2>$null | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Sort-Object -Unique)

  $services = docker compose @baseComposeArgs config --services
  if (-not $services -or $services.Count -eq 0) {
    throw "No services found in $composePath"
  }

  $exitService = $services | Where-Object { $_ -ne 'stackyard' } | Select-Object -First 1
  if (-not $exitService) {
    $exitService = $services[0]
  }
  $hasStackyard = $services -contains 'stackyard'
  $nonStackyardCount = ($services | Where-Object { $_ -ne 'stackyard' }).Count

  $composeArgs = @('-f', $composePath)
  $overrideFile = $null
  if ($hasStackyard) {
    $overrideFile = New-StackyardOverrideFile
    $composeArgs += @('-f', $overrideFile)
  }

  try {
    docker compose @composeArgs down --remove-orphans -v | Out-Null
  } catch {
    # ignore teardown errors before run
  }

  $status = 0
  if ($hasStackyard -and $nonStackyardCount -eq 1 -and $exitService -ne 'stackyard') {
    docker compose @composeArgs up -d --build stackyard
    if ($LASTEXITCODE -ne 0) {
      $status = $LASTEXITCODE
    } elseif (-not (Wait-StackyardReady -ComposeArgs $composeArgs)) {
      Write-Host "Stackyard health check failed for compose: $composePath"
      docker compose @composeArgs logs stackyard
      $status = 1
    } else {
      docker compose @composeArgs up --build --no-deps --abort-on-container-exit --exit-code-from $exitService $exitService
      if ($LASTEXITCODE -ne 0) {
        $status = $LASTEXITCODE
      }
    }
  } else {
    docker compose @composeArgs up --build --abort-on-container-exit --exit-code-from $exitService
    if ($LASTEXITCODE -ne 0) {
      $status = $LASTEXITCODE
    }
  }

  if ($status -ne 0) {
    try {
      docker compose @composeArgs down --remove-orphans -v | Out-Null
    } catch {
      # ignore teardown errors on failure path
    }
    if ($overrideFile -and (Test-Path $overrideFile)) {
      Remove-Item -Force $overrideFile
    }
    throw "Example compose failed: $composePath (exit code $status)"
  }

  try {
    docker compose @composeArgs down --remove-orphans -v | Out-Null
  } catch {
    # ignore teardown errors after successful run
  }
  if ($overrideFile -and (Test-Path $overrideFile)) {
    Remove-Item -Force $overrideFile
  }

  foreach ($image in $composeImages) {
    try {
      docker image rm -f $image | Out-Null
    } catch {
      # ignore image cleanup errors
    }
  }
}

Write-Host 'All example docker-compose runs completed successfully.'
