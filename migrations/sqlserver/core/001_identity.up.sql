-- BHIS identity tables (P1-H04). Apply with sqlcmd against the instance; this file has no password.
--   SQLCMDPASSWORD is read from the environment (do not commit it).
--   sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/001_identity.up.sql
-- The bcrypt hash matches the memory-mode demo account documented in README. Plaintext is not stored here.

IF DB_ID(N'BHIS') IS NULL
BEGIN
    CREATE DATABASE BHIS;
END
GO

USE BHIS;
GO

IF OBJECT_ID(N'dbo.app_user', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.app_user (
        user_id        varchar(64)    NOT NULL PRIMARY KEY,
        username       varchar(64)    NOT NULL,
        display_name   nvarchar(128)  NOT NULL,
        password_hash  varchar(255)   NOT NULL,
        status         varchar(16)    NOT NULL CONSTRAINT df_app_user_status DEFAULT 'ACTIVE',
        created_at     datetime2      NOT NULL CONSTRAINT df_app_user_created DEFAULT SYSUTCDATETIME(),
        created_by     varchar(64)    NULL,
        updated_at     datetime2      NOT NULL CONSTRAINT df_app_user_updated DEFAULT SYSUTCDATETIME(),
        updated_by     varchar(64)    NULL,
        is_deleted     bit            NOT NULL CONSTRAINT df_app_user_deleted DEFAULT 0
    );
    CREATE UNIQUE INDEX ux_app_user_username ON dbo.app_user(username) WHERE is_deleted = 0;
END
GO

IF OBJECT_ID(N'dbo.role', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.role (
        role_code varchar(32)   NOT NULL PRIMARY KEY,
        role_name nvarchar(64)  NOT NULL
    );
END
GO

IF OBJECT_ID(N'dbo.user_role', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.user_role (
        user_id   varchar(64) NOT NULL,
        role_code varchar(32) NOT NULL,
        CONSTRAINT pk_user_role PRIMARY KEY (user_id, role_code),
        CONSTRAINT fk_user_role_user FOREIGN KEY (user_id) REFERENCES dbo.app_user(user_id),
        CONSTRAINT fk_user_role_role FOREIGN KEY (role_code) REFERENCES dbo.role(role_code)
    );
END
GO

IF OBJECT_ID(N'dbo.token_revocation', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.token_revocation (
        jti         varchar(64) NOT NULL PRIMARY KEY,
        user_id     varchar(64) NOT NULL,
        expires_at  datetime2   NOT NULL,
        revoked_at  datetime2   NOT NULL CONSTRAINT df_token_revoked DEFAULT SYSUTCDATETIME()
    );
END
GO

IF OBJECT_ID(N'dbo.audit_log', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.audit_log (
        audit_id       varchar(64)   NOT NULL PRIMARY KEY,
        occurred_at    datetime2     NOT NULL,
        actor_id       varchar(64)   NULL,
        actor_roles    nvarchar(512) NULL,
        action         varchar(64)   NOT NULL,
        resource_type  varchar(64)   NULL,
        resource_id    varchar(64)   NULL,
        outcome        varchar(16)   NOT NULL,
        error_code     varchar(64)   NULL,
        request_id     varchar(64)   NULL,
        client_ip      varchar(64)   NULL,
        remark         nvarchar(512) NULL
    );
END
GO

MERGE dbo.role AS t
USING (VALUES
    ('REGISTRAR',  N'建档挂号员'),
    ('DOCTOR',     N'门诊医生'),
    ('CASHIER',    N'收费员'),
    ('PHARMACIST', N'药房人员'),
    ('SYSTEM',     N'系统')
) AS s(role_code, role_name)
ON t.role_code = s.role_code
WHEN NOT MATCHED THEN INSERT (role_code, role_name) VALUES (s.role_code, s.role_name);
GO

IF NOT EXISTS (SELECT 1 FROM dbo.app_user WHERE user_id = '00000000-0000-4000-8000-000000000001')
INSERT INTO dbo.app_user (user_id, username, display_name, password_hash, status, created_by, updated_by)
VALUES (
    '00000000-0000-4000-8000-000000000001',
    'admin',
    N'系统管理员',
    '$2a$10$uVUbI161ahIWvB2/rbXUjuWubRAMog3zNNfldqp4mx/M.ZtuD8TgS',
    'ACTIVE',
    'system',
    'system'
);
GO

IF NOT EXISTS (SELECT 1 FROM dbo.user_role WHERE user_id = '00000000-0000-4000-8000-000000000001' AND role_code = 'SYSTEM')
INSERT INTO dbo.user_role (user_id, role_code) VALUES ('00000000-0000-4000-8000-000000000001', 'SYSTEM');
GO

IF NOT EXISTS (SELECT 1 FROM dbo.app_user WHERE user_id = '00000000-0000-4000-8000-000000000002')
INSERT INTO dbo.app_user (user_id, username, display_name, password_hash, status, created_by, updated_by)
VALUES (
    '00000000-0000-4000-8000-000000000002',
    'disabled',
    N'已停用账号',
    '$2a$10$uVUbI161ahIWvB2/rbXUjuWubRAMog3zNNfldqp4mx/M.ZtuD8TgS',
    'DISABLED',
    'system',
    'system'
);
GO
