-- BHIS charge / charge_item / payment (P1-H03-CHARGE). Store-swappable: SQL Server Phase1.
-- Apply with sqlcmd; this file has no server name and no credentials.
--   sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/005_charge.up.sql
-- PK style: varchar(64) (aligned with order_sheet.order_id / register_bill.register_id).
-- Depends on: 001_identity (database BHIS), 002_patient, 003_register, 004_order. Does not CREATE DATABASE.
-- 拍板#22：强制有效 register_id（≡encounter_id）；现金一次付清；F=0 主路径；默认不拆方 1:1；
-- 全退 newSquare=null（不生成金额0新单）；实收=应收；收费侧拦库存；已发药须先退药。

SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

-- ---------------------------------------------------------------------------
-- dbo.charge（Square 结算单）
-- 域内 encounter_id ≡ register_id（列名用 register_id）
-- Phase1 主路径：一处方一张 Charge（A21 默认不拆；不建 uk_charge_order 强制）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.charge', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.charge (
        charge_id             varchar(64)    NOT NULL,
        square_no             varchar(32)    NOT NULL,
        order_id              varchar(64)    NULL,       -- FK→order_sheet；Phase1 主路径有值；允许多来源时 nullable
        register_id           varchar(64)    NOT NULL,   -- ≡ encounter_id；FK→register_bill 必填（#22a / C-07）
        patient_id            varchar(64)    NOT NULL,   -- FK→patient
        total_fee             decimal(18,2)  NOT NULL CONSTRAINT df_charge_total_fee DEFAULT 0,      -- 毛额 G
        square_fee            decimal(18,2)  NOT NULL CONSTRAINT df_charge_square_fee DEFAULT 0,     -- 应收 R = G - F
        favourable_fee        decimal(18,2)  NOT NULL CONSTRAINT df_charge_favourable DEFAULT 0,     -- 优惠 F；验收主路径 0
        status                varchar(16)    NOT NULL CONSTRAINT df_charge_status DEFAULT 'NORMAL', -- NORMAL|ABATE
        successor_charge_id   varchar(64)    NULL,       -- 作废后新单（未退部分）；全退则为 NULL
        abated_from_id        varchar(64)    NULL,       -- 若本单由退费生成，指向被作废原单
        paid_at               datetime2      NULL,       -- 支付成功时间；待收为空
        created_at            datetime2      NOT NULL CONSTRAINT df_charge_created DEFAULT SYSUTCDATETIME(),
        created_by            varchar(64)    NULL,
        updated_at            datetime2      NOT NULL CONSTRAINT df_charge_updated DEFAULT SYSUTCDATETIME(),
        updated_by            varchar(64)    NULL,
        is_deleted            bit            NOT NULL CONSTRAINT df_charge_deleted DEFAULT 0,
        CONSTRAINT pk_charge PRIMARY KEY (charge_id),
        CONSTRAINT fk_charge_order FOREIGN KEY (order_id) REFERENCES dbo.order_sheet(order_id),
        CONSTRAINT fk_charge_register FOREIGN KEY (register_id) REFERENCES dbo.register_bill(register_id),
        CONSTRAINT fk_charge_patient FOREIGN KEY (patient_id) REFERENCES dbo.patient(patient_id),
        CONSTRAINT fk_charge_successor FOREIGN KEY (successor_charge_id) REFERENCES dbo.charge(charge_id),
        CONSTRAINT fk_charge_abated_from FOREIGN KEY (abated_from_id) REFERENCES dbo.charge(charge_id),
        CONSTRAINT ck_charge_status CHECK (status IN ('NORMAL', 'ABATE')),
        CONSTRAINT ck_charge_total_fee CHECK (total_fee >= 0),
        CONSTRAINT ck_charge_square_fee CHECK (square_fee >= 0),
        CONSTRAINT ck_charge_favourable_fee CHECK (favourable_fee >= 0)
    );

    CREATE UNIQUE INDEX ux_charge_square_no
        ON dbo.charge(square_no)
        WHERE is_deleted = 0;

    CREATE INDEX ix_charge_register_created
        ON dbo.charge(register_id, created_at)
        WHERE is_deleted = 0;

    CREATE INDEX ix_charge_patient
        ON dbo.charge(patient_id)
        WHERE is_deleted = 0;

    CREATE INDEX ix_charge_status
        ON dbo.charge(status)
        WHERE is_deleted = 0;

    CREATE INDEX ix_charge_order
        ON dbo.charge(order_id)
        WHERE is_deleted = 0 AND order_id IS NOT NULL;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.charge_item（结算明细；qty>0；refunded_qty ∈ [0, qty]）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.charge_item', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.charge_item (
        charge_item_id  varchar(64)    NOT NULL,
        charge_id       varchar(64)    NOT NULL,
        order_item_id   varchar(64)    NULL,       -- FK→order_item；可空（非处方来源时）
        line_no         int            NOT NULL,
        item_no         varchar(32)    NOT NULL,
        item_name       nvarchar(128)  NOT NULL,
        qty             decimal(18,3)  NOT NULL,   -- 收费数量；须 >0（C-02）
        unit_price      decimal(18,2)  NOT NULL,   -- 单价
        total_fee       decimal(18,2)  NOT NULL,   -- 行毛额
        can_back        bit            NOT NULL CONSTRAINT df_charge_item_can_back DEFAULT 1,
        refunded_qty    decimal(18,3)  NOT NULL CONSTRAINT df_charge_item_refunded DEFAULT 0,
        created_at      datetime2      NOT NULL CONSTRAINT df_charge_item_created DEFAULT SYSUTCDATETIME(),
        updated_at      datetime2      NOT NULL CONSTRAINT df_charge_item_updated DEFAULT SYSUTCDATETIME(),
        CONSTRAINT pk_charge_item PRIMARY KEY (charge_item_id),
        CONSTRAINT fk_charge_item_charge FOREIGN KEY (charge_id) REFERENCES dbo.charge(charge_id),
        CONSTRAINT fk_charge_item_order_item FOREIGN KEY (order_item_id) REFERENCES dbo.order_item(order_item_id),
        CONSTRAINT ck_charge_item_qty CHECK (qty > 0),
        CONSTRAINT ck_charge_item_unit_price CHECK (unit_price >= 0),
        CONSTRAINT ck_charge_item_total_fee CHECK (total_fee >= 0),
        CONSTRAINT ck_charge_item_refunded CHECK (refunded_qty >= 0 AND refunded_qty <= qty)
    );

    CREATE UNIQUE INDEX ux_charge_item_line
        ON dbo.charge_item(charge_id, line_no);

    CREATE INDEX ix_charge_item_charge
        ON dbo.charge_item(charge_id);
END
GO

-- ---------------------------------------------------------------------------
-- dbo.payment（支付/退款流水；只追加）
-- Phase1：pay_type 演示 CASH；电子枚举预留不联调；成功写 SUCCEEDED
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.payment', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.payment (
        payment_id    varchar(64)    NOT NULL,
        payment_no    varchar(32)    NOT NULL,
        charge_id     varchar(64)    NOT NULL,
        direction     varchar(8)     NOT NULL,   -- CHARGE | REFUND
        pay_type      varchar(16)    NOT NULL,   -- CASH / OTHER_E / WECHAT / ALIPAY / OTHER
        amount        decimal(18,2)  NOT NULL,
        status        varchar(16)    NOT NULL CONSTRAINT df_payment_status DEFAULT 'PENDING',
        paid_at       datetime2      NULL,       -- 成功时间；Phase1 成功即写
        operator_id   varchar(64)    NULL,
        created_at    datetime2      NOT NULL CONSTRAINT df_payment_created DEFAULT SYSUTCDATETIME(),
        created_by    varchar(64)    NULL,
        updated_at    datetime2      NOT NULL CONSTRAINT df_payment_updated DEFAULT SYSUTCDATETIME(),
        updated_by    varchar(64)    NULL,
        CONSTRAINT pk_payment PRIMARY KEY (payment_id),
        CONSTRAINT fk_payment_charge FOREIGN KEY (charge_id) REFERENCES dbo.charge(charge_id),
        CONSTRAINT ck_payment_direction CHECK (direction IN ('CHARGE', 'REFUND')),
        CONSTRAINT ck_payment_pay_type CHECK (pay_type IN ('CASH', 'OTHER_E', 'WECHAT', 'ALIPAY', 'OTHER')),
        CONSTRAINT ck_payment_status CHECK (status IN ('SUCCEEDED', 'FAILED', 'PENDING')),
        CONSTRAINT ck_payment_amount CHECK (amount >= 0)
    );

    CREATE UNIQUE INDEX ux_payment_no
        ON dbo.payment(payment_no);

    CREATE INDEX ix_payment_charge
        ON dbo.payment(charge_id);
END
GO
