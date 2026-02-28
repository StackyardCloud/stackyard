param(
  [switch]$Verbose,
  [switch]$Aws,
  [switch]$Reset
)

$ErrorActionPreference = "Stop"
$success = $false

$baseUrl = $env:STACKYARD_URL
if ([string]::IsNullOrWhiteSpace($baseUrl)) {
  $baseUrl = "http://localhost:4566"
}
$healthAttempts = 30
$healthDelaySeconds = 1
if (-not [string]::IsNullOrWhiteSpace($env:STACKYARD_HEALTH_ATTEMPTS)) {
  $healthAttempts = [int]$env:STACKYARD_HEALTH_ATTEMPTS
}
if (-not [string]::IsNullOrWhiteSpace($env:STACKYARD_HEALTH_DELAY_SEC)) {
  $healthDelaySeconds = [int]$env:STACKYARD_HEALTH_DELAY_SEC
}

Write-Host "Stackyard smoke test against $baseUrl"

if ($Reset) {
  if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker not found; cannot reset container."
  }
  if ($Verbose) {
    Write-Host "+ docker compose down"
    Write-Host "+ docker compose up -d --build"
  }
  docker compose down --remove-orphans | Out-Null
  try {
    docker rm -f stackyard 2>$null | Out-Null
  } catch {
  }
  docker compose up -d --build | Out-Null
}

function Assert-Status {
  param(
    [int]$Status,
    [int[]]$Allowed
  )
  if ($Allowed -contains $Status) {
    return
  }
  throw "Unexpected HTTP status: $Status"
}

function Invoke-CurlStatus {
  param(
    [string]$Label,
    [int[]]$Allowed,
    [string[]]$CurlArgs,
    [string]$Body
  )

  if ($Verbose) {
    Write-Host "+ curl.exe $($CurlArgs -join ' ')"
    if ($Body) {
      Write-Host $Body
    }
  }

  if ($Body) {
    $output = $Body | curl.exe -sS -w "`n%{http_code}" @CurlArgs --data-binary "@-"
  } else {
    $output = curl.exe -sS -w "`n%{http_code}" @CurlArgs
  }

  $outputText = $output -join "`n"
  $lines = $outputText -split "`n"
  $status = [int]$lines[-1].Trim()
  $bodyText = ""
  if ($lines.Length -gt 1) {
    $bodyText = ($lines[0..($lines.Length - 2)] -join "`n")
  }

  if ($Verbose -and -not [string]::IsNullOrWhiteSpace($bodyText)) {
    Write-Host $bodyText
  }

  Write-Host $Label
  Assert-Status $status $Allowed
}

function Wait-ForHealth {
  param(
    [string]$Url,
    [int]$Attempts,
    [int]$DelaySeconds
  )

  for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
    $statusText = curl.exe -sS -o NUL -w "%{http_code}" "$Url" 2>$null
    if ($LASTEXITCODE -eq 0 -and $statusText -eq "200") {
      Write-Host "Health check..."
      return
    }
    if ($Verbose) {
      Write-Host "Waiting for health ($attempt/$Attempts)..."
    }
    Start-Sleep -Seconds $DelaySeconds
  }

  throw "Health check failed after $Attempts attempts."
}

Wait-ForHealth -Url "$baseUrl/_stackyard/health" -Attempts $healthAttempts -DelaySeconds $healthDelaySeconds

Invoke-CurlStatus "Create bucket..." @(201, 409) @("-X", "PUT", "$baseUrl/s3/buckets/demo") ""

Invoke-CurlStatus "List buckets..." @(200) @("$baseUrl/s3/buckets") ""

Invoke-CurlStatus "Put object..." @(201) @("-X", "PUT", "--data", "hello stackyard", "-H", "Content-Type: text/plain", "$baseUrl/s3/objects/demo/notes/hello.txt") ""

Invoke-CurlStatus "Get object..." @(200) @("$baseUrl/s3/objects/demo/notes/hello.txt") ""

Invoke-CurlStatus "Create queue..." @(201, 409) @("-X", "PUT", "$baseUrl/sqs/queues/jobs") ""

$sendPayload = @'
{"body":"run build"}
'@
Invoke-CurlStatus "Send message..." @(201) @("-X", "POST", "$baseUrl/sqs/messages/jobs", "-H", "Content-Type: application/json") $sendPayload

$receivePayload = @'
{"max_messages":1}
'@
Invoke-CurlStatus "Receive message..." @(200) @("-X", "POST", "$baseUrl/sqs/messages/jobs/receive", "-H", "Content-Type: application/json") $receivePayload

if ($Aws) {
  if (-not (Get-Command aws -ErrorAction SilentlyContinue)) {
    Write-Warning "AWS CLI not found; skipping SigV4/XML smoke tests."
  } else {
    if (-not $env:AWS_ACCESS_KEY_ID) { $env:AWS_ACCESS_KEY_ID = "stackyard" }
    if (-not $env:AWS_SECRET_ACCESS_KEY) { $env:AWS_SECRET_ACCESS_KEY = "stackyard" }
    if (-not $env:AWS_REGION) { $env:AWS_REGION = "us-east-1" }
    $env:AWS_S3_FORCE_PATH_STYLE = "true"

    $bucketName = "demo-aws-" + [Guid]::NewGuid().ToString("N").Substring(0, 8)
    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api create-bucket --bucket $bucketName" }
    aws --endpoint-url $baseUrl s3api create-bucket --bucket $bucketName | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI create-bucket failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api list-objects-v2 --bucket $bucketName" }
    aws --endpoint-url $baseUrl s3api list-objects-v2 --bucket $bucketName | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI list-objects-v2 failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api put-object --bucket $bucketName --key notes/hello.txt --body ./README.md" }
    aws --endpoint-url $baseUrl s3api put-object --bucket $bucketName --key notes/hello.txt --body ./README.md | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI put-object failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api get-object --bucket $bucketName --key notes/hello.txt $env:TEMP\\stackyard-hello.txt" }
    aws --endpoint-url $baseUrl s3api get-object --bucket $bucketName --key notes/hello.txt "$env:TEMP\\stackyard-hello.txt" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI get-object failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api head-bucket --bucket $bucketName" }
    aws --endpoint-url $baseUrl s3api head-bucket --bucket $bucketName | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI head-bucket failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api head-object --bucket $bucketName --key notes/hello.txt" }
    aws --endpoint-url $baseUrl s3api head-object --bucket $bucketName --key notes/hello.txt | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI head-object failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api copy-object --copy-source $bucketName/notes/hello.txt --bucket $bucketName --key notes/hello-copy.txt" }
    aws --endpoint-url $baseUrl s3api copy-object --copy-source "$bucketName/notes/hello.txt" --bucket $bucketName --key notes/hello-copy.txt | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI copy-object failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3 presign s3://$bucketName/notes/hello.txt --expires-in 300" }
    $presigned = aws --endpoint-url $baseUrl s3 presign "s3://$bucketName/notes/hello.txt" --expires-in 300
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI presign failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ curl.exe -H `"Range: bytes=0-9`" $presigned" }
    $rangeStatus = [int](curl.exe -sS -o NUL -w "%{http_code}" -H "Range: bytes=0-9" $presigned)
    if ($rangeStatus -ne 206) { throw "Unexpected range status: $rangeStatus" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api delete-object --bucket $bucketName --key notes/hello-copy.txt" }
    aws --endpoint-url $baseUrl s3api delete-object --bucket $bucketName --key notes/hello-copy.txt | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI delete-object failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api delete-objects --bucket $bucketName --delete file://./scripts/delete-objects.json" }
    $deletePayload = @"
{
  "Objects": [
    { "Key": "notes/hello.txt" }
  ],
  "Quiet": true
}
"@
    $deletePath = Join-Path $env:TEMP "stackyard-delete-objects.json"
    $deletePayload | Set-Content -Path $deletePath -Encoding ASCII
    aws --endpoint-url $baseUrl s3api delete-objects --bucket $bucketName --delete "file://$deletePath" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI delete-objects failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api create-multipart-upload --bucket $bucketName --key notes/multipart.txt" }
    $uploadId = aws --endpoint-url $baseUrl s3api create-multipart-upload --bucket $bucketName --key notes/multipart.txt --query UploadId --output text
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI create-multipart-upload failed (exit $LASTEXITCODE)" }

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api upload-part --bucket $bucketName --key notes/multipart.txt --part-number 1 --upload-id $uploadId --body ./README.md" }
    $partEtag = aws --endpoint-url $baseUrl s3api upload-part --bucket $bucketName --key notes/multipart.txt --part-number 1 --upload-id $uploadId --body ./README.md --query ETag --output text
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI upload-part failed (exit $LASTEXITCODE)" }

    $completePayload = @"
{
  "Parts": [
    { "ETag": $partEtag, "PartNumber": 1 }
  ]
}
"@
    $completePath = Join-Path $env:TEMP "stackyard-complete-mpu.json"
    $completePayload | Set-Content -Path $completePath -Encoding ASCII

    if ($Verbose) { Write-Host "+ aws --endpoint-url $baseUrl s3api complete-multipart-upload --bucket $bucketName --key notes/multipart.txt --upload-id $uploadId --multipart-upload file://$completePath" }
    aws --endpoint-url $baseUrl s3api complete-multipart-upload --bucket $bucketName --key notes/multipart.txt --upload-id $uploadId --multipart-upload "file://$completePath" | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "AWS CLI complete-multipart-upload failed (exit $LASTEXITCODE)" }
  }
}

Write-Host "Smoke tests passed."
$success = $true

if ($Reset -and $success) {
  if ($Verbose) {
    Write-Host "+ docker compose down"
  }
  docker compose down --remove-orphans | Out-Null
}
