-- P2-6A isolated read-only inventory batch/expiry snapshot.
-- The caller must select the target database. This migration refuses the
-- production BHIS and source HospDataV9 databases by design.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

IF DB_NAME() IN (N'BHIS', N'HospDataV9')
    THROW 51026, 'P2-6A migration is restricted to an isolated test database', 1;
GO

IF OBJECT_ID(N'dbo.inventory_batch', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.inventory_batch (
        batch_id             varchar(64)    NOT NULL,
        sku_id               varchar(64)    NOT NULL,
        item_no              varchar(64)    NOT NULL,
        item_name            nvarchar(128)  NOT NULL,
        storage_unit_id      varchar(64)    NULL,
        batch_no             nvarchar(64)   NULL,
        product_no           nvarchar(128)  NULL,
        supplier_id          varchar(64)    NULL,
        product_date         date           NULL,
        expire_date          date           NULL,
        qty_on_hand          decimal(18,3)  NOT NULL CONSTRAINT df_inventory_batch_qty DEFAULT 0,
        units                nvarchar(32)   NOT NULL,
        source_ref           varchar(128)   NULL,
        row_version          varchar(64)    NOT NULL CONSTRAINT df_inventory_batch_version DEFAULT 'fixture-1',
        is_deleted           bit            NOT NULL CONSTRAINT df_inventory_batch_deleted DEFAULT 0,
        normalized_storage_unit_id varchar(64) NOT NULL CONSTRAINT df_inventory_batch_storage_key DEFAULT '',
        normalized_batch_no  nvarchar(64)   NOT NULL CONSTRAINT df_inventory_batch_no_key DEFAULT '',
        normalized_source_ref varchar(128)  NOT NULL CONSTRAINT df_inventory_batch_source_key DEFAULT '',
        CONSTRAINT pk_inventory_batch PRIMARY KEY (batch_id),
        CONSTRAINT ck_inventory_batch_qty CHECK (qty_on_hand >= 0)
    );

    CREATE UNIQUE INDEX ux_inventory_batch_logical_key
        ON dbo.inventory_batch(normalized_storage_unit_id, sku_id, normalized_batch_no, normalized_source_ref)
        WHERE is_deleted = 0;

    CREATE INDEX ix_inventory_batch_item_status
        ON dbo.inventory_batch(item_no, expire_date, is_deleted);

    CREATE INDEX ix_inventory_batch_storage
        ON dbo.inventory_batch(storage_unit_id, batch_id)
        WHERE is_deleted = 0;
END
GO
IF NOT EXISTS (SELECT 1 FROM dbo.access_permission WHERE permission_code = N'inventory:batch-read')
BEGIN
    INSERT INTO dbo.access_permission (
        permission_code, permission_name, description, module_code, category_id,
        mode_code, second_permission_code, actions, status, hidden
    ) VALUES (
        N'inventory:batch-read', N'库存批次/效期查询', N'只读批次与效期快照', N'master', N'inventory',
        N'password', NULL, N'S', N'ACTIVE', 0
    );
END
GO

IF NOT EXISTS (
    SELECT 1 FROM dbo.access_role_permission
    WHERE role_code = N'PHARMACIST' AND permission_code = N'inventory:batch-read'
)
BEGIN
    INSERT INTO dbo.access_role_permission(role_code, permission_code)
    VALUES (N'PHARMACIST', N'inventory:batch-read');
END
GO
