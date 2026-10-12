-- BHIS catalog + order sheet/items (P1-H03-ORDER). Store-swappable: SQL Server Phase1.
-- Apply with sqlcmd; this file has no server name and no credentials.
--   sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/004_order.up.sql
-- PK style: varchar(64) (aligned with register_bill.register_id / patient.patient_id).
-- Depends on: 001_identity (database BHIS), 002_patient, 003_register. Does not CREATE DATABASE.
-- 拍板#21：强制有效 registerId（非 BACK）；save+commit；多处方；诊断不强制；毒麻钩子关；审核一期跳过。
-- 域内 encounter_id ≡ register_id（列名用 register_id）。

SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

-- ---------------------------------------------------------------------------
-- dbo.catalog_item（药品/治疗目录；一期种子 DRUG-A/B、TREAT-1）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.catalog_item', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.catalog_item (
        catalog_item_id  varchar(64)    NOT NULL,
        item_no          varchar(32)    NOT NULL,
        item_name        nvarchar(128)  NOT NULL,
        item_type        varchar(16)    NOT NULL,
        spec             nvarchar(64)   NULL,
        units            nvarchar(16)   NOT NULL,
        price            decimal(18,2)  NOT NULL,
        status           varchar(16)    NOT NULL CONSTRAINT df_catalog_item_status DEFAULT 'ACTIVE',
        created_at       datetime2      NOT NULL CONSTRAINT df_catalog_item_created DEFAULT SYSUTCDATETIME(),
        created_by       varchar(64)    NULL,
        updated_at       datetime2      NOT NULL CONSTRAINT df_catalog_item_updated DEFAULT SYSUTCDATETIME(),
        updated_by       varchar(64)    NULL,
        is_deleted       bit            NOT NULL CONSTRAINT df_catalog_item_deleted DEFAULT 0,
        CONSTRAINT pk_catalog_item PRIMARY KEY (catalog_item_id),
        CONSTRAINT ck_catalog_item_type CHECK (item_type IN ('DRUG', 'TREAT')),
        CONSTRAINT ck_catalog_item_status CHECK (status IN ('ACTIVE', 'INACTIVE')),
        CONSTRAINT ck_catalog_item_price CHECK (price >= 0)
    );

    CREATE UNIQUE INDEX ux_catalog_item_no
        ON dbo.catalog_item(item_no)
        WHERE is_deleted = 0;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.order_sheet（处方头；一张挂号允许多张；encounter ≡ register_id）
-- 一期状态四态：SAVE / COMMIT / PROC / ABATE（RECIPE_CHECK 等延后，不入 CHECK）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.order_sheet', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.order_sheet (
        order_id           varchar(64)    NOT NULL,
        order_no           varchar(32)    NOT NULL,
        register_id        varchar(64)    NOT NULL,  -- ≡ encounter_id；FK→register_bill
        patient_id         varchar(64)    NOT NULL,  -- 可由挂号带出，仍落库
        department_id      varchar(64)    NOT NULL,
        doctor_id          varchar(64)    NOT NULL,  -- 默认=挂号医生，允许改（#21b）
        diagnose_name      nvarchar(128)  NULL,      -- 不强制（#21c）
        recipe_sort        varchar(32)    NULL CONSTRAINT df_order_recipe_sort DEFAULT 'NORMAL',
        status             varchar(16)    NOT NULL CONSTRAINT df_order_status DEFAULT 'SAVE',
        total_fee          decimal(18,2)  NOT NULL CONSTRAINT df_order_total_fee DEFAULT 0,
        charged            bit            NOT NULL CONSTRAINT df_order_charged DEFAULT 0,
        submitted_at       datetime2      NULL,
        abated_at          datetime2      NULL,
        abated_by          varchar(64)    NULL,
        created_at         datetime2      NOT NULL CONSTRAINT df_order_created DEFAULT SYSUTCDATETIME(),
        created_by         varchar(64)    NULL,
        updated_at         datetime2      NOT NULL CONSTRAINT df_order_updated DEFAULT SYSUTCDATETIME(),
        updated_by         varchar(64)    NULL,
        is_deleted         bit            NOT NULL CONSTRAINT df_order_deleted DEFAULT 0,
        CONSTRAINT pk_order_sheet PRIMARY KEY (order_id),
        CONSTRAINT fk_order_register FOREIGN KEY (register_id) REFERENCES dbo.register_bill(register_id),
        CONSTRAINT fk_order_patient FOREIGN KEY (patient_id) REFERENCES dbo.patient(patient_id),
        CONSTRAINT fk_order_department FOREIGN KEY (department_id) REFERENCES dbo.department(department_id),
        CONSTRAINT fk_order_doctor FOREIGN KEY (doctor_id) REFERENCES dbo.doctor(doctor_id),
        CONSTRAINT ck_order_status CHECK (status IN ('SAVE', 'COMMIT', 'PROC', 'ABATE')),
        CONSTRAINT ck_order_total_fee CHECK (total_fee >= 0)
    );

    CREATE UNIQUE INDEX ux_order_no
        ON dbo.order_sheet(order_no)
        WHERE is_deleted = 0;

    CREATE INDEX ix_order_register
        ON dbo.order_sheet(register_id)
        WHERE is_deleted = 0;

    CREATE INDEX ix_order_status
        ON dbo.order_sheet(status)
        WHERE is_deleted = 0;

    CREATE INDEX ix_order_patient
        ON dbo.order_sheet(patient_id)
        WHERE is_deleted = 0;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.order_item（处方明细；UK (order_id, line_no)；amount>0）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.order_item', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.order_item (
        order_item_id        varchar(64)    NOT NULL,
        order_id             varchar(64)    NOT NULL,
        line_no              int            NOT NULL,
        item_no              varchar(32)    NOT NULL,
        item_type            varchar(16)    NOT NULL,
        item_name            nvarchar(128)  NOT NULL,
        spec                 nvarchar(64)   NULL,
        units                nvarchar(16)   NOT NULL,
        amount               decimal(18,3)  NOT NULL,
        price                decimal(18,2)  NOT NULL,
        total_fee            decimal(18,2)  NOT NULL,
        usage_text           nvarchar(64)   NULL,
        frequency_text       nvarchar(64)   NULL,
        dosage               nvarchar(64)   NULL,
        dosage_unit          nvarchar(16)   NULL,
        proc_department_id   varchar(64)    NULL,
        created_at           datetime2      NOT NULL CONSTRAINT df_order_item_created DEFAULT SYSUTCDATETIME(),
        updated_at           datetime2      NOT NULL CONSTRAINT df_order_item_updated DEFAULT SYSUTCDATETIME(),
        CONSTRAINT pk_order_item PRIMARY KEY (order_item_id),
        CONSTRAINT fk_order_item_sheet FOREIGN KEY (order_id) REFERENCES dbo.order_sheet(order_id),
        CONSTRAINT ck_order_item_type CHECK (item_type IN ('DRUG', 'TREAT')),
        CONSTRAINT ck_order_item_amount CHECK (amount > 0),
        CONSTRAINT ck_order_item_price CHECK (price >= 0),
        CONSTRAINT ck_order_item_total_fee CHECK (total_fee >= 0)
    );

    CREATE UNIQUE INDEX ux_order_item_line
        ON dbo.order_item(order_id, line_no);

    CREATE INDEX ix_order_item_order
        ON dbo.order_item(order_id);
END
GO
