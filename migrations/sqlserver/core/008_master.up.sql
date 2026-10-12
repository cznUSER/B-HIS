-- BHIS master-data maintenance (P1-H03-MASTER · 拍板#24 + H02 基础数据清单 + H09 药品目录与库存口径).
-- Store-swappable: SQL Server Phase1.
-- Apply with sqlcmd; this file has NO connection string, password, or server name.
--   sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/008_master.up.sql
-- PK style: varchar(64). Soft-delete + filtered unique indexes.
-- Depends on: 001_identity (BHIS + app_user), 003_register (department/doctor),
--             004_order (catalog_item), 006_inventory (inventory_item/ledger).
-- Does not CREATE DATABASE. Does not recreate app_user (see 001_identity / 009_seed notes).
-- H09：期初/盘盈/盘亏均走 inventory_ledger.biz_type=ADJUST（不新造 INIT）；无独立 adjust 单头表。

SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

-- ---------------------------------------------------------------------------
-- dbo.department：补 sort_no（#24c / H02 排序）
-- ---------------------------------------------------------------------------
IF COL_LENGTH(N'dbo.department', N'sort_no') IS NULL
BEGIN
    ALTER TABLE dbo.department
        ADD sort_no int NOT NULL
            CONSTRAINT df_department_sort_no DEFAULT 0;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.catalog_item：毒麻钩子列（#24e / H09 §1.1；默认关，本版不拦）
-- ---------------------------------------------------------------------------
IF COL_LENGTH(N'dbo.catalog_item', N'is_controlled') IS NULL
BEGIN
    ALTER TABLE dbo.catalog_item
        ADD is_controlled bit NOT NULL
            CONSTRAINT df_catalog_item_is_controlled DEFAULT 0;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.dict_item：轻量可维护码表（#24g；本期至少号别 register_sort）
-- 性别/证件/支付方式等仍为契约只读枚举，不入本表。
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.dict_item', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.dict_item (
        dict_item_id  varchar(64)    NOT NULL,
        dict_type     varchar(32)    NOT NULL,   -- 如 register_sort
        code          varchar(32)    NOT NULL,   -- 如 sort-normal
        name          nvarchar(64)   NOT NULL,
        status        varchar(16)    NOT NULL CONSTRAINT df_dict_item_status DEFAULT 'ACTIVE',
        sort_no       int            NOT NULL CONSTRAINT df_dict_item_sort_no DEFAULT 0,
        created_at    datetime2      NOT NULL CONSTRAINT df_dict_item_created DEFAULT SYSUTCDATETIME(),
        created_by    varchar(64)    NULL,
        updated_at    datetime2      NOT NULL CONSTRAINT df_dict_item_updated DEFAULT SYSUTCDATETIME(),
        updated_by    varchar(64)    NULL,
        is_deleted    bit            NOT NULL CONSTRAINT df_dict_item_deleted DEFAULT 0,
        CONSTRAINT pk_dict_item PRIMARY KEY (dict_item_id),
        CONSTRAINT ck_dict_item_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
    );

    CREATE UNIQUE INDEX ux_dict_item_type_code
        ON dbo.dict_item(dict_type, code)
        WHERE is_deleted = 0;

    CREATE INDEX ix_dict_item_type_status
        ON dbo.dict_item(dict_type, status)
        WHERE is_deleted = 0;
END
GO

-- ---------------------------------------------------------------------------
-- Seed：号别 register_sort / sort-normal（#24g / #24i 可维护）
-- ---------------------------------------------------------------------------
IF NOT EXISTS (
    SELECT 1 FROM dbo.dict_item
    WHERE dict_type = N'register_sort' AND code = N'sort-normal' AND is_deleted = 0
)
BEGIN
    INSERT INTO dbo.dict_item (
        dict_item_id, dict_type, code, name, status, sort_no, created_by, updated_by
    ) VALUES (
        N'dict-sort-normal',
        N'register_sort',
        N'sort-normal',
        N'普通号',
        N'ACTIVE',
        0,
        N'system',
        N'system'
    );
END
GO

-- ---------------------------------------------------------------------------
-- Notes（非 DDL）：
-- * inventory 调整：复用 006 inventory_ledger（biz_type=ADJUST）；同事务改 qty_on_hand；
--   调后 ≥0；remark 区分期初/盘盈/盘亏；不新造 INIT；不建 inventory_adjust 单头。
-- * 新建 DRUG catalog：应用层同事务或紧随建 inventory_item（qty_on_hand=0）；TREAT 不建。
-- * app_user / role：依赖实现仓 001_identity；轻量用户维护 API 读写既有列；
--   演示口令策略见 009_seed_auth_users.sql（demo default passphrase per #15/#16；禁止明文生产秘密）。
-- * 种子 dept-1/2、doc-1/2、DRUG-A/B、TREAT-1 须可被维护 API 修改（#24i）。
-- ---------------------------------------------------------------------------
