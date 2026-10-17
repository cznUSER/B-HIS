[CmdletBinding()]
param(
    [string]$Server = 'localhost',
    [string]$Database = 'HIS',
    [string]$OutputPath = (Join-Path $PSScriptRoot '..\deliverables\h01\his-history-import-exceptions-20261014.csv')
)

$ErrorActionPreference = 'Stop'
if ($Database -ne 'HIS') { throw "Refusing to read a non-HIS database: $Database" }
$sqlPath = Join-Path $PSScriptRoot 'export-his-import-exceptions-20261014.sql'
if (-not (Test-Path -LiteralPath $sqlPath)) { throw "Missing SQL export: $sqlPath" }
if (-not (Get-Command sqlcmd -ErrorAction SilentlyContinue)) { throw 'sqlcmd was not found on PATH.' }

# All three columns are bounded identifiers/reason codes. -W trims their padding;
# it cannot be combined with sqlcmd's -y/-Y variable-width option. A wide console
# row prevents an otherwise valid CSV record from wrapping into multiple lines.
$raw = @(& sqlcmd -S $Server -E -I -b -d $Database -i $sqlPath -h -1 -W -s ',' -w 65535 2>&1)
if ($LASTEXITCODE -ne 0) { throw "Exception export failed: $($raw -join ' ')" }
$rows = @($raw | ForEach-Object { ([string]$_).Trim() } | Where-Object {
    $_ -and $_ -notmatch '^\(\d+ rows? affected\)$' -and $_ -match '^[^,]+,[^,]+,[^,]+$'
})
$parent = Split-Path -Parent $OutputPath
New-Item -ItemType Directory -Path $parent -Force | Out-Null
@('source_table,source_id,reason') + $rows | Set-Content -LiteralPath $OutputPath -Encoding UTF8
Write-Output "EXCEPTIONS_OUTPUT=$OutputPath;ROWS=$($rows.Count)"
