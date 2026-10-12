# 契约副本

自 H03 同步，供实现对照。契约文件 `api/openapi.yaml` 为 **0.3.8-draft**。已实现进程 `GET /health` 报 **0.3.8-draft**（登录、建档、挂号、开单、收费）。Go Charge/Payment handler 已注册。

- `接口契约.md`：Auth 仍为 v0.3.2；§3.3 开单对齐 **0.3.7**；§3.4 收费对齐 **0.3.8-draft**
- 患者建档见 OpenAPI **0.3.5**（仍在 0.3.8 中）：必填 name/sex/birthday/fareIdentity；address 选填可空；idNumber/cardNo 非空重复 → 409 DUPLICATE_*
- 挂号见 OpenAPI **0.3.6**（仍在 0.3.8 中）主路径 `/registers`（create / list / get / check-in / cancel）：`CreateRegisterRequest.required=[patientId,departmentId,doctorId,visitDate]`；初始 status=NORMAL；接诊仅 NORMAL 或 USED→DIAGNOSING，否则 409 ILLEGAL_STATE_TRANSITION；退号仅 NORMAL→BACK。种子 dept-1/dept-2、doc-1/doc-2。旧 `/encounters*` 为 deprecated alias，当前进程不注册 alias / complete
- 开单见 OpenAPI **0.3.7**（仍在 0.3.8 中）与《接口契约》§3.3：主路径 `/orders`（POST 创建 SAVE、GET 列表/详情、PATCH 仅 SAVE、`POST /orders/{orderId}/commit`、`POST /orders/{orderId}/abate`）。`/submit` 为 commit 的 deprecated 别名；recipe-check 一期跳过。主字段 `registerId`（`encounterId` ≡ `registerId`）。强制有效挂号（status≠BACK），退号 BACK 拒开 → `ENCOUNTER_REQUIRED`。种子 DRUG-A/DRUG-B/TREAT-1。Go `/orders` 主路径已注册；`recipe-check`、`dispense` 返回 501
- 收费见 OpenAPI **0.3.8-draft** 与《接口契约》§3.4：`GET /orders/{orderId}/charge-preview`、`POST /charges`、`GET /charges`、`GET /charges/{chargeId}`、`POST /charges/{chargeId}/pay`、`POST /charges/{chargeId}/refund`。主字段 `registerId`（`encounterId` ≡ `registerId`）。硬约束：`ENCOUNTER_REQUIRED`；现金一次付清；F=0 主路径；不拆方；全退 `newSquare=null`；收费拦库存 `INSUFFICIENT_STOCK`；先退药 `REFUND_REQUIRES_RETURN`。迁移 `005_charge`（空种子）。Go 已注册这些收费路径
- `工程约定.md`、`权限与审计.md`、`架构方案草案.md`、`03-用户拍板.md`

这些文件不记录数据库主机、账号或口令。连接只在运行环境的 `BHIS_DB_DSN` 或 gitignore 本地配置中。
