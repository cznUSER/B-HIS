-- Refuse data loss or removal of new permission dependencies.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
SET XACT_ABORT ON;
SET NOCOUNT ON;

IF DB_NAME() <> N'BHIS'
    THROW 51027, '015 rollback is restricted to the local BHIS development database', 1;

IF OBJECT_ID(N'dbo.inventory_batch', N'U') IS NULL
BEGIN
    PRINT 'LOCAL_BHIS_BATCH_ROLLBACK_NOOP';
    RETURN;
END;
DECLARE @owned_permission bit, @owned_grant bit;
SELECT @owned_permission=CONVERT(bit, value) FROM sys.extended_properties
WHERE major_id=OBJECT_ID(N'dbo.inventory_batch') AND minor_id=0 AND name=N'BHIS_015_CREATED_PERMISSION';
SELECT @owned_grant=CONVERT(bit, value) FROM sys.extended_properties
WHERE major_id=OBJECT_ID(N'dbo.inventory_batch') AND minor_id=0 AND name=N'BHIS_015_CREATED_GRANT';
IF @owned_permission IS NULL OR @owned_grant IS NULL
    THROW 51027, '015 does not own this existing table; review a manual rollback', 1;

BEGIN TRANSACTION;
IF EXISTS (SELECT 1 FROM dbo.inventory_batch WITH (TABLOCKX, HOLDLOCK))
    THROW 51027, '015 rollback refuses a nonempty inventory_batch table', 1;
IF @owned_permission=1 AND (
    EXISTS (SELECT 1 FROM dbo.user_permission_override WHERE permission_code=N'inventory:batch-read')
    OR EXISTS (SELECT 1 FROM dbo.access_role_permission
        WHERE permission_code=N'inventory:batch-read' AND (role_code<>N'PHARMACIST' OR @owned_grant=0))
    OR EXISTS (SELECT 1 FROM dbo.access_permission WHERE second_permission_code=N'inventory:batch-read')
)
    THROW 51027, '015 rollback refuses new permission dependencies', 1;
IF @owned_grant=1
    DELETE FROM dbo.access_role_permission WHERE role_code=N'PHARMACIST' AND permission_code=N'inventory:batch-read';
IF @owned_permission=1
    DELETE FROM dbo.access_permission WHERE permission_code=N'inventory:batch-read';
DROP TABLE dbo.inventory_batch;
COMMIT TRANSACTION;
PRINT 'LOCAL_BHIS_BATCH_SCHEMA_ROLLED_BACK';
