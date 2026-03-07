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

$composeFiles = @()
if (-not [string]::IsNullOrWhiteSpace($env:REFARCH_COMPOSE)) {
  if (-not (Test-Path $env:REFARCH_COMPOSE)) {
    throw "REFARCH_COMPOSE file not found: $($env:REFARCH_COMPOSE)"
  }
  $composeFiles = @((Resolve-Path $env:REFARCH_COMPOSE).Path)
} else {
  $composeFiles = @(Get-ChildItem -Path (Join-Path $repoRoot 'reference-architecture') -Filter 'docker-compose.yml' -Recurse -File |
    Where-Object { $_.FullName -like "*\$Provider\example\docker-compose.yml" } |
    Sort-Object FullName |
    Select-Object -ExpandProperty FullName)
}

if ($composeFiles.Count -eq 0) {
  Write-Host "No reference architecture compose files found for provider '$Provider'"
  exit 0
}

function Wait-StackyardReady {
  param([string]$ComposeFile)
  for ($i = 0; $i -lt 90; $i++) {
    $logs = docker compose -f $ComposeFile logs --no-color stackyard 2>$null
    if ($logs -match 'stackyard listening on') {
      return $true
    }
    Start-Sleep -Seconds 1
  }
  return $false
}

function Wait-BackendReady {
  param([string]$BaseUrl)
  for ($i = 0; $i -lt 90; $i++) {
    try {
      $resp = Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/healthz" -TimeoutSec 2
      if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 300) {
        return $true
      }
    } catch {
      # continue
    }
    Start-Sleep -Seconds 1
  }
  return $false
}

foreach ($composeFile in $composeFiles) {
  Write-Host "==> Running reference architecture compose: $composeFile"

  $services = @(docker compose -f $composeFile config --services)
  if ($services.Count -eq 0) {
    throw "No services found in $composeFile"
  }

  $backendService = $services | Where-Object { $_ -eq 'backend' } | Select-Object -First 1
  if (-not $backendService) {
    $backendService = $services | Where-Object { $_ -ne 'stackyard' } | Select-Object -First 1
  }

  try { docker compose -f $composeFile down --remove-orphans -v | Out-Null } catch {}

  $status = 0
  if (($services -contains 'stackyard') -and $backendService) {
    docker compose -f $composeFile up -d --build stackyard $backendService
    if ($LASTEXITCODE -ne 0) { $status = $LASTEXITCODE }

    if ($status -eq 0 -and -not (Wait-StackyardReady -ComposeFile $composeFile)) {
      Write-Host "Stackyard health check failed for compose: $composeFile"
      docker compose -f $composeFile logs stackyard
      $status = 1
    }

    if ($status -eq 0) {
      $portOut = docker compose -f $composeFile port $backendService 8080 2>$null | Select-Object -First 1
      if (-not [string]::IsNullOrWhiteSpace($portOut)) {
        $backendPort = ($portOut -split ':')[-1]
        $backendUrl = "http://127.0.0.1:$backendPort"
        if (-not (Wait-BackendReady -BaseUrl $backendUrl)) {
          Write-Host "Backend health check failed for compose: $composeFile"
          docker compose -f $composeFile logs $backendService
          $status = 1
        }
      }
    }
  } else {
    docker compose -f $composeFile up --build --abort-on-container-exit
    if ($LASTEXITCODE -ne 0) { $status = $LASTEXITCODE }
  }

  try { docker compose -f $composeFile down --remove-orphans -v | Out-Null } catch {}

  if ($status -ne 0) {
    throw "Reference architecture compose failed: $composeFile (exit code $status)"
  }
}

Write-Host 'All reference architecture compose runs completed successfully.'
