# P0-D3-ORDER-FEEDBACK-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`6e28083`（全 `6e2808393d164439b0890873163292ece4526b1b`）
- 岗位：H05 · 本机实现（编排器本机落盘；未用 CloudAgent；未 reset/force）

## 实现

| 文件 | 变更 |
| --- | --- |
| `web/src/components/feedback/useSubmitFeedback.js` | **薄加法** `orderId`/`orderNo`；开单动作默认成功文案含流水号；保留 register*/diagnoseId |
| `web/src/components/feedback/SubmitStatusBanner.vue` | 保留 `data-testid=register-submit-feedback`；开单附加 `data-order-testid` |
| `web/src/api/orderApi.js` | create/commit/abate **调用方可选幂等键**；空键 VALIDATION_ERROR；PATCH 不强制键 |
| `web/src/api/mockOrders.js` | create 既有认键表；**补** commit/abate 同键同结果 |
| `web/src/views/OrderView.vue` | 复用 D1/D2 `createSubmitFeedback` + banner；save 持键；**onCommit 双段各持稳定键**；abate 持键；busy 含 submitting |
| `web/scripts/check-order-submit-feedback.mjs` | **新建** 专项验收 |

禁区遵守：未改 Frame / OutpatientQueue* / StatusChain* / Register·Diagnosis 业务接线 / DoctorOrderPanel / Charge等他页 / Go / OpenAPI / orderContract 业务语义。

## 验收

```text
node web/scripts/check-order-submit-feedback.mjs
# => P0_D3_ORDER_FEEDBACK_OK  exit 0

node web/scripts/check-register-submit-feedback.mjs
# => P0_D1_REGISTER_FEEDBACK_OK  exit 0

node web/scripts/check-diagnosis-submit-feedback.mjs
# => P0_D2_DIAGNOSIS_FEEDBACK_OK  exit 0
```

覆盖：延迟锁定｜成功含 orderNo｜重复点同键不双单（save/commit/abate）｜commit 双段各键｜4xx ORDER_NOT_COMMITTABLE｜5xx/网络可恢复｜D1/D2 harness 不回归。

## Git

- feat commit：见后续 tip 戳（不以本文件自指 tip）
- 分支：`codex/register-slot-card-20261028`
- `git push origin HEAD`：见回报
- 未混交无关 dirty

`DONE_P0_D3`
