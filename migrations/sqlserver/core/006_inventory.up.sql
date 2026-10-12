-- BHIS inventory_item / inventory_ledger (P1-H03-DISPENSE · 方案 A 简易数量账). Store-swappable: SQL Server Phase1.
-- Apply with sqlcmd; this file has NO connection string, password, or server name.
--   sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/006_inventory.up.sql
-- PK style: varchar(64) (aligned with catalog_item / order_sheet).
-- Depends on: 001_identity (database BHIS), 004_order (catalog DRUG-* 对齐 item_no). Does not CREATE DATABASE.
-- 拍板#2/#23：简易总量账；发药扣减/退药回加；无批次效期；TREAT 不入库存表。

SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

-- ---------------------------------------------------------------------------
-- dbo.inventory_item（一药品一 SKU；item_no 对齐 catalog_item.item_no，如 DRUG-A/B）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.inventory_item', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.inventory_item (
        sku_id        varchar(64)    NOT NULL,
        item_no       varchar(32)    NOT NULL,   -- 对齐 catalog DRUG-A / DRUG-B（UK）
        item_name     nvarchar(128)  NOT NULL,
        qty_on_hand   decimal(18,3)  NOT NULL CONSTRAINT df_inv_item_qty DEFAULT 0,
        units         nvarchar(16)   NOT NULL,
        status        varchar(16)    NOT NULL CONSTRAINT df_inv_item_status DEFAULT 'ACTIVE',
        row_version   rowversion     NOT NULL,   -- 并发扣减
        created_at    datetime2      NOT NULL CONSTRAINT df_inv_item_created DEFAULT SYSUTCDATETIME(),
        created_by    varchar(64)    NULL,
        updated_at    datetime2      NOT NULL CONSTRAINT df_inv_item_updated DEFAULT SYSUTCDATETIME(),
        updated_by    varchar(64)    NULL,
        is_deleted    bit            NOT NULL CONSTRAINT df_inv_item_deleted DEFAULT 0,
        CONSTRAINT pk_inventory_item PRIMARY KEY (sku_id),
        CONSTRAINT ck_inv_item_qty CHECK (qty_on_hand >= 0),
        CONSTRAINT ck_inv_item_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
    );

    CREATE UNIQUE INDEX ux_inventory_item_no
        ON dbo.inventory_item(item_no)
        WHERE is_deleted = 0;

    CREATE INDEX ix_inventory_item_status
        ON dbo.inventory_item(status)
        WHERE is_deleted = 0;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.inventory_ledger（只追加；发药负 / 退药正 / 调整）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.inventory_ledger', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.inventory_ledger (
        ledger_id   varchar(64)    NOT NULL,
        sku_id      varchar(64)    NOT NULL,
        qty_delta   decimal(18,3)  NOT NULL,   -- DISPENSE 负；RETURN 正；ADJUST 可正负
        biz_type    varchar(16)    NOT NULL,   -- DISPENSE | RETURN | ADJUST
        biz_id      varchar(64)    NULL,       -- dispense_id 等
        remark      nvarchar(256)  NULL,
        created_at  datetime2      NOT NULL CONSTRAINT df_inv_ledger_created DEFAULT SYSUTCDATETIME(),
        created_by  varchar(64)    NULL,
        CONSTRAINT pk_inventory_ledger PRIMARY KEY (ledger_id),
        CONSTRAINT fk_inv_ledger_sku FOREIGN KEY (sku_id) REFERENCES dbo.inventory_item(sku_id),
        CONSTRAINT ck_inv_ledger_biz_type CHECK (biz_type IN ('DISPENSE', 'RETURN', 'ADJUST'))
    );

    CREATE INDEX ix_inv_ledger_sku_created
        ON dbo.inventory_ledger(sku_id, created_at);

    CREATE INDEX ix_inv_ledger_biz
        ON dbo.inventory_ledger(biz_type, biz_id)
        WHERE biz_id IS NOT NULL;
END
GO
