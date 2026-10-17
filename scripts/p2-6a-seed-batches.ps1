[CmdletBinding()]
param(
    [string]$Server = "localhost",
    [string]$Database = "BHIS_PHASE2_TEST_20261015"
)

$ErrorActionPreference = "Stop"
if ($Database -ne "BHIS_PHASE2_TEST_20261015") {
    throw "P2-6A fixture script only permits BHIS_PHASE2_TEST_20261015; refusing '$Database'."
}
if (-not (Get-Command sqlcmd -ErrorAction SilentlyContinue)) {
    throw "sqlcmd was not found on PATH."
}

$root = Split-Path -Parent $PSScriptRoot
$migration = Join-Path $root "migrations\sqlserver\core\012_inventory_batch_readonly.up.sql"
if (-not (Test-Path -LiteralPath $migration)) { throw "Migration file not found: $migration" }

& sqlcmd -S $Server -E -d $Database -I -f 65001 -b -i $migration
if ($LASTEXITCODE -ne 0) { throw "migration failed with exit code $LASTEXITCODE" }

$fixture = @"
DELETE FROM dbo.inventory_batch WHERE batch_id LIKE 'P2-6A-%';
INSERT INTO dbo.inventory_batch (
    batch_id, sku_id, item_no, item_name, storage_unit_id, batch_no,
    product_date, expire_date, qty_on_hand, units, source_ref,
    normalized_storage_unit_id, normalized_batch_no, normalized_source_ref, row_version
) VALUES
('P2-6A-EXPIRED','P2-6A-SKU','P2-6A-DRUG',N'合成测试药品','fixture-pharmacy','B-EXPIRED','2026-01-01','2026-10-14',3.000,N'盒','FIXTURE-P2-6A','fixture-pharmacy','B-EXPIRED','FIXTURE-P2-6A','fixture-1'),
('P2-6A-EXPIRING-ASOF','P2-6A-SKU','P2-6A-DRUG',N'合成测试药品','fixture-pharmacy','B-EXPIRING-ASOF','2026-01-01','2026-10-15',1.000,N'盒','FIXTURE-P2-6A','fixture-pharmacy','B-EXPIRING-ASOF','FIXTURE-P2-6A-2','fixture-1'),
('P2-6A-EXPIRING-END','P2-6A-SKU','P2-6A-DRUG',N'合成测试药品','fixture-pharmacy','B-EXPIRING-END','2026-01-01','2026-11-14',4.000,N'盒','FIXTURE-P2-6A','fixture-pharmacy','B-EXPIRING-END','FIXTURE-P2-6A-3','fixture-1'),
('P2-6A-VALID','P2-6A-SKU','P2-6A-DRUG',N'合成测试药品','fixture-pharmacy','B-VALID','2026-01-01','2026-11-15',10.000,N'盒','FIXTURE-P2-6A','fixture-pharmacy','B-VALID','FIXTURE-P2-6A-4','fixture-1'),
('P2-6A-UNKNOWN','P2-6A-SKU','P2-6A-DRUG',N'合成测试药品','fixture-pharmacy','B-UNKNOWN','2026-01-01',NULL,7.000,N'盒','FIXTURE-P2-6A','fixture-pharmacy','B-UNKNOWN','FIXTURE-P2-6A-5','fixture-1');
"@
$fixturePath = Join-Path $env:TEMP "p2-6a-seed-$PID.sql"
Set-Content -LiteralPath $fixturePath -Value $fixture -Encoding utf8
try {
    & sqlcmd -S $Server -E -d $Database -I -f 65001 -b -i $fixturePath
    if ($LASTEXITCODE -ne 0) { throw "fixture insert failed with exit code $LASTEXITCODE" }
}
finally {
    Remove-Item -LiteralPath $fixturePath -Force -ErrorAction SilentlyContinue
}

Write-Host "P2-6A fixture seeded in ${Database}: 5 synthetic rows with prefix P2-6A-."
