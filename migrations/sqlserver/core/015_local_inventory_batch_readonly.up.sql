-- H01: final local development target; keep 012's isolated-only guard intact.
-- No batch rows, stock changes, or source-database writes are performed.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
SET XACT_ABORT ON;
SET NOCOUNT ON;

IF DB_NAME() <> N'BHIS'
    THROW 51027, '015 is restricted to the local BHIS development database', 1;

IF OBJECT_ID(N'dbo.access_permission', N'U') IS NULL
   OR OBJECT_ID(N'dbo.access_role_permission', N'U') IS NULL
   OR NOT EXISTS (SELECT 1 FROM dbo.access_role WHERE role_code = N'PHARMACIST')
    THROW 51027, '015 requires the existing access-control schema and PHARMACIST role', 1;

BEGIN TRANSACTION;
DECLARE @created_table bit = 0, @created_permission bit = 0, @created_grant bit = 0;
IF OBJECT_ID(N'dbo.inventory_batch', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.inventory_batch (
        batch_id varchar(64) NOT NULL,
        sku_id varchar(64) NOT NULL,
        item_no varchar(64) NOT NULL,
        item_name nvarchar(128) NOT NULL,
        storage_unit_id varchar(64) NULL,
        batch_no nvarchar(64) NULL,
        product_no nvarchar(128) NULL,
        supplier_id varchar(64) NULL,
        product_date date NULL,
        expire_date date NULL,
        qty_on_hand decimal(18,3) NOT NULL CONSTRAINT df_inventory_batch_qty DEFAULT 0,
        units nvarchar(32) NOT NULL,
        source_ref varchar(128) NULL,
        row_version varchar(64) NOT NULL CONSTRAINT df_inventory_batch_version DEFAULT 'snapshot-1',
        is_deleted bit NOT NULL CONSTRAINT df_inventory_batch_deleted DEFAULT 0,
        normalized_storage_unit_id varchar(64) NOT NULL CONSTRAINT df_inventory_batch_storage_key DEFAULT '',
        normalized_batch_no nvarchar(64) NOT NULL CONSTRAINT df_inventory_batch_no_key DEFAULT '',
        normalized_source_ref varchar(128) NOT NULL CONSTRAINT df_inventory_batch_source_key DEFAULT '',
        CONSTRAINT pk_inventory_batch PRIMARY KEY (batch_id),
        CONSTRAINT ck_inventory_batch_qty CHECK (qty_on_hand >= 0)
    );
    CREATE UNIQUE INDEX ux_inventory_batch_logical_key
        ON dbo.inventory_batch(normalized_storage_unit_id, sku_id, normalized_batch_no, normalized_source_ref)
        WHERE is_deleted = 0;
    CREATE INDEX ix_inventory_batch_item_status ON dbo.inventory_batch(item_no, expire_date, is_deleted);
    CREATE INDEX ix_inventory_batch_storage ON dbo.inventory_batch(storage_unit_id, batch_id) WHERE is_deleted = 0;
    SET @created_table = 1;
END;

IF NOT EXISTS (SELECT 1 FROM dbo.access_permission WHERE permission_code = N'inventory:batch-read')
BEGIN
    INSERT INTO dbo.access_permission(permission_code, permission_name, description, module_code, category_id,
        mode_code, second_permission_code, actions, status, hidden)
    VALUES(N'inventory:batch-read', N'库存批次/效期查询', N'只读批次与效期快照', N'master', N'inventory',
        N'password', NULL, N'S', N'ACTIVE', 0);
    SET @created_permission = 1;
END;
IF NOT EXISTS (SELECT 1 FROM dbo.access_role_permission
    WHERE role_code = N'PHARMACIST' AND permission_code = N'inventory:batch-read')
BEGIN
    INSERT INTO dbo.access_role_permission(role_code, permission_code) VALUES(N'PHARMACIST', N'inventory:batch-read');
    SET @created_grant = 1;
END;

-- Ownership only belongs to a table created by this migration. A rerun must
-- preserve these flags, including when all resources now already exist.
IF @created_table = 1
BEGIN
    EXEC sys.sp_addextendedproperty @name=N'BHIS_015_CREATED_PERMISSION', @value=@created_permission,
        @level0type=N'SCHEMA', @level0name=N'dbo', @level1type=N'TABLE', @level1name=N'inventory_batch';
    EXEC sys.sp_addextendedproperty @name=N'BHIS_015_CREATED_GRANT', @value=@created_grant,
        @level0type=N'SCHEMA', @level0name=N'dbo', @level1type=N'TABLE', @level1name=N'inventory_batch';
END;
COMMIT TRANSACTION;
PRINT 'LOCAL_BHIS_BATCH_SCHEMA_READY';
