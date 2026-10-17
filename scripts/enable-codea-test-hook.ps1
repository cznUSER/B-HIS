$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
git config core.hooksPath .githooks
Write-Host 'Enabled .githooks/post-commit for Code A test startup.'
