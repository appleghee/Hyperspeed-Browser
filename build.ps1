# Ultra-Browser v4.0.0-ultra Build Script
param([string]$Mode = "release")

$ErrorActionPreference = "Stop"
$env:CGO_ENABLED = "1"
$env:CC = "C:\mingw64\bin\gcc.exe"
$env:PATH = "C:\mingw64\bin;$env:PATH"

if ($Mode -eq "dev") {
    $out = "ultra-browser-dev.exe"
    Write-Host "[build] Building $out (mode=dev)" -ForegroundColor Cyan
    go vet ./...
    go build -o $out .
} else {
    $out = "ultra-browser.exe"
    Write-Host "[build] Building $out (mode=release)" -ForegroundColor Cyan
    go vet ./...
    go build -ldflags="-s -w -H windowsgui" -o $out .
}
Write-Host "[build] OK - $out" -ForegroundColor Green
