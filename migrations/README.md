# 迁移

Phase1 只维护 SQL Server 脚本。目标库名 **BHIS**。

| 文件 | 作用 |
|------|------|
| `sqlserver/core/001_identity.up.sql` | 创建数据库 BHIS（若无）、身份表、角色、演示账号的 bcrypt 哈希 |
| `sqlserver/core/001_identity.down.sql` | 删除身份表，不删除数据库 |

脚本给 `sqlcmd` 用（含 `GO`）。密码不要写进命令行历史或仓库：让 `sqlcmd` 从环境变量 `SQLCMDPASSWORD` 读取。

```text
sqlcmd -S <host> -U <user> -f 65001 -d master -i migrations/sqlserver/core/001_identity.up.sql
```

API 默认 `Database.Driver=memory`，不连 SQL Server。切到 `sqlserver` 时，DSN 放在环境变量 `BHIS_DB_DSN` 或 gitignore 的 `config.local.yaml` / `.env.local`。
