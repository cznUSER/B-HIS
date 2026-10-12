# Apply migrations/sqlserver/core/001_identity.up.sql with sqlcmd.
# Preferred Windows entrypoint. Run from anywhere:
#   $env:SQLCMDPASSWORD = '<sql-login-password>'
#   powershell -ExecutionPolicy Bypass -File scripts\apply-migration.ps1
#   powershell -ExecutionPolicy Bypass -File scripts\apply-migration.ps1 -Server localhost -User sa
#
# Defaults: -Server localhost, -User sa. Override both when the instance differs.
# The SQL login password is read only from the SQLCMDPASSWORD environment variable.
# It is not a parameter, it is not hardcoded, and this script never prints it.
# sqlcmd -I sets QUOTED_IDENTIFIER ON. The filtered unique index
# ux_app_user_username needs that option for INSERT and UPDATE.
# -f 65001 is UTF-8. The script starts in master and creates or uses BHIS.
# 2026-10-12

[CmdletBinding()]
param(
    [string]$Server = "localhost",
    [string]$User = "sa"
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($env:SQLCMDPASSWORD)) {
    Write-Error "SQLCMDPASSWORD is not set. Set that environment variable before running this script. The password is not accepted as a parameter and is never printed."
    exit 1
}

if (-not (Get-Command sqlcmd -ErrorAction SilentlyContinue)) {
    Write-Error "sqlcmd was not found on PATH."
    exit 1
}

$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$Migration = Join-Path "migrations" (Join-Path "sqlserver" (Join-Path "core" "001_identity.up.sql"))
if (-not (Test-Path -LiteralPath $Migration)) {
    Write-Error "Migration file not found: $Migration"
    exit 1
}

Write-Host "Applying $Migration to server $Server as $User (database master, then BHIS)."
& sqlcmd -S $Server -U $User -I -f 65001 -d master -b -i $Migration
if ($LASTEXITCODE -ne 0) {
    Write-Error "sqlcmd failed with exit code $LASTEXITCODE"
    exit $LASTEXITCODE
}

Write-Host "Identity migration applied."
