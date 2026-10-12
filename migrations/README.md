# 迁移

Phase1 只维护 SQL Server 脚本。目标库名 **BHIS**。

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

脚本给 `sqlcmd` 用（含 `GO`）。密码不要写进命令行历史或仓库：让 `sqlcmd` 从环境变量 `SQLCMDPASSWORD` 读取。

`app_user.username` 上的过滤唯一索引要求 `QUOTED_IDENTIFIER ON`。Windows 优先走 `scripts/apply-migration.ps1`（已带 `sqlcmd -I`，口令只读 `SQLCMDPASSWORD`）。手写 `sqlcmd` 也必须带 `-I`，否则对该索引的 `INSERT` / `UPDATE` 会失败。

`002_patient` 里的过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；调用 `sqlcmd` 时仍要带 `-I`，与身份迁移一致。不要改这份已提交的 DDL。

`003_register` 的科室编码、医生编码、挂号单号过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。种子在表创建之后执行。

`004_order` 依赖 `003_register`（`register_bill` / `department` / `doctor`）以及 `002_patient`。目录项目编码、处方单号过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。种子 `004_order_seed.up.sql` 在表创建之后执行（`DRUG-A` / `DRUG-B` / `TREAT-1`）。

`005_charge` 依赖 `004_order`（`order_sheet` / `order_item`）以及挂号单与患者表。`square_no` 过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。`005_charge_seed.up.sql` 在表创建之后执行，故意不插入演示收费或支付（运行时从已 COMMIT 处方生成）。本片不改 `001`/`002`/`003`/`004`。

`006_inventory` 依赖 BHIS，药品编码对齐 `004_order` 目录（`DRUG-A` / `DRUG-B`）。`item_no` 过滤唯一索引依赖 `QUOTED_IDENTIFIER ON`。脚本内已 `SET QUOTED_IDENTIFIER ON`；`sqlcmd` 仍要带 `-I`。种子在表创建之后执行。`007_dispense` 依赖患者、挂号、处方、收费与库存表；`dispense_no` 过滤唯一索引同样依赖 `QUOTED_IDENTIFIER ON`。发药只记 DRUG；收费侧只拦库存、不预扣，扣减与回加由后续发药事务写 ledger。本片只落表，不改 `001`～`005`，也不注册 Go handler。

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
```

API 默认 `Database.Driver=memory`，不连 SQL Server。切到 `sqlserver` 时，DSN 放在环境变量 `BHIS_DB_DSN` 或 gitignore 的 `config.local.yaml` / `.env.local`。
