# B-HIS 系统总流程图设计

## 目标

绘制一张面向产品、研发和实施人员的 B-HIS Phase 1 系统总流程图。图稿需要同时说明运行边界、认证与权限、门诊主链、数据落点和失败回退，读者无需先打开源码即可理解系统如何运行。

## 图面结构

采用四条泳道，从上到下依次为：

1. 用户与浏览器：Vue3 登录页、工作台和门诊模块。
2. Host 与 HTTP API：Windows Host、同源 SPA、Go `/api/v1`、请求链和健康检查。
3. 应用服务与领域：JWT/RBAC、应用用例、领域实体、状态机和事务编排。
4. 存储与可观测性：Store 工厂、Memory/SQL Server、库存账、业务审计和运行日志。

## 主流程

登录成功后取得 Bearer JWT，经当前用户和权限范围解析进入工作台。门诊闭环按“患者建档 → 挂号 → 接诊/诊断 → 开单 → 提交 → 收费/支付 → 发药 → 退药 → 退费/完成就诊”展示。状态节点标出 `NORMAL`、`DIAGNOSING`、`DIAGNOSED`、`SAVE`、`COMMIT`、`PROC`、`ABATE` 等关键状态。

收费只做库存校验，不预扣；发药在事务内扣库存并写 `DISPENSE` ledger；退药在事务内回库存并写 `RETURN` ledger；已发药的费用必须先完成退药才能进入退费。退费标出旧 Charge 作废、部分退费 successor 和退款 Payment 的同事务关系。

## 旁路流程

在主链旁标出基础数据维护（科室、医生、目录、字典）、用户/角色/权限与科室范围、库存查询与 `ADJUST`、审计查询、Dashboard 和 `/health`。这些旁路统一经过应用服务和权限检查。

## 失败与回退

统一标注 `400/401/403/409/429/501/503` 错误和 `requestId` envelope。未认证/无权限回登录或拒绝；校验、重复、非法状态、库存不足、未收费发药、超量退药和已发药退费前置失败时，状态不推进；涉及库存、收费、退药、退费或用户权限整屏保存的事务失败时回滚。`recipe-check` 保留 `501 NOT_IMPLEMENTED`。

## 版本与证据

流程图以当前代码和 OpenAPI `0.3.11-draft` 为准，证据包括：

- `internal/bootstrap/run.go`、`internal/adapter/http/auth_handler.go`、`internal/webui/webui.go`
- `internal/app` 下 patient/register/order/charge/dispense/diagnose 用例
- `internal/store/factory.go` 与 memory/SQL Server Store
- `web/src/router/index.js`、`web/src/workbench/catalog.js`、`web/src/views/OutpatientModuleView.vue`
- `docs/contracts/架构方案草案.md`、`docs/contracts/接口契约.md`、`docs/contracts/权限与审计.md`

README 中关于版本和“完成就诊/诊断未注册”的描述已滞后，图注需明确“以当前代码/OpenAPI 为准”。

## 交付形态

交付一个独立 Codex Canvas 文件，内嵌全部节点和连线，不引入外部请求。画布应提供图例、版本注记和证据说明，便于在聊天旁打开查看。
