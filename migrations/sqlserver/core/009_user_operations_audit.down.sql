-- Roll back only the added read-model fields and index; original audit rows remain.
USE BHIS;
GO
DROP INDEX IF EXISTS ix_audit_log_user_operations ON dbo.audit_log;
DROP INDEX IF EXISTS ix_audit_log_module_operations ON dbo.audit_log;
DROP INDEX IF EXISTS ix_audit_log_user_operations_actor ON dbo.audit_log;
IF COL_LENGTH(N'dbo.audit_log', N'build_version') IS NOT NULL ALTER TABLE dbo.audit_log DROP COLUMN build_version;
IF COL_LENGTH(N'dbo.audit_log', N'client_mac') IS NOT NULL ALTER TABLE dbo.audit_log DROP COLUMN client_mac;
IF COL_LENGTH(N'dbo.audit_log', N'description') IS NOT NULL ALTER TABLE dbo.audit_log DROP COLUMN description;
IF COL_LENGTH(N'dbo.audit_log', N'department_id') IS NOT NULL ALTER TABLE dbo.audit_log DROP COLUMN department_id;
IF COL_LENGTH(N'dbo.audit_log', N'module_id') IS NOT NULL ALTER TABLE dbo.audit_log DROP COLUMN module_id;
IF COL_LENGTH(N'dbo.audit_log', N'actor_username') IS NOT NULL ALTER TABLE dbo.audit_log DROP COLUMN actor_username;
GO
