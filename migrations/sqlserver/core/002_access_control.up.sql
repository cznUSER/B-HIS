SET ANSI_NULLS ON;
SET QUOTED_IDENTIFIER ON;
USE BHIS;
GO

IF COL_LENGTH(N'dbo.app_user', N'description') IS NULL ALTER TABLE dbo.app_user ADD description nvarchar(512) NULL;
IF COL_LENGTH(N'dbo.app_user', N'card_no') IS NULL ALTER TABLE dbo.app_user ADD card_no varchar(64) NULL;
IF COL_LENGTH(N'dbo.app_user', N'input_code') IS NULL ALTER TABLE dbo.app_user ADD input_code varchar(64) NULL;
IF COL_LENGTH(N'dbo.app_user', N'service_star') IS NULL ALTER TABLE dbo.app_user ADD service_star tinyint NULL;
IF COL_LENGTH(N'dbo.app_user', N'expand_interface1') IS NULL ALTER TABLE dbo.app_user ADD expand_interface1 int NULL;
IF COL_LENGTH(N'dbo.app_user', N'expand_interface2') IS NULL ALTER TABLE dbo.app_user ADD expand_interface2 int NULL;
IF COL_LENGTH(N'dbo.app_user', N'input_code_compare_mode') IS NULL
BEGIN
    ALTER TABLE dbo.app_user ADD input_code_compare_mode int NULL CONSTRAINT df_app_user_input_code_compare_mode DEFAULT 2;
    EXEC(N'UPDATE dbo.app_user SET input_code_compare_mode=2 WHERE input_code_compare_mode IS NULL;');
END;
IF COL_LENGTH(N'dbo.app_user', N'must_change_password') IS NULL ALTER TABLE dbo.app_user ADD must_change_password bit NOT NULL CONSTRAINT df_app_user_must_change_password DEFAULT 0;
IF COL_LENGTH(N'dbo.app_user', N'cannot_modify_password') IS NULL ALTER TABLE dbo.app_user ADD cannot_modify_password bit NOT NULL CONSTRAINT df_app_user_cannot_modify_password DEFAULT 0;
IF COL_LENGTH(N'dbo.app_user', N'account_locked') IS NULL ALTER TABLE dbo.app_user ADD account_locked bit NOT NULL CONSTRAINT df_app_user_account_locked DEFAULT 0;
IF COL_LENGTH(N'dbo.app_user', N'account_stopped') IS NULL ALTER TABLE dbo.app_user ADD account_stopped bit NOT NULL CONSTRAINT df_app_user_account_stopped DEFAULT 0;
IF COL_LENGTH(N'dbo.app_user', N'hidden') IS NULL ALTER TABLE dbo.app_user ADD hidden bit NOT NULL CONSTRAINT df_app_user_hidden DEFAULT 0;
IF COL_LENGTH(N'dbo.app_user', N'is_system') IS NULL ALTER TABLE dbo.app_user ADD is_system bit NOT NULL CONSTRAINT df_app_user_is_system DEFAULT 0;
IF COL_LENGTH(N'dbo.app_user', N'is_super') IS NULL ALTER TABLE dbo.app_user ADD is_super bit NOT NULL CONSTRAINT df_app_user_is_super DEFAULT 0;
IF COL_LENGTH(N'dbo.app_user', N'is_developer') IS NULL
BEGIN
    ALTER TABLE dbo.app_user ADD is_developer bit NOT NULL CONSTRAINT df_app_user_is_developer DEFAULT 0;
    -- The built-in local system account is the one-time bootstrap developer for access administration.
    EXEC(N'UPDATE dbo.app_user SET is_developer = 1 WHERE username = ''system'' AND is_deleted = 0;');
END;
IF COL_LENGTH(N'dbo.app_user', N'last_login_at') IS NULL ALTER TABLE dbo.app_user ADD last_login_at datetime2 NULL;
IF COL_LENGTH(N'dbo.app_user', N'last_login_ip') IS NULL ALTER TABLE dbo.app_user ADD last_login_ip varchar(64) NULL;
IF COL_LENGTH(N'dbo.app_user', N'lock_until') IS NULL ALTER TABLE dbo.app_user ADD lock_until datetime2 NULL;
IF COL_LENGTH(N'dbo.app_user', N'password_changed_at') IS NULL ALTER TABLE dbo.app_user ADD password_changed_at datetime2 NULL;
IF COL_LENGTH(N'dbo.app_user', N'failed_login_count') IS NULL ALTER TABLE dbo.app_user ADD failed_login_count int NOT NULL CONSTRAINT df_app_user_failed_login_count DEFAULT 0;
IF COL_LENGTH(N'dbo.app_user', N'auth_version') IS NULL ALTER TABLE dbo.app_user ADD auth_version bigint NOT NULL CONSTRAINT df_app_user_auth_version DEFAULT 0;
IF COL_LENGTH(N'dbo.role', N'role_name') < 256 ALTER TABLE dbo.role ALTER COLUMN role_name nvarchar(128) NOT NULL;
GO

IF OBJECT_ID(N'dbo.access_module', N'U') IS NULL
CREATE TABLE dbo.access_module (
    module_id varchar(64) NOT NULL CONSTRAINT pk_access_module PRIMARY KEY,
    module_code varchar(64) NOT NULL CONSTRAINT ux_access_module_code UNIQUE,
    module_name nvarchar(128) NOT NULL,
    description nvarchar(512) NULL,
    status varchar(16) NOT NULL CONSTRAINT df_access_module_status DEFAULT 'ACTIVE',
    hidden bit NOT NULL CONSTRAINT df_access_module_hidden DEFAULT 0,
    department_functions bigint NOT NULL CONSTRAINT df_access_module_department_functions DEFAULT 0,
    need_specify_department bit NOT NULL CONSTRAINT df_access_module_need_specify_department DEFAULT 0,
    second_permission_id int NOT NULL CONSTRAINT df_access_module_second_permission_id DEFAULT 0,
    home_page_url nvarchar(512) NULL
);
IF OBJECT_ID(N'dbo.permission_category', N'U') IS NULL
CREATE TABLE dbo.permission_category (
    category_id varchar(64) NOT NULL CONSTRAINT pk_permission_category PRIMARY KEY,
    category_name nvarchar(128) NOT NULL,
    description nvarchar(512) NULL,
    status varchar(16) NOT NULL CONSTRAINT df_permission_category_status DEFAULT 'ACTIVE'
);
IF OBJECT_ID(N'dbo.permission_mode', N'U') IS NULL
CREATE TABLE dbo.permission_mode (
    mode_code varchar(32) NOT NULL CONSTRAINT pk_permission_mode PRIMARY KEY,
    mode_name nvarchar(128) NOT NULL,
    status varchar(16) NOT NULL CONSTRAINT df_permission_mode_status DEFAULT 'ACTIVE'
);
IF OBJECT_ID(N'dbo.access_permission', N'U') IS NULL
CREATE TABLE dbo.access_permission (
    permission_code varchar(128) NOT NULL CONSTRAINT pk_access_permission PRIMARY KEY,
    permission_name nvarchar(128) NOT NULL,
    description nvarchar(512) NULL,
    module_code varchar(64) NOT NULL,
    category_id varchar(64) NULL,
    mode_code varchar(32) NULL,
    second_permission_code varchar(128) NULL,
    actions varchar(64) NULL,
    status varchar(16) NOT NULL CONSTRAINT df_access_permission_status DEFAULT 'ACTIVE',
    hidden bit NOT NULL CONSTRAINT df_access_permission_hidden DEFAULT 0,
    db_role_name nvarchar(128) NULL,
    need_second_permission bit NOT NULL CONSTRAINT df_access_permission_need_second_permission DEFAULT 0
);
IF OBJECT_ID(N'dbo.access_role', N'U') IS NULL
CREATE TABLE dbo.access_role (
    role_code varchar(32) NOT NULL PRIMARY KEY,
    role_name nvarchar(128) NOT NULL,
    description nvarchar(512) NULL,
    module_code varchar(64) NULL,
    department_id varchar(64) NULL,
    status varchar(16) NOT NULL CONSTRAINT df_access_role_status DEFAULT 'ACTIVE',
    hidden bit NOT NULL CONSTRAINT df_access_role_hidden DEFAULT 0,
    is_system bit NOT NULL CONSTRAINT df_access_role_system DEFAULT 0,
    order_sort int NOT NULL CONSTRAINT df_access_role_order_sort DEFAULT 0,
    stand_name nvarchar(128) NULL
);
IF OBJECT_ID(N'dbo.access_role_permission', N'U') IS NULL
CREATE TABLE dbo.access_role_permission (
    role_code varchar(32) NOT NULL,
    permission_code varchar(128) NOT NULL,
    CONSTRAINT pk_access_role_permission PRIMARY KEY (role_code, permission_code),
    CONSTRAINT fk_access_role_permission_role FOREIGN KEY (role_code) REFERENCES dbo.access_role(role_code),
    CONSTRAINT fk_access_role_permission_permission FOREIGN KEY (permission_code) REFERENCES dbo.access_permission(permission_code)
);
IF OBJECT_ID(N'dbo.user_permission_override', N'U') IS NULL
CREATE TABLE dbo.user_permission_override (
    user_id varchar(64) NOT NULL,
    permission_code varchar(128) NOT NULL,
    module_code varchar(64) NOT NULL CONSTRAINT df_user_permission_override_module DEFAULT '',
    department_id varchar(64) NOT NULL CONSTRAINT df_user_permission_override_department DEFAULT '',
    effect varchar(8) NOT NULL CONSTRAINT ck_user_permission_override_effect CHECK (effect IN ('ALLOW','DENY')),
    CONSTRAINT pk_user_permission_override PRIMARY KEY (user_id, permission_code, module_code, department_id),
    CONSTRAINT fk_user_permission_override_user FOREIGN KEY (user_id) REFERENCES dbo.app_user(user_id),
    CONSTRAINT fk_user_permission_override_permission FOREIGN KEY (permission_code) REFERENCES dbo.access_permission(permission_code)
);
IF NOT EXISTS (
    SELECT 1 FROM sys.check_constraints
    WHERE name = N'ck_user_permission_override_effect'
      AND parent_object_id = OBJECT_ID(N'dbo.user_permission_override')
)
    EXEC(N'ALTER TABLE dbo.user_permission_override WITH CHECK ADD CONSTRAINT ck_user_permission_override_effect CHECK (effect IN (''ALLOW'',''DENY''));');
GO

IF COL_LENGTH(N'dbo.access_module', N'department_functions') IS NULL ALTER TABLE dbo.access_module ADD department_functions bigint NOT NULL CONSTRAINT df_access_module_department_functions DEFAULT 0;
IF COL_LENGTH(N'dbo.access_module', N'need_specify_department') IS NULL ALTER TABLE dbo.access_module ADD need_specify_department bit NOT NULL CONSTRAINT df_access_module_need_specify_department DEFAULT 0;
IF COL_LENGTH(N'dbo.access_module', N'second_permission_id') IS NULL ALTER TABLE dbo.access_module ADD second_permission_id int NOT NULL CONSTRAINT df_access_module_second_permission_id DEFAULT 0;
IF COL_LENGTH(N'dbo.access_module', N'home_page_url') IS NULL ALTER TABLE dbo.access_module ADD home_page_url nvarchar(512) NULL;
IF COL_LENGTH(N'dbo.access_permission', N'db_role_name') IS NULL ALTER TABLE dbo.access_permission ADD db_role_name nvarchar(128) NULL;
IF COL_LENGTH(N'dbo.access_permission', N'need_second_permission') IS NULL ALTER TABLE dbo.access_permission ADD need_second_permission bit NOT NULL CONSTRAINT df_access_permission_need_second_permission DEFAULT 0;
IF COL_LENGTH(N'dbo.access_role', N'order_sort') IS NULL ALTER TABLE dbo.access_role ADD order_sort int NOT NULL CONSTRAINT df_access_role_order_sort DEFAULT 0;
IF COL_LENGTH(N'dbo.access_role', N'stand_name') IS NULL ALTER TABLE dbo.access_role ADD stand_name nvarchar(128) NULL;
IF COL_LENGTH(N'dbo.permission_mode', N'status') IS NULL
BEGIN
    ALTER TABLE dbo.permission_mode ADD status varchar(16) NOT NULL CONSTRAINT df_permission_mode_status DEFAULT 'ACTIVE';
    EXEC(N'UPDATE dbo.permission_mode SET status=''INACTIVE'' WHERE mode_code=''card'';');
END;
GO

MERGE dbo.access_module AS t USING (VALUES
    ('module-outpatient','outpatient',N'门诊业务',N'', 'ACTIVE',0),
    ('module-master','master',N'系统设置',N'', 'ACTIVE',0)
) AS s(module_id,module_code,module_name,description,status,hidden)
ON t.module_code=s.module_code
WHEN NOT MATCHED THEN INSERT(module_id,module_code,module_name,description,status,hidden) VALUES(s.module_id,s.module_code,s.module_name,s.description,s.status,s.hidden);
MERGE dbo.permission_category AS t USING (VALUES
	('patient',N'患者与挂号',N'','ACTIVE'),('order',N'医嘱与发药',N'','ACTIVE'),('charge',N'收费与退费',N'','ACTIVE'),
	('inventory',N'药品库存',N'','ACTIVE'),('audit',N'操作日志',N'','ACTIVE'),('admin',N'系统管理',N'','ACTIVE')
) AS s(category_id,category_name,description,status)
ON t.category_id=s.category_id
WHEN NOT MATCHED THEN INSERT(category_id,category_name,description,status) VALUES(s.category_id,s.category_name,s.description,s.status);
MERGE dbo.permission_mode AS t USING (VALUES ('password',N'用户名密码','ACTIVE'),('card',N'刷卡（二次认证关闭）','INACTIVE')) AS s(mode_code,mode_name,status)
ON t.mode_code=s.mode_code
WHEN NOT MATCHED THEN INSERT(mode_code,mode_name,status) VALUES(s.mode_code,s.mode_name,s.status);
MERGE dbo.access_role AS t USING (VALUES
    ('REGISTRAR',N'建档挂号员',N'','',NULL,'ACTIVE',0,0),('DOCTOR',N'门诊医生',N'','',NULL,'ACTIVE',0,0),
    ('CASHIER',N'收费员',N'','',NULL,'ACTIVE',0,0),('PHARMACIST',N'药房人员',N'','',NULL,'ACTIVE',0,0),('SYSTEM',N'系统管理员',N'','',NULL,'ACTIVE',0,1)
) AS s(role_code,role_name,description,module_code,department_id,status,hidden,is_system)
ON t.role_code=s.role_code
WHEN NOT MATCHED THEN INSERT(role_code,role_name,description,module_code,department_id,status,hidden,is_system) VALUES(s.role_code,s.role_name,s.description,s.module_code,s.department_id,s.status,s.hidden,s.is_system);

MERGE dbo.role AS t USING (SELECT role_code, role_name FROM dbo.access_role) AS s(role_code, role_name)
ON t.role_code=s.role_code
WHEN MATCHED THEN UPDATE SET role_name=s.role_name
WHEN NOT MATCHED THEN INSERT(role_code, role_name) VALUES(s.role_code, s.role_name);

MERGE dbo.access_permission AS t USING (VALUES
    ('patient:create',N'患者建档',N'','outpatient','patient','password',NULL,'N,S','ACTIVE',0),
    ('encounter:create',N'挂号',N'','outpatient','patient','password',NULL,'N,S','ACTIVE',0),
    ('encounter:check-in',N'签到',N'','outpatient','patient','password',NULL,'M,S','ACTIVE',0),
    ('encounter:cancel',N'退号',N'','outpatient','patient','password',NULL,'D,S','ACTIVE',0),
    ('diagnose:write',N'诊断录入',N'保存、提交和作废门诊诊断',N'outpatient','patient','password',NULL,'N,M,D,S','ACTIVE',0),
    ('order:submit',N'提交医嘱',N'','outpatient','order','password',NULL,'N,S','ACTIVE',0),
    ('order:recipe-check',N'处方审核',N'','outpatient','order','password',NULL,'M,S','ACTIVE',0),
    ('charge:pay',N'收费',N'','outpatient','charge','password',NULL,'N,S','ACTIVE',0),
    ('charge:refund',N'退费',N'','outpatient','charge','password',NULL,'N,S','ACTIVE',0),
    ('dispense:create',N'发药',N'','outpatient','order','password',NULL,'N,S','ACTIVE',0),
    ('dispense:return',N'退药',N'','outpatient','order','password',NULL,'D,S','ACTIVE',0),
    ('inventory:read',N'库存查询',N'','master','inventory','password',NULL,'S','ACTIVE',0),
    ('inventory:adjust',N'库存调整',N'','master','inventory','password',NULL,'M,S','ACTIVE',0),
    ('audit:read',N'操作日志查询',N'','master','audit','password',NULL,'S','ACTIVE',0),
    ('admin:*',N'全部管理权限',N'','master','admin','password',NULL,'N,M,D,S','ACTIVE',0),
    ('admin:user',N'用户管理',N'','master','admin','password',NULL,'N,M,D,S','ACTIVE',0),
    ('admin:role',N'角色管理',N'','master','admin','password',NULL,'N,M,D,S','ACTIVE',0),
    ('admin:permission',N'权限管理',N'','master','admin','password',NULL,'N,M,D,S','ACTIVE',0)
) AS s(permission_code,permission_name,description,module_code,category_id,mode_code,second_permission_code,actions,status,hidden)
ON t.permission_code=s.permission_code
WHEN NOT MATCHED THEN INSERT(permission_code,permission_name,description,module_code,category_id,mode_code,second_permission_code,actions,status,hidden)
VALUES(s.permission_code,s.permission_name,s.description,s.module_code,s.category_id,s.mode_code,s.second_permission_code,s.actions,s.status,s.hidden);

INSERT INTO dbo.access_role_permission (role_code, permission_code)
SELECT s.role_code, s.permission_code
FROM (VALUES
    ('REGISTRAR','patient:create'),('REGISTRAR','encounter:create'),
    ('REGISTRAR','encounter:check-in'),('REGISTRAR','encounter:cancel'),
    ('DOCTOR','patient:create'),('DOCTOR','encounter:check-in'),('DOCTOR','diagnose:write'),('DOCTOR','order:submit'),
    ('CASHIER','patient:create'),('CASHIER','charge:pay'),('CASHIER','charge:refund'),
    ('PHARMACIST','order:recipe-check'),('PHARMACIST','dispense:create'),('PHARMACIST','dispense:return'),('PHARMACIST','inventory:read'),
    ('SYSTEM','admin:*'),('SYSTEM','admin:user'),('SYSTEM','admin:role'),('SYSTEM','admin:permission'),('SYSTEM','inventory:adjust'),('SYSTEM','audit:read')
) AS s(role_code, permission_code)
WHERE NOT EXISTS (
    SELECT 1 FROM dbo.access_role_permission arp
    WHERE arp.role_code=s.role_code AND arp.permission_code=s.permission_code
);
GO
