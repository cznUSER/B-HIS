# 契约副本

自 H03 同步，供实现对照。契约 tip 为 **0.3.10-draft** / **P1-H03-MASTER**（`api/openapi.yaml`：科室/医生/目录/库存调整/字典/用户）。已实现进程 `GET /health` 的 `contractVersion`、`implementedVersion` 与 `version` 均为 **0.3.10-draft**（登录、建档、挂号、开单、收费、发药/退药、库存查询与调整、基础数据维护）。Go Charge/Payment 与 Dispense/Inventory handler 已注册。`POST /orders/{orderId}/dispense` 为发药别名。`recipe-check` 仍返回 501。

- `接口契约.md`：Auth 仍为 v0.3.2；§3.3 开单对齐 **0.3.7**；§3.4 收费对齐 **0.3.8**（仍含于 0.3.10-draft）；§3.5 发药 / §3.7 库存对齐 **0.3.9**；基础数据对齐 **0.3.10-draft**
- 患者建档见 OpenAPI **0.3.5**（仍在 0.3.8 中）：必填 name/sex/birthday/fareIdentity；address 选填可空；idNumber/cardNo 非空重复 → 409 DUPLICATE_*
- 挂号见 OpenAPI **0.3.6**（仍在 0.3.8 中）主路径 `/registers`（create / list / get / check-in / cancel）：`CreateRegisterRequest.required=[patientId,departmentId,doctorId,visitDate]`；初始 status=NORMAL；接诊仅 NORMAL 或 USED→DIAGNOSING，否则 409 ILLEGAL_STATE_TRANSITION；退号仅 NORMAL→BACK。种子 dept-1/dept-2、doc-1/doc-2。旧 `/encounters*` 为 deprecated alias，当前进程不注册 alias / complete
- 开单见 OpenAPI **0.3.7**（仍在 0.3.8 中）与《接口契约》§3.3：主路径 `/orders`（POST 创建 SAVE、GET 列表/详情、PATCH 仅 SAVE、`POST /orders/{orderId}/commit`、`POST /orders/{orderId}/abate`）。`/submit` 为 commit 的 deprecated 别名；recipe-check 一期跳过。主字段 `registerId`（`encounterId` ≡ `registerId`）。强制有效挂号（status≠BACK），退号 BACK 拒开 → `ENCOUNTER_REQUIRED`。种子 DRUG-A/DRUG-B/TREAT-1。Go `/orders` 主路径已注册；`recipe-check` 返回 501；`dispense` 为发药别名
- 收费见 OpenAPI **0.3.8**（仍含于 **0.3.9-draft**）与《接口契约》§3.4：`GET /orders/{orderId}/charge-preview`、`POST /charges`、`GET /charges`、`GET /charges/{chargeId}`、`POST /charges/{chargeId}/pay`、`POST /charges/{chargeId}/refund`。主字段 `registerId`（`encounterId` ≡ `registerId`）。硬约束：`ENCOUNTER_REQUIRED`；现金一次付清；F=0 主路径；不拆方；全退 `newSquare=null`；收费**只拦不预扣** `INSUFFICIENT_STOCK`；先退药 `REFUND_REQUIRES_RETURN`。迁移 `005_charge`（空种子）。Go 已注册这些收费路径
- 发药见 OpenAPI **0.3.9-draft** 与《接口契约》§3.5：`POST /dispenses`（`POST /orders/{orderId}/dispense` 为 deprecated 别名）、`POST /dispenses/{dispenseId}/return`、`GET /dispenses`、`GET /dispenses/{dispenseId}`。仅 DRUG；须 Charge `NORMAL` 且已支付，否则 `NOT_CHARGED`。部分发；行态 `NONE`|`PARTIAL`|`FULL`。发药事务扣减，不足 `INSUFFICIENT_STOCK`；退药事务回加。退费前置：净发>0 → `REFUND_REQUIRES_RETURN`。错误码收口：`ALREADY_DISPENSED`（≡ H02 `ALREADY_FULLY_DISPENSED`）、`DISPENSE_EXCEEDS_AVAILABLE`、`EMPTY_DISPENSE_ITEMS`、`NOT_DISPENSED`、`RETURN_EXCEEDS_DISPENSE`、`RETURN_REASON_REQUIRED`
- 库存见《接口契约》§3.7：`GET /inventory/items`；迁移 `006_inventory`（种子 DRUG-A/B qty=100，TREAT 不入）+ `007_dispense`。方案 A 简易数量账。Go 已注册，与发药共用 `qty_on_hand`
- 基础数据见 OpenAPI **0.3.10-draft** / MASTER：`/departments`、`/doctors`、`/catalog-items`、`POST /inventory/items/{skuId}/adjust`（仅 `ADJUST`；权限 `inventory:adjust`；审计 `inventory.adjust`）、`/dicts`、`/users`。迁移 `008_master`（`sort_no`、`is_controlled`、`dict_item`）。Go 已注册这些 handler（memory + sqlserver）
- `工程约定.md`、`权限与审计.md`、`数据模型.md`、`库表约束.md`、`假设与随H02修订记录.md`、`架构方案草案.md`、`03-用户拍板.md`

这些文件不记录数据库主机、账号或口令。连接只在运行环境的 `BHIS_DB_DSN` 或 gitignore 本地配置中。
