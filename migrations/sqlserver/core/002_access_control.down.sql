USE BHIS;
GO
IF OBJECT_ID(N'dbo.user_permission_override', N'U') IS NOT NULL DROP TABLE dbo.user_permission_override;
IF OBJECT_ID(N'dbo.access_role_permission', N'U') IS NOT NULL DROP TABLE dbo.access_role_permission;
IF OBJECT_ID(N'dbo.access_role', N'U') IS NOT NULL DROP TABLE dbo.access_role;
IF OBJECT_ID(N'dbo.access_permission', N'U') IS NOT NULL DROP TABLE dbo.access_permission;
IF OBJECT_ID(N'dbo.permission_mode', N'U') IS NOT NULL DROP TABLE dbo.permission_mode;
IF OBJECT_ID(N'dbo.permission_category', N'U') IS NOT NULL DROP TABLE dbo.permission_category;
IF OBJECT_ID(N'dbo.access_module', N'U') IS NOT NULL DROP TABLE dbo.access_module;
GO
DECLARE @constraint sysname;
DECLARE drop_user_defaults CURSOR LOCAL FAST_FORWARD FOR
SELECT dc.name
FROM sys.default_constraints AS dc
INNER JOIN sys.columns AS c ON c.object_id = dc.parent_object_id AND c.column_id = dc.parent_column_id
WHERE dc.parent_object_id = OBJECT_ID(N'dbo.app_user')
  AND c.name IN (N'must_change_password', N'cannot_modify_password', N'account_locked', N'account_stopped', N'hidden', N'is_system', N'is_super', N'is_developer', N'failed_login_count', N'auth_version', N'input_code_compare_mode');
OPEN drop_user_defaults;
FETCH NEXT FROM drop_user_defaults INTO @constraint;
WHILE @@FETCH_STATUS = 0
BEGIN
    EXEC(N'ALTER TABLE dbo.app_user DROP CONSTRAINT ' + QUOTENAME(@constraint));
    FETCH NEXT FROM drop_user_defaults INTO @constraint;
END;
CLOSE drop_user_defaults;
DEALLOCATE drop_user_defaults;
GO
IF COL_LENGTH(N'dbo.app_user', N'auth_version') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN auth_version;
IF COL_LENGTH(N'dbo.app_user', N'input_code_compare_mode') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN input_code_compare_mode;
IF COL_LENGTH(N'dbo.app_user', N'expand_interface2') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN expand_interface2;
IF COL_LENGTH(N'dbo.app_user', N'expand_interface1') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN expand_interface1;
IF COL_LENGTH(N'dbo.app_user', N'service_star') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN service_star;
IF COL_LENGTH(N'dbo.app_user', N'failed_login_count') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN failed_login_count;
IF COL_LENGTH(N'dbo.app_user', N'password_changed_at') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN password_changed_at;
IF COL_LENGTH(N'dbo.app_user', N'lock_until') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN lock_until;
IF COL_LENGTH(N'dbo.app_user', N'last_login_ip') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN last_login_ip;
IF COL_LENGTH(N'dbo.app_user', N'last_login_at') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN last_login_at;
IF COL_LENGTH(N'dbo.app_user', N'is_developer') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN is_developer;
IF COL_LENGTH(N'dbo.app_user', N'is_super') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN is_super;
IF COL_LENGTH(N'dbo.app_user', N'is_system') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN is_system;
IF COL_LENGTH(N'dbo.app_user', N'hidden') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN hidden;
IF COL_LENGTH(N'dbo.app_user', N'account_stopped') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN account_stopped;
IF COL_LENGTH(N'dbo.app_user', N'account_locked') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN account_locked;
IF COL_LENGTH(N'dbo.app_user', N'cannot_modify_password') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN cannot_modify_password;
IF COL_LENGTH(N'dbo.app_user', N'must_change_password') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN must_change_password;
IF COL_LENGTH(N'dbo.app_user', N'input_code') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN input_code;
IF COL_LENGTH(N'dbo.app_user', N'card_no') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN card_no;
IF COL_LENGTH(N'dbo.app_user', N'description') IS NOT NULL ALTER TABLE dbo.app_user DROP COLUMN description;
GO
-- dbo.role.role_name remains nvarchar(128): user_role can still reference custom roles after access tables are removed.
