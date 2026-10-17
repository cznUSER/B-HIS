-- BHIS durable idempotency records (P2-8/A04). Apply only to the selected BHIS database.
-- The application uses this table first for patient.create; later write scopes may reuse it.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
GO

USE BHIS;
GO

IF OBJECT_ID(N'dbo.idempotency_record', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.idempotency_record (
        scope         varchar(64)  NOT NULL,
        idem_key      varchar(128) NOT NULL,
        request_hash  char(64)     NOT NULL,
        state         varchar(16)  NOT NULL,
        resource_type varchar(64)  NULL,
        resource_id   varchar(64)  NULL,
        created_at    datetime2    NOT NULL CONSTRAINT df_idem_created DEFAULT SYSUTCDATETIME(),
        expires_at    datetime2    NOT NULL,
        completed_at  datetime2    NULL,
        CONSTRAINT pk_idempotency_record PRIMARY KEY (scope, idem_key),
        CONSTRAINT ck_idem_state CHECK (state IN ('PENDING', 'COMPLETED', 'FAILED'))
    );
    CREATE INDEX ix_idem_expires_at ON dbo.idempotency_record(expires_at);
END
GO
