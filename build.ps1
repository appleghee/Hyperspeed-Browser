param(
    [ValidateSet("debug","release","dev")]
    [string]$Mode = "release"
)

$env:CGO_ENABLED = "1"
$env:CC = "C:\mingw64\bin\gcc.exe"
$env:PATH = "C:\mingw64\bin;$env:PATH"

$ldflags = "-s -w -H=windowsgui"
$out = "hyperspeed-browser.exe"

if ($Mode -eq "debug") {
    $ldflags = "-H=windowsgui"
    Write-Host "[build] DEBUG mode" -ForegroundColor Yellow
}
if ($Mode -eq "dev") {
    $ldflags = ""
    $out = "hyperspeed-browser-dev.exe"
}

Write-Host "[build] Building $out (CGO=1, CC=gcc, mode=$Mode)" -ForegroundColor Cyan

go vet .
if ($LASTEXITCODE -ne 0) {
    Write-Host "[build] VET FAILED" -ForegroundColor Red
    exit 1
}
Write-Host "[build] go vet: OK" -ForegroundColor Green

go build "-ldflags=$ldflags" -trimpath -o $out
if ($LASTEXITCODE -eq 0) {
    $size = (Get-Item $out).Length / 1MB
    Write-Host "[build] OK - $out ($("{0:N1}" -f $size) MB)" -ForegroundColor Green
} else {
    Write-Host "[build] FAILED" -ForegroundColor Red
    exit 1
}