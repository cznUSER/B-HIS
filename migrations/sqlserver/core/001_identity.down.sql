-- Drops BHIS identity tables. Does not DROP DATABASE. No credentials in this file.
USE BHIS;
GO
DROP TABLE IF EXISTS dbo.audit_log;
DROP TABLE IF EXISTS dbo.token_revocation;
DROP TABLE IF EXISTS dbo.user_role;
DROP TABLE IF EXISTS dbo.app_user;
DROP TABLE IF EXISTS dbo.role;
GO
