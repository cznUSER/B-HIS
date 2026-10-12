# 契约副本

自 H03 同步，供实现对照。以本目录与 `api/openapi.yaml`（0.3.7-draft）为准。

- `接口契约.md` v0.3.2：Auth login / logout / me；§3.3 开单已对齐 **0.3.7-draft**
- 患者建档见 OpenAPI **0.3.5**（仍在 0.3.7 中）：必填 name/sex/birthday/fareIdentity；address 选填可空；idNumber/cardNo 非空重复 → 409 DUPLICATE_*
- 挂号见 OpenAPI **0.3.6**（仍在 0.3.7 中）主路径 `/registers`（create / list / get / cancel）：`CreateRegisterRequest.required=[patientId,departmentId,doctorId,visitDate]`；初始 status=NORMAL；退号仅 NORMAL→BACK。种子 dept-1/dept-2、doc-1/doc-2。旧 `/encounters*` 为 deprecated alias，当前进程不注册 alias / check-in / complete
- 开单见 OpenAPI **0.3.7-draft** 与《接口契约》§3.3：主路径 `/orders`（POST 创建 SAVE、GET 列表/详情、PATCH 仅 SAVE、`POST /orders/{orderId}/commit`、`POST /orders/{orderId}/abate`）。`/submit` 为 commit 的 deprecated 别名；recipe-check 一期跳过。主字段 `registerId`（`encounterId` ≡ `registerId`）。强制有效挂号（status≠BACK），退号 BACK 拒开 → `ENCOUNTER_REQUIRED`。种子 DRUG-A/DRUG-B/TREAT-1。本仓不注册 Go `/orders` handler
- `工程约定.md`、`权限与审计.md`、`架构方案草案.md`、`03-用户拍板.md`

这些文件不记录数据库主机、账号或口令。连接只在运行环境的 `BHIS_DB_DSN` 或 gitignore 本地配置中。
