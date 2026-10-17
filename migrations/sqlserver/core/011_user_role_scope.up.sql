-- 011_user_role_scope: explicit per-user department scopes per assigned role.
-- Apply only after 001_identity, 003_register, and 002_access_control.
SET XACT_ABORT ON;
SET QUOTED_IDENTIFIER ON;
GO

IF OBJECT_ID(N'dbo.user_role_scope', N'U') IS NULL
BEGIN
    CREATE TABLE dbo.user_role_scope (
        user_id       varchar(64) NOT NULL,
        role_code     varchar(32) NOT NULL,
        department_id varchar(64) NOT NULL,
        status        varchar(16) NOT NULL,
        CONSTRAINT pk_user_role_scope PRIMARY KEY (user_id, role_code, department_id),
        CONSTRAINT ck_user_role_scope_status CHECK (status IN ('ACTIVE', 'DISABLED')),
        CONSTRAINT fk_user_role_scope_user_role FOREIGN KEY (user_id, role_code)
            REFERENCES dbo.user_role(user_id, role_code),
        CONSTRAINT fk_user_role_scope_department FOREIGN KEY (department_id)
            REFERENCES dbo.department(department_id)
    );
END
GO
