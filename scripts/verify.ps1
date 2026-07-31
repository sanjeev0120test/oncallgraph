#!/usr/bin/env pwsh
# Fast local validation for Windows - keeps your laptop responsive.
# Heavy checks (race, 3-OS matrix, coverage) run in GitHub Actions instead.
$ErrorActionPreference = "Stop"
$env:CGO_ENABLED = "0"

Set-Location (Split-Path $PSScriptRoot -Parent)

Write-Host "==> gofmt" -ForegroundColor Cyan
$unformatted = (gofmt -l .)
if ($unformatted) { Write-Host "Unformatted files:`n$unformatted" -ForegroundColor Red; exit 1 }

Write-Host "==> go vet" -ForegroundColor Cyan
go vet ./...
if ($LASTEXITCODE -ne 0) { exit 1 }

Write-Host "==> go build" -ForegroundColor Cyan
go build -o bin/opsgraph.exe ./cmd/opsgraph
if ($LASTEXITCODE -ne 0) { exit 1 }

Write-Host "==> go test (no race)" -ForegroundColor Cyan
go test ./...
if ($LASTEXITCODE -ne 0) { exit 1 }

if (Test-Path ./fixtures/incident_checkout) {
    Write-Host "==> fixture/golden test" -ForegroundColor Cyan
    & ./bin/opsgraph.exe test ./fixtures/incident_checkout
    if ($LASTEXITCODE -ne 0) { exit 1 }

    Write-Host "==> demo" -ForegroundColor Cyan
    & ./bin/opsgraph.exe demo --format json | Out-Null
    if ($LASTEXITCODE -ne 0) { exit 1 }
}

Write-Host "OK - local validation passed" -ForegroundColor Green
