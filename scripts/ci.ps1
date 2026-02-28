$ErrorActionPreference = "Stop"

Write-Host "Running full test suite..."
go test ./...
