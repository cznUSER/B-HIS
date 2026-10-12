# Build the Vue login UI and a Windows amd64 launcher exe.
# Run from anywhere:  powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1
# Output: bin\b-his-host.exe
# Double-click that exe. Demo account: admin / Admin@123 (memory API, not Mock).

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location (Join-Path $Root "web")

npm ci
npm run build:demo

$Assets = Join-Path $Root "internal\webui\assets"
if (Test-Path $Assets) {
    Remove-Item -Recurse -Force $Assets
}
New-Item -ItemType Directory -Force -Path $Assets | Out-Null
Copy-Item -Recurse -Force (Join-Path $Root "web\dist\*") $Assets

Set-Location $Root
New-Item -ItemType Directory -Force -Path (Join-Path $Root "bin") | Out-Null
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o (Join-Path $Root "bin\b-his-host.exe") ./cmd/host

Write-Host "Built bin\b-his-host.exe"
Write-Host "Double-click it. Browser opens http://127.0.0.1:8080/login"
Write-Host "Demo account: admin / Admin@123"
Write-Host "Optional: put config.local.yaml next to the exe (do not commit secrets)."
