-- BHIS register / department / doctor (P1-H03-REGISTER). Store-swappable: SQL Server Phase1.
-- Apply with sqlcmd; this file has no connection string and no server name.
--   sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/003_register.up.sql
-- PK style: varchar(64) (aligned with patient.patient_id / app_user.user_id).
-- Depends on: 001_identity (database BHIS), 002_patient (dbo.patient). Does not CREATE DATABASE.
-- 拍板#19：号源=科室+医生简化；无排班表。Encounter 域名 ≡ RegisterBill（register_id）。

SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

-- ---------------------------------------------------------------------------
-- dbo.department（科室主数据）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.department', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.department (
        department_id  varchar(64)    NOT NULL,
        dept_code      varchar(32)    NOT NULL,
        dept_name      nvarchar(64)   NOT NULL,
        status         varchar(16)    NOT NULL CONSTRAINT df_department_status DEFAULT 'ACTIVE',
        created_at     datetime2      NOT NULL CONSTRAINT df_department_created DEFAULT SYSUTCDATETIME(),
        created_by     varchar(64)    NULL,
        updated_at     datetime2      NOT NULL CONSTRAINT df_department_updated DEFAULT SYSUTCDATETIME(),
        updated_by     varchar(64)    NULL,
        is_deleted     bit            NOT NULL CONSTRAINT df_department_deleted DEFAULT 0,
        CONSTRAINT pk_department PRIMARY KEY (department_id),
        CONSTRAINT ck_department_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
    );

    CREATE UNIQUE INDEX ux_department_dept_code
        ON dbo.department(dept_code)
        WHERE is_deleted = 0;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.doctor（医生主数据；隶属科室）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.doctor', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.doctor (
        doctor_id      varchar(64)    NOT NULL,
        department_id  varchar(64)    NOT NULL,
        doctor_code    varchar(32)    NOT NULL,
        doctor_name    nvarchar(64)   NOT NULL,
        status         varchar(16)    NOT NULL CONSTRAINT df_doctor_status DEFAULT 'ACTIVE',
        created_at     datetime2      NOT NULL CONSTRAINT df_doctor_created DEFAULT SYSUTCDATETIME(),
        created_by     varchar(64)    NULL,
        updated_at     datetime2      NOT NULL CONSTRAINT df_doctor_updated DEFAULT SYSUTCDATETIME(),
        updated_by     varchar(64)    NULL,
        is_deleted     bit            NOT NULL CONSTRAINT df_doctor_deleted DEFAULT 0,
        CONSTRAINT pk_doctor PRIMARY KEY (doctor_id),
        CONSTRAINT fk_doctor_department FOREIGN KEY (department_id) REFERENCES dbo.department(department_id),
        CONSTRAINT ck_doctor_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
    );

    CREATE UNIQUE INDEX ux_doctor_doctor_code
        ON dbo.doctor(doctor_code)
        WHERE is_deleted = 0;

    CREATE INDEX ix_doctor_department
        ON dbo.doctor(department_id)
        WHERE is_deleted = 0;
END
GO

-- ---------------------------------------------------------------------------
-- dbo.register_bill（挂号单；域内 Encounter ≡ RegisterBill；PK=register_id）
-- 初始 status=NORMAL；退号默认仅 NORMAL→BACK（AC-04）；已用/就诊中等拒绝（AC-03）
-- ---------------------------------------------------------------------------
IF OBJECT_ID(N'dbo.register_bill', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.register_bill (
        register_id        varchar(64)    NOT NULL,
        register_no        varchar(32)    NOT NULL,
        patient_id         varchar(64)    NOT NULL,
        department_id      varchar(64)    NOT NULL,
        doctor_id          varchar(64)    NOT NULL,
        register_sort      varchar(32)    NULL,          -- 原 clinic_sort / 号别；可空占位
        visit_date         date           NOT NULL,
        status             varchar(16)    NOT NULL CONSTRAINT df_register_status DEFAULT 'NORMAL',
        register_fee       decimal(18,2)  NULL CONSTRAINT df_register_fee DEFAULT 0,
        register_charged   bit            NOT NULL CONSTRAINT df_register_charged DEFAULT 0,
        registered_at      datetime2      NOT NULL CONSTRAINT df_register_registered DEFAULT SYSUTCDATETIME(),
        back_operator_id   varchar(64)    NULL,
        back_operated_at   datetime2      NULL,
        created_at         datetime2      NOT NULL CONSTRAINT df_register_created DEFAULT SYSUTCDATETIME(),
        created_by         varchar(64)    NULL,
        updated_at         datetime2      NOT NULL CONSTRAINT df_register_updated DEFAULT SYSUTCDATETIME(),
        updated_by         varchar(64)    NULL,
        is_deleted         bit            NOT NULL CONSTRAINT df_register_deleted DEFAULT 0,
        CONSTRAINT pk_register_bill PRIMARY KEY (register_id),
        CONSTRAINT fk_register_patient FOREIGN KEY (patient_id) REFERENCES dbo.patient(patient_id),
        CONSTRAINT fk_register_department FOREIGN KEY (department_id) REFERENCES dbo.department(department_id),
        CONSTRAINT fk_register_doctor FOREIGN KEY (doctor_id) REFERENCES dbo.doctor(doctor_id),
        CONSTRAINT ck_register_status CHECK (status IN (
            'NORMAL', 'USED', 'BACK', 'DIAGNOSING', 'DIAGNOSED', 'PAUSED'
        )),
        CONSTRAINT ck_register_fee CHECK (register_fee IS NULL OR register_fee >= 0)
    );

    CREATE UNIQUE INDEX ux_register_no
        ON dbo.register_bill(register_no)
        WHERE is_deleted = 0;

    CREATE INDEX ix_register_patient_visit
        ON dbo.register_bill(patient_id, visit_date)
        WHERE is_deleted = 0;

    CREATE INDEX ix_register_status
        ON dbo.register_bill(status)
        WHERE is_deleted = 0;
END
GO
