# 契约副本

自 H03 同步，供实现对照。以本目录与 `api/openapi.yaml`（0.3.2-draft）为准。

- `接口契约.md` v0.3.2：Auth login / logout / me
- 患者建档见 OpenAPI **0.3.5-draft**：必填 name/sex/birthday/fareIdentity；address 选填可空；idNumber/cardNo 非空重复 → 409 DUPLICATE_*
- `工程约定.md`、`权限与审计.md`、`架构方案草案.md`、`03-用户拍板.md`

这些文件不记录数据库主机、账号或口令。连接只在运行环境的 `BHIS_DB_DSN` 或 gitignore 本地配置中。
