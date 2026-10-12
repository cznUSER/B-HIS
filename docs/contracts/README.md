# 契约副本

自 H03 同步，供实现对照。以本目录与 `api/openapi.yaml`（0.3.7-draft）为准。

- `接口契约.md` v0.3.7：Auth login / logout / me；开单见 §3.3
- 患者建档见 OpenAPI **0.3.5**（仍在 0.3.7 中）：必填 name/sex/birthday/fareIdentity；address 选填可空；idNumber/cardNo 非空重复 → 409 DUPLICATE_*
- 挂号见 OpenAPI **0.3.6**（仍在 0.3.7 中）主路径 `/registers`（create / list / get / cancel）：`CreateRegisterRequest.required=[patientId,departmentId,doctorId,visitDate]`；初始 status=NORMAL；退号仅 NORMAL→BACK。种子 dept-1/dept-2、doc-1/doc-2。旧 `/encounters*` 为 deprecated alias，当前进程不注册 alias / check-in / complete
- 开单见 OpenAPI **0.3.7-draft** 主路径 `/orders`（save / commit / abate）：强制有效 `registerId`（status≠BACK，否则 409 ENCOUNTER_REQUIRED，拍板#21）；状态 SAVE|COMMIT|PROC|ABATE；一张挂号多处方；diagnoseName 不强制；毒麻钩子关；recipe-check 一期跳过。种子 DRUG-A/DRUG-B、TREAT-1。Go 处理程序由 H04 实现，本同步不含业务 handler
- `数据模型.md`、`库表约束.md`
- `工程约定.md`、`权限与审计.md`、`架构方案草案.md`、`03-用户拍板.md`

这些文件不记录数据库主机、账号或口令。连接只在运行环境的 `BHIS_DB_DSN` 或 gitignore 本地配置中。
