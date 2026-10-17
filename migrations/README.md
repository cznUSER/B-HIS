# 迁移

Phase1 只维护 SQL Server 脚本。目标库名 **BHIS**。

契约 tip：**v0.3.11-draft**（OpenAPI **0.3.11-draft**，P2-0A 用户角色科室范围）。迁移清单包括门诊主流程、基础资料、用户权限、用户操作日志与用户角色科室范围。

| 文件 | 作用 |
|------|------|
| `sqlserver/core/001_identity.up.sql` | 创建数据库 BHIS（若无）、身份表、角色、演示账号 `system` 与 `disabled` 的 bcrypt 哈希。明文不写入脚本；员工默认密码见根 README |
| `sqlserver/core/001_identity.down.sql` | 删除身份表，不删除数据库 |
| `sqlserver/core/002_patient.up.sql` | 患者/MPI 表 `dbo.patient`（依赖 BHIS；varchar(64) 主键；证件号、卡号、院内主键过滤唯一索引）。不建库 |
| `sqlserver/core/002_patient.down.sql` | 删除患者表，不删除数据库 |
| `sqlserver/core/003_register.up.sql` | 科室 `dbo.department`、医生 `dbo.doctor`、挂号单 `dbo.register_bill`（依赖 BHIS 与 `dbo.patient`；varchar(64) 主键；号源=科室+医生，无排班表）。不建库 |
| `sqlserver/core/003_register.down.sql` | 按外键依赖删除挂号单、医生、科室表，不删除数据库 |
| `sqlserver/core/003_register_seed.up.sql` | 幂等种子：科室 `dept-1`/`dept-2`、医生 `doc-1`/`doc-2`。无口令、无连接串 |
| `sqlserver/core/004_order.up.sql` | 目录 `dbo.catalog_item`、处方头 `dbo.order_sheet`、明细 `dbo.order_item`（依赖 BHIS、`dbo.patient` 与 `003_register` 的科室/医生/挂号单；varchar(64) 主键；`register_id` ≡ encounter；一张挂号多处方）。不建库 |
| `sqlserver/core/004_order.down.sql` | 按外键依赖删除明细、处方头、目录表，不删除数据库 |
| `sqlserver/core/004_order_seed.up.sql` | 幂等种子：目录 `DRUG-A`/`DRUG-B`/`TREAT-1`。不含凭据 |
| `sqlserver/core/005_charge.up.sql` | 收费单 `dbo.charge`、明细 `dbo.charge_item`、支付 `dbo.payment`（依赖 BHIS、`004_order` 的处方头/明细、`003_register` 的挂号单与 `002_patient`；varchar(64) 主键；`register_id` 必填，语义 ≡ encounter）。不建库 |
| `sqlserver/core/005_charge.down.sql` | 按外键依赖删除 payment、charge_item、charge，不删除数据库 |
| `sqlserver/core/005_charge_seed.up.sql` | 空种子：不插入演示收费或支付。无连接串 |
| `sqlserver/core/006_inventory.up.sql` | 库存 `dbo.inventory_item`、流水 `dbo.inventory_ledger`（依赖 BHIS；`item_no` 对齐 `004_order` 药品目录；varchar(64) 主键；简易数量账；`qty_on_hand`≥0；ledger 只追加，发药负、退药正）。不建库。TREAT 不入表 |
| `sqlserver/core/006_inventory.down.sql` | 按外键依赖删除 inventory_ledger、inventory_item，不删除数据库 |
| `sqlserver/core/006_inventory_seed.up.sql` | 幂等种子：`DRUG-A`/`DRUG-B`（sku-drug-a / sku-drug-b）`qty_on_hand=100`。已存在行不重置数量。TREAT-1 不入。无口令、无连接串 |
| `sqlserver/core/007_dispense.up.sql` | 发药单 `dbo.dispense`、明细 `dbo.dispense_item`（依赖 002 患者、003 挂号、004 处方、005 收费、006 库存；仅 DRUG；部分发；行态 NONE\|PARTIAL\|FULL）。不建库。无独立种子 |
| `sqlserver/core/007_dispense.down.sql` | 按外键依赖删除 dispense_item、dispense，不删除数据库 |
| `sqlserver/core/008_master.up.sql` | P1-H03-MASTER：`department.sort_no`、`catalog_item.is_controlled`、`dict_item`（号别种子 `register_sort` / `sort-normal`）。库存 ADJUST 复用 006 的 `inventory_ledger`（无新流水表）。无连接串、无口令 |
| `sqlserver/core/008_master.down.sql` | 回滚 008：删除 `dict_item`，去掉 `is_controlled` 与 `sort_no`，不删除数据库 |
| `sqlserver/core/002_access_control.up.sql` | 用户资料与账号标记、角色排序字段、模块/权限分类/认证方式/权限/角色/用户例外目录；依赖 001 的用户和角色表。刷卡方式只保存为元数据，默认关闭，不提供刷卡登录 |
| `sqlserver/core/002_access_control.down.sql` | 回滚用户权限目录及新增资料字段；会删除该迁移创建的权限配置，请确认后再用于回滚 |
| `sqlserver/core/009_user_operations_audit.up.sql` | 为 `audit_log` 增加用户操作日志字段，并为已有审计行补充可推导的模块、用户名与说明；依赖 001 的 `audit_log` 和 `app_user` |
| `sqlserver/core/009_user_operations_audit.down.sql` | 删除 009 新增的审计列；回滚会移除这些列数据 |
| `sqlserver/core/010_diagnose.up.sql` | 可选诊断实体 `dbo.diagnose`（挂在 `register_bill`/encounter 上；SAVE/COMMIT/ABATE 生命周期）。不启用临床规则 |
| `sqlserver/core/010_diagnose.down.sql` | 删除可选诊断表，不删除数据库 |
| `sqlserver/core/011_user_role_scope.up.sql` | 为已分配用户角色配置显式科室范围；任意显式行（包括 DISABLED）抑制该角色的旧单科室回退；依赖 002_access_control 与 003_register |
| `sqlserver/core/011_user_role_scope.down.sql` | 删除用户角色科室范围表，不删除数据库 |
| `sqlserver/core/014_diagnose_permission.up.sql` | 为已有 BHIS 数据库补充独立诊断录入权限，并授予 DOCTOR；幂等，不启用临床规则 |
| `sqlserver/core/014_diagnose_permission.down.sql` | 删除诊断录入权限及角色映射，不删除数据库 |
| `sqlserver/core/012_inventory_batch_readonly.up.sql` | 历史隔离批次快照原型；拒绝 BHIS 和 HospDataV9，最终本机 BHIS 使用 015 |
| `sqlserver/core/013_idempotency_record.up.sql` | 持久化幂等记录；脚本内固定 USE BHIS，隔离库使用经核对的隔离版本，不可只靠调用参数切库 |
| `sqlserver/core/015_local_inventory_batch_readonly.up.sql` | 只允许本机开发目标 BHIS：创建空批次快照表及查询权限；不导入数据、不改变库存总量账；同事务、可重复执行 |
| `sqlserver/core/015_local_inventory_batch_readonly.down.sql` | 仅回滚 015 创建的空表及其新建权限/药师授权；已有表、非空表或新增权限依赖拒绝回滚；不删除迁移前已有权限 |
| `sqlserver/core/016_patient_registration_detail.up.sql` | 为 `dbo.patient` 增加会员、出生地、民族国籍、婚姻职业学历、工作单位、户籍与联系人资料；依赖 002 患者表 |
| `sqlserver/core/016_patient_registration_detail.down.sql` | 回滚 016 新增的患者详细建档资料列 |

脚本给 `sqlcmd` 用（含 `GO`）。密码不要写进命令行历史或仓库：让 `sqlcmd` 从环境变量 `SQLCMDPASSWORD` 读取。

`app_user.username` 上的过滤唯一索引要求 `QUOTED_IDENTIFIER ON`。Windows 优先走 `scripts/apply-migration.ps1`（已带 `sqlcmd -I`，口令只读 `SQLCMDPASSWORD`）。手写 `sqlcmd` 也必须带 `-I`，否则对该索引的 `INSERT` / `UPDATE` 会失败。

`002_patient` 里的过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；调用 `sqlcmd` 时仍要带 `-I`，与身份迁移一致。不要改这份已提交的 DDL。

`003_register` 的科室编码、医生编码、挂号单号过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。种子在表创建之后执行。

`004_order` 依赖 `003_register`（`register_bill` / `department` / `doctor`）以及 `002_patient`。目录项目编码、处方单号过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。种子 `004_order_seed.up.sql` 在表创建之后执行（`DRUG-A` / `DRUG-B` / `TREAT-1`）。

`005_charge` 依赖 `004_order`（`order_sheet` / `order_item`）以及挂号单与患者表。`square_no` 过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。`005_charge_seed.up.sql` 在表创建之后执行，故意不插入演示收费或支付（运行时从已 COMMIT 处方生成）。本片不改 `001`/`002`/`003`/`004`。

`006_inventory` 依赖 BHIS，药品编码对齐 `004_order` 目录（`DRUG-A` / `DRUG-B`）。`item_no` 过滤唯一索引依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。种子在表创建之后执行。`007_dispense` 依赖患者、挂号、处方、收费与库存表；`dispense_no` 过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。发药只记 DRUG；收费侧只拦库存、不预扣，扣减与回加由后续发药事务写 ledger。本片只落表，不改 `001`～`005`，也不注册 Go handler。

`008_master` 依赖 001（BHIS / `app_user`）、003（`department`）、004（`catalog_item`）、006（`inventory_item` / `inventory_ledger`）。补 `department.sort_no`、`catalog_item.is_controlled`（默认关），新建 `dict_item` 并种子号别 `register_sort` / `sort-normal`。库存调整复用 `inventory_ledger.biz_type=ADJUST`（不新造 INIT，不建调整单头）。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。本片只落表，不注册基础数据 Go handler。

```text
powershell -ExecutionPolicy Bypass -File scripts/apply-migration.ps1 -Server <host> -User <user>
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/001_identity.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/002_patient.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/003_register.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/003_register_seed.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/004_order.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/004_order_seed.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/005_charge.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/005_charge_seed.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/006_inventory.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/006_inventory_seed.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/007_dispense.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/008_master.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/002_access_control.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/009_user_operations_audit.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/011_user_role_scope.up.sql
sqlcmd -S <host> -U <user> -I -f 65001 -d master -i migrations/sqlserver/core/016_patient_registration_detail.up.sql
```

API 默认 `Database.Driver=memory`，不连 SQL Server。切到 `sqlserver` 时，DSN 放在环境变量 `BHIS_DB_DSN` 或 gitignore 的 `config.local.yaml` / `.env.local`。
