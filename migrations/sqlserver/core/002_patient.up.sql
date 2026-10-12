-- BHIS patient / MPI (P1-H03-PATIENT). Store-swappable: this script is SQL Server Phase1; PG parallel later.
-- Apply with sqlcmd; this file has no connection string and no server name.
--   sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/002_patient.up.sql
-- PK style: varchar(64) (aligned with app_user.user_id); OpenAPI may still describe uuid-format strings.
-- Depends on: 001_identity (database BHIS). Does not CREATE DATABASE.

SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

IF OBJECT_ID(N'dbo.patient', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.patient (
        patient_id     varchar(64)    NOT NULL,
        hosp_pk_no     varchar(32)    NOT NULL,
        patient_no     varchar(32)    NULL,
        card_no        varchar(32)    NULL,
        name           nvarchar(64)   NOT NULL,
        sex            varchar(8)     NULL,
        birthday       date           NOT NULL,
        id_type        varchar(16)    NULL,
        id_number      varchar(32)    NULL,
        phone          varchar(32)    NULL,
        address        nvarchar(256)  NULL,
        fare_identity  varchar(32)    NULL,
        card_status    varchar(16)    NOT NULL CONSTRAINT df_patient_card_status DEFAULT 'NOT_SENT',
        status         varchar(16)    NOT NULL CONSTRAINT df_patient_status DEFAULT 'ACTIVE',
        created_at     datetime2      NOT NULL CONSTRAINT df_patient_created DEFAULT SYSUTCDATETIME(),
        created_by     varchar(64)    NULL,
        updated_at     datetime2      NOT NULL CONSTRAINT df_patient_updated DEFAULT SYSUTCDATETIME(),
        updated_by     varchar(64)    NULL,
        is_deleted     bit            NOT NULL CONSTRAINT df_patient_deleted DEFAULT 0,
        CONSTRAINT pk_patient PRIMARY KEY (patient_id),
        CONSTRAINT ck_patient_birthday CHECK (birthday > CAST('1900-01-01' AS date)),
        CONSTRAINT ck_patient_card_status CHECK (card_status IN ('NOT_SENT', 'NORMAL', 'LOST', 'CANCELLED')),
        CONSTRAINT ck_patient_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
    );

    -- hosp_pk_no globally unique among non-deleted rows
    CREATE UNIQUE INDEX ux_patient_hosp_pk
        ON dbo.patient(hosp_pk_no)
        WHERE is_deleted = 0;

    -- id_number unique when non-null / non-empty and not deleted (duplicate → 409 DUPLICATE_*)
    CREATE UNIQUE INDEX ux_patient_id_number
        ON dbo.patient(id_number)
        WHERE id_number IS NOT NULL AND id_number <> '' AND is_deleted = 0;

    -- card_no unique when non-null / non-empty and not deleted
    CREATE UNIQUE INDEX ux_patient_card_no
        ON dbo.patient(card_no)
        WHERE card_no IS NOT NULL AND card_no <> '' AND is_deleted = 0;

    CREATE INDEX ix_patient_name ON dbo.patient(name) WHERE is_deleted = 0;
    CREATE INDEX ix_patient_phone ON dbo.patient(phone) WHERE is_deleted = 0 AND phone IS NOT NULL;
    CREATE INDEX ix_patient_address ON dbo.patient(address) WHERE is_deleted = 0 AND address IS NOT NULL;
END
GO
