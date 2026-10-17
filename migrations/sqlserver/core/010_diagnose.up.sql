-- Optional structured diagnosis lifecycle (P2-H01-DIAGNOSE).
-- Depends on 003_register. Does not create the database.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

IF OBJECT_ID(N'dbo.diagnose', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.diagnose (
        diagnose_id  varchar(64)    NOT NULL,
        encounter_id varchar(64)    NOT NULL,
        status       varchar(16)    NOT NULL CONSTRAINT df_diagnose_status DEFAULT 'SAVE',
        content      nvarchar(max)  NULL,
        created_at   datetime2       NOT NULL CONSTRAINT df_diagnose_created DEFAULT SYSUTCDATETIME(),
        created_by   varchar(64)     NULL,
        updated_at   datetime2       NOT NULL CONSTRAINT df_diagnose_updated DEFAULT SYSUTCDATETIME(),
        updated_by   varchar(64)     NULL,
        is_deleted   bit             NOT NULL CONSTRAINT df_diagnose_deleted DEFAULT 0,
        CONSTRAINT pk_diagnose PRIMARY KEY (diagnose_id),
        CONSTRAINT fk_diagnose_register FOREIGN KEY (encounter_id) REFERENCES dbo.register_bill(register_id),
        CONSTRAINT ck_diagnose_status CHECK (status IN ('NEW', 'SAVE', 'COMMIT', 'ABATE'))
    );

    CREATE INDEX ix_diagnose_encounter
        ON dbo.diagnose(encounter_id, created_at, diagnose_id)
        WHERE is_deleted = 0;
END
GO
