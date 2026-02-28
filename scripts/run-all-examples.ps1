$ErrorActionPreference = 'Stop'

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Set-Location $repoRoot

function Wait-StackyardHealth {
  param(
    [int]$Attempts = 60,
    [int]$DelaySec = 1
  )

  $url = 'http://localhost:4566/_stackyard/health'
  for ($i = 0; $i -lt $Attempts; $i++) {
    try {
      $resp = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 2
      if ($resp.StatusCode -eq 200) {
        return $true
      }
    } catch {
      # keep waiting
    }
    Start-Sleep -Seconds $DelaySec
  }
  return $false
}

$composeFiles = @()
if ($env:RUN_ALL_EXAMPLES -eq '1') {
  $composeFiles = Get-ChildItem -Path (Join-Path $repoRoot 'examples') -Filter 'docker-compose.yml' -Recurse -File |
    Where-Object { $_.FullName -match [regex]::Escape((Join-Path 'examples' '')) + '.+[\\/].+[\\/]docker-compose\.yml$' } |
    Sort-Object FullName
} else {
  $serviceDirs = Get-ChildItem -Path (Join-Path $repoRoot 'examples') -Directory | Sort-Object FullName
  $preferredVariant = $env:EXAMPLE_VARIANT
  if ([string]::IsNullOrWhiteSpace($preferredVariant)) {
    $preferredVariant = 'advanced'
  }
  foreach ($serviceDir in $serviceDirs) {
    $serviceName = $serviceDir.Name
    $preferredCompose = Join-Path $serviceDir.FullName (Join-Path "$serviceName-$preferredVariant" 'docker-compose.yml')
    if (Test-Path $preferredCompose) {
      $composeFiles += Get-Item $preferredCompose
      continue
    }

    $fallbackVariant = if ($preferredVariant -ieq 'advanced') { 'basic' } else { 'advanced' }
    $fallbackCompose = Join-Path $serviceDir.FullName (Join-Path "$serviceName-$fallbackVariant" 'docker-compose.yml')
    if (Test-Path $fallbackCompose) {
      $composeFiles += Get-Item $fallbackCompose
      continue
    }

    $firstCompose = Get-ChildItem -Path $serviceDir.FullName -Filter 'docker-compose.yml' -Recurse -File |
      Sort-Object FullName |
      Select-Object -First 1
    if ($firstCompose) {
      $composeFiles += $firstCompose
    }
  }
}

if (-not $composeFiles -or $composeFiles.Count -eq 0) {
  Write-Host 'No example docker-compose files found under examples/*/*/docker-compose.yml'
  exit 0
}

foreach ($compose in $composeFiles) {
  $composePath = $compose.FullName
  Write-Host "==> Running example compose: $composePath"

  $composeImages = @(docker compose -f $composePath config --images 2>$null | Where-Object { -not [string]::IsNullOrWhiteSpace($_) } | Sort-Object -Unique)

  $services = docker compose -f $composePath config --services
  if (-not $services -or $services.Count -eq 0) {
    throw "No services found in $composePath"
  }

  $exitService = $services | Where-Object { $_ -ne 'stackyard' } | Select-Object -First 1
  if (-not $exitService) {
    $exitService = $services[0]
  }
  $hasStackyard = $services -contains 'stackyard'
  $nonStackyardCount = ($services | Where-Object { $_ -ne 'stackyard' }).Count

  try {
    docker compose -f $composePath down --remove-orphans -v | Out-Null
  } catch {
    # ignore teardown errors before run
  }

  $status = 0
  if ($hasStackyard -and $nonStackyardCount -eq 1 -and $exitService -ne 'stackyard') {
    try {
      docker rm -f stackyard | Out-Null
    } catch {
      # ignore if container doesn't exist
    }
    docker compose -f $composePath up -d --build stackyard
    if ($LASTEXITCODE -ne 0) {
      $status = $LASTEXITCODE
    } elseif (-not (Wait-StackyardHealth)) {
      Write-Host "Stackyard health check failed for compose: $composePath"
      docker compose -f $composePath logs stackyard
      $status = 1
    } else {
      docker compose -f $composePath up --build --no-deps --abort-on-container-exit --exit-code-from $exitService $exitService
      if ($LASTEXITCODE -ne 0) {
        $status = $LASTEXITCODE
      }
    }
  } else {
    docker compose -f $composePath up --build --abort-on-container-exit --exit-code-from $exitService
    if ($LASTEXITCODE -ne 0) {
      $status = $LASTEXITCODE
    }
  }

  if ($status -ne 0) {
    try {
      docker compose -f $composePath down --remove-orphans -v | Out-Null
    } catch {
      # ignore teardown errors on failure path
    }
    throw "Example compose failed: $composePath (exit code $status)"
  }

  try {
    docker compose -f $composePath down --remove-orphans -v | Out-Null
  } catch {
    # ignore teardown errors after successful run
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
