-- Adds the fields consumed by the user-operations log while preserving existing audit rows.
SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
USE BHIS;
GO

IF COL_LENGTH(N'dbo.audit_log', N'actor_username') IS NULL ALTER TABLE dbo.audit_log ADD actor_username nvarchar(64) NULL;
IF COL_LENGTH(N'dbo.audit_log', N'module_id') IS NULL ALTER TABLE dbo.audit_log ADD module_id varchar(64) NULL;
IF COL_LENGTH(N'dbo.audit_log', N'department_id') IS NULL ALTER TABLE dbo.audit_log ADD department_id varchar(64) NULL;
IF COL_LENGTH(N'dbo.audit_log', N'description') IS NULL ALTER TABLE dbo.audit_log ADD description nvarchar(512) NULL;
IF COL_LENGTH(N'dbo.audit_log', N'client_mac') IS NULL ALTER TABLE dbo.audit_log ADD client_mac varchar(64) NULL;
IF COL_LENGTH(N'dbo.audit_log', N'build_version') IS NULL ALTER TABLE dbo.audit_log ADD build_version varchar(64) NULL;
GO

UPDATE a SET actor_username = u.username
FROM dbo.audit_log a
JOIN dbo.app_user u ON u.user_id = a.actor_id
WHERE a.actor_username IS NULL;

UPDATE dbo.audit_log
SET module_id = CASE
    WHEN action IN ('patient.create','register.create','register.check-in','register.cancel','order.create','order.commit','order.abate','charge.create','charge.pay','charge.refund','dispense.create','dispense.return') THEN 'module-outpatient'
    WHEN action IN ('LOGIN_SUCCESS','LOGIN_FAILURE','LOGOUT','department.create','department.update','department.deactivate','doctor.create','doctor.update','catalog.create','catalog.update','inventory.adjust','dict.create','dict.update','user.create','user.update','user.deactivate','user.reset_password','user.change_password','role.create','role.update','permission.create','permission.update','user.permission_override','module.create','module.update','category.create','category.update','mode.create','mode.update') THEN 'module-master'
    ELSE module_id
END
WHERE module_id IS NULL;

UPDATE dbo.audit_log
SET description = CASE action
    WHEN 'LOGIN_SUCCESS' THEN N'用户登录'
    WHEN 'LOGIN_FAILURE' THEN N'用户登录失败'
    WHEN 'LOGOUT' THEN N'用户退出'
    WHEN 'patient.create' THEN N'患者建档'
    WHEN 'register.create' THEN N'挂号'
    WHEN 'register.check-in' THEN N'签到'
    WHEN 'register.cancel' THEN N'退号'
    WHEN 'order.create' THEN N'新建医嘱'
    WHEN 'order.commit' THEN N'提交医嘱'
    WHEN 'order.abate' THEN N'作废医嘱'
    WHEN 'charge.create' THEN N'创建收费单'
    WHEN 'charge.pay' THEN N'收费'
    WHEN 'charge.refund' THEN N'退费'
    WHEN 'dispense.create' THEN N'发药'
    WHEN 'dispense.return' THEN N'退药'
    WHEN 'department.create' THEN N'创建科室'
    WHEN 'department.update' THEN N'修改科室'
    WHEN 'department.deactivate' THEN N'停用科室'
    WHEN 'doctor.create' THEN N'创建医生'
    WHEN 'doctor.update' THEN N'修改医生'
    WHEN 'catalog.create' THEN N'创建目录项'
    WHEN 'catalog.update' THEN N'修改目录项'
    WHEN 'inventory.adjust' THEN N'调整库存'
    WHEN 'dict.create' THEN N'创建字典项'
    WHEN 'dict.update' THEN N'修改字典项'
    WHEN 'user.create' THEN N'创建用户'
    WHEN 'user.update' THEN N'修改用户'
    WHEN 'user.deactivate' THEN N'停用用户'
    WHEN 'user.reset_password' THEN N'重置用户密码'
    WHEN 'user.change_password' THEN N'修改密码'
    WHEN 'role.create' THEN N'创建角色'
    WHEN 'role.update' THEN N'修改角色'
    WHEN 'permission.create' THEN N'创建权限'
    WHEN 'permission.update' THEN N'修改权限'
    WHEN 'user.permission_override' THEN N'修改用户权限例外'
    WHEN 'module.create' THEN N'创建功能模块'
    WHEN 'module.update' THEN N'修改功能模块'
    WHEN 'category.create' THEN N'创建权限分类'
    WHEN 'category.update' THEN N'修改权限分类'
    WHEN 'mode.create' THEN N'创建授权方式'
    WHEN 'mode.update' THEN N'修改授权方式'
    ELSE action
END
WHERE description IS NULL;
GO

IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'ix_audit_log_user_operations' AND object_id = OBJECT_ID(N'dbo.audit_log'))
    CREATE INDEX ix_audit_log_user_operations ON dbo.audit_log (occurred_at DESC, audit_id DESC) INCLUDE (module_id, actor_id, actor_username, department_id, action, outcome, error_code, request_id, client_ip, client_mac, build_version);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'ix_audit_log_module_operations' AND object_id = OBJECT_ID(N'dbo.audit_log'))
    CREATE INDEX ix_audit_log_module_operations ON dbo.audit_log (module_id, occurred_at DESC, audit_id DESC) INCLUDE (actor_id, actor_username, department_id, action, outcome, error_code, request_id, client_ip, client_mac, build_version);
IF NOT EXISTS (SELECT 1 FROM sys.indexes WHERE name = N'ix_audit_log_user_operations_actor' AND object_id = OBJECT_ID(N'dbo.audit_log'))
    CREATE INDEX ix_audit_log_user_operations_actor ON dbo.audit_log (actor_id, occurred_at DESC, audit_id DESC) INCLUDE (module_id, actor_username, department_id, action, outcome, error_code, request_id, client_ip, client_mac, build_version);
GO
