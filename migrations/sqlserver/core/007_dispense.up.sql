-- BHIS dispense / dispense_item (P1-H03-DISPENSE). Store-swappable: SQL Server Phase1.
-- Apply with sqlcmd; this file has NO connection string, password, or server name.
--   sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/007_dispense.up.sql
-- PK style: varchar(64). Depends on: 002_patient, 003_register, 004_order, 005_charge, 006_inventory.
-- 拍板#23 + H02《发药字段清单》：仅已收费 DRUG；部分发；行态 NONE|PARTIAL|FULL；退药≤已发未退；毒麻不拦。
-- 每次 POST /dispenses 新建一张发药单（本批次 qty）；可发量=charged−Σ净发；退费前置看净发>0 → REFUND_REQUIRES_RETURN。

SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

-- ---------------------------------------------------------------------------
-- dbo.dispense（发药单头；一张可对应一次确认发药动作）
-- status：DONE=本单已记账；PARTIAL_RETURNED=部分退；RETURNED=本单净发为0
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.dispense', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.dispense (
        dispense_id    varchar(64)    NOT NULL,
        dispense_no    varchar(32)    NOT NULL,
        order_id       varchar(64)    NOT NULL,   -- FK→order_sheet
        charge_id      varchar(64)    NULL,       -- FK→charge；主路径有值（已支付 NORMAL）
        register_id    varchar(64)    NOT NULL,   -- FK→register_bill（由收费/处方带出）
        patient_id     varchar(64)    NOT NULL,   -- FK→patient
        status         varchar(24)    NOT NULL CONSTRAINT df_dispense_status DEFAULT 'DONE',
        dispensed_at   datetime2      NOT NULL CONSTRAINT df_dispense_at DEFAULT SYSUTCDATETIME(),
        operator_id    varchar(64)    NULL,
        dept_id        varchar(64)    NULL,       -- 执行科室（药房）；可选
        return_reason  nvarchar(256)  NULL,       -- 最近一次退药原因留痕（明细亦可）
        created_at     datetime2      NOT NULL CONSTRAINT df_dispense_created DEFAULT SYSUTCDATETIME(),
        created_by     varchar(64)    NULL,
        updated_at     datetime2      NOT NULL CONSTRAINT df_dispense_updated DEFAULT SYSUTCDATETIME(),
        updated_by     varchar(64)    NULL,
        is_deleted     bit            NOT NULL CONSTRAINT df_dispense_deleted DEFAULT 0,
        CONSTRAINT pk_dispense PRIMARY KEY (dispense_id),
        CONSTRAINT fk_dispense_order FOREIGN KEY (order_id) REFERENCES dbo.order_sheet(order_id),
        CONSTRAINT fk_dispense_charge FOREIGN KEY (charge_id) REFERENCES dbo.charge(charge_id),
        CONSTRAINT fk_dispense_register FOREIGN KEY (register_id) REFERENCES dbo.register_bill(register_id),
        CONSTRAINT fk_dispense_patient FOREIGN KEY (patient_id) REFERENCES dbo.patient(patient_id),
        CONSTRAINT ck_dispense_status CHECK (status IN ('DONE', 'PARTIAL_RETURNED', 'RETURNED'))
    );

    CREATE UNIQUE INDEX ux_dispense_no
        ON dbo.dispense(dispense_no)
        WHERE is_deleted = 0;

    CREATE INDEX ix_dispense_order
        ON dbo.dispense(order_id)
        WHERE is_deleted = 0;

    CREATE INDEX ix_dispense_charge
        ON dbo.dispense(charge_id)
        WHERE is_deleted = 0 AND charge_id IS NOT NULL;

    CREATE INDEX ix_dispense_patient_at
        ON dbo.dispense(patient_id, dispensed_at)
        WHERE is_deleted = 0;

    CREATE INDEX ix_dispense_register
        ON dbo.dispense(register_id)
        WHERE is_deleted = 0;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.dispense_item（本单行；qty 为本批次发药量；累计可发/行态由应用按 charge/order 聚合）
-- line_status：相对收费行净发 — NONE|PARTIAL|FULL（写入时按当时净发快照；查询以聚合为准）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.dispense_item', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.dispense_item (
        dispense_item_id  varchar(64)    NOT NULL,
        dispense_id       varchar(64)    NOT NULL,
        order_item_id     varchar(64)    NULL,       -- FK→order_item
        charge_item_id    varchar(64)    NULL,       -- FK→charge_item；主路径有值
        item_no           varchar(32)    NOT NULL,   -- 仅 DRUG
        item_name         nvarchar(128)  NOT NULL,
        units             nvarchar(16)   NULL,
        qty_charged       decimal(18,3)  NOT NULL,   -- 收费数量快照（charged_qty）
        qty_dispensed     decimal(18,3)  NOT NULL,   -- 本单发药数量（dispense_qty）
        qty_returned      decimal(18,3)  NOT NULL CONSTRAINT df_dispense_item_returned DEFAULT 0,
        line_status       varchar(16)    NOT NULL CONSTRAINT df_dispense_item_line DEFAULT 'PARTIAL',
        created_at        datetime2      NOT NULL CONSTRAINT df_dispense_item_created DEFAULT SYSUTCDATETIME(),
        updated_at        datetime2      NOT NULL CONSTRAINT df_dispense_item_updated DEFAULT SYSUTCDATETIME(),
        CONSTRAINT pk_dispense_item PRIMARY KEY (dispense_item_id),
        CONSTRAINT fk_dispense_item_dispense FOREIGN KEY (dispense_id) REFERENCES dbo.dispense(dispense_id),
        CONSTRAINT fk_dispense_item_order_item FOREIGN KEY (order_item_id) REFERENCES dbo.order_item(order_item_id),
        CONSTRAINT fk_dispense_item_charge_item FOREIGN KEY (charge_item_id) REFERENCES dbo.charge_item(charge_item_id),
        CONSTRAINT ck_dispense_item_qty_charged CHECK (qty_charged > 0),
        CONSTRAINT ck_dispense_item_qty_dispensed CHECK (qty_dispensed > 0),
        CONSTRAINT ck_dispense_item_qty_returned CHECK (qty_returned >= 0 AND qty_returned <= qty_dispensed),
        CONSTRAINT ck_dispense_item_qty_disp_vs_charged CHECK (qty_dispensed <= qty_charged),
        CONSTRAINT ck_dispense_item_line_status CHECK (line_status IN ('NONE', 'PARTIAL', 'FULL'))
    );

    CREATE INDEX ix_dispense_item_dispense
        ON dbo.dispense_item(dispense_id);

    CREATE INDEX ix_dispense_item_charge_item
        ON dbo.dispense_item(charge_item_id)
        WHERE charge_item_id IS NOT NULL;

    CREATE INDEX ix_dispense_item_order_item
        ON dbo.dispense_item(order_item_id)
        WHERE order_item_id IS NOT NULL;
END
GO
