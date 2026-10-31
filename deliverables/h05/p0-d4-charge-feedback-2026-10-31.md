# P0-D4-CHARGE-FEEDBACK-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`d5240c1`（全 `d5240c1` 见 git）
- 岗位：H05 · 本机实现（编排器本机落盘；未用 CloudAgent；未 reset/force）

## 实现

| 文件 | 变更 |
| --- | --- |
| `web/src/components/feedback/useSubmitFeedback.js` | **薄扩展** `chargeId`/`squareNo`；pay/refund 成功文案含收费单号；保留 register*/diagnoseId/order* |
| `web/src/components/feedback/SubmitStatusBanner.vue` | 保留既有 testid；收费附加 `data-charge-testid` |
| `web/src/api/chargeApi.js` | create 强制 Idempotency-Key；`payCharge` 接受 `createIdempotencyKey` + `idempotencyKey` 双段；refund 透传键 |
| `web/src/api/mockCharges.js` | pay/refund 认键表；**幂等查找先于** ALREADY_CHARGED / ALREADY_ABATED |
| `web/src/views/ChargeView.vue` | 复用 D1–D3 `createSubmitFeedback` + banner；`onPay` 双段各持稳定键；退费面板同片接线；busy ← submitting |
| `web/scripts/check-charge-submit-feedback.mjs` | **新建** 专项验收 |

禁区遵守：未改 Frame / OutpatientQueue* / StatusChain* / Register·Diagnosis·Order 业务接线 / Dispense / Refund 壳 / Go / OpenAPI / 挂号费旁路；未平行栈。

## 验收

```text
node web/scripts/check-charge-submit-feedback.mjs
# => P0_D4_CHARGE_FEEDBACK_OK  exit 0

node web/scripts/check-register-submit-feedback.mjs
# => P0_D1_REGISTER_FEEDBACK_OK  exit 0

node web/scripts/check-diagnosis-submit-feedback.mjs
# => P0_D2_DIAGNOSIS_FEEDBACK_OK  exit 0

node web/scripts/check-order-submit-feedback.mjs
# => P0_D3_ORDER_FEEDBACK_OK  exit 0
```

覆盖：延迟锁定｜成功含 squareNo｜重复点同键不双单（pay 双段 + refund）｜4xx ALREADY_CHARGED｜5xx/网络可恢复｜D1–D3 harness 不回归。

## Git

- feat commit：见后续 tip 戳（不以本文件自报 tip）
- 分支：`codex/register-slot-card-20261028`
- `git push origin HEAD`：见回报
- 未混交无关 dirty

`DONE_P0_D4`
