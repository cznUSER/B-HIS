# P0-D5-DISPENSE-FEEDBACK-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`7473271`
- 岗位：H05 · 本机实现（编排器本机落盘；未用 CloudAgent；未 reset/force）

## 实现

| 文件 | 变更 |
| --- | --- |
| `web/src/components/feedback/useSubmitFeedback.js` | **薄扩展** `dispenseId`/`dispenseNo`；dispense/return 成功文案含发药单号；保留 register*/diagnoseId/order*/chargeId/squareNo |
| `web/src/components/feedback/SubmitStatusBanner.vue` | 保留既有 testid；发药附加 `data-dispense-testid` |
| `web/src/api/dispenseApi.js` | 多 order 派生键 `${base}-n` 注释对齐；return 传键薄对齐 |
| `web/src/api/mockDispenses.js` | confirm/return 认键表；**幂等查找先于**业务拒绝；同键同结果；冲突体 `IDEMPOTENCY_KEY_CONFLICT` |
| `web/src/views/DispenseView.vue` | 复用 D1–D4 `createSubmitFeedback` + banner；confirm/return 调用方持稳定键；部分发药成功可解释；busy ⊇ submitting |
| `web/scripts/check-dispense-submit-feedback.mjs` | **新建** 专项验收 |

禁区遵守：未改 Frame / OutpatientQueue* / StatusChain* / Register·Diagnosis·Order·Charge 业务接线 / Refund 壳 / Go / OpenAPI；未平行栈。

## 多 order 派生键

真路径 `confirmDispense`：单组用调用方键；多 order 分组时每组 `Idempotency-Key = ${given}-${index+1}`（1-based）。Mock 侧通常一组，同键同结果复放。Harness 以派生函数断言 + Mock 同键复放覆盖。

## 验收

```text
node web/scripts/check-dispense-submit-feedback.mjs
# => P0_D5_DISPENSE_FEEDBACK_OK  exit 0

node web/scripts/check-register-submit-feedback.mjs
# => P0_D1_REGISTER_FEEDBACK_OK  exit 0

node web/scripts/check-diagnosis-submit-feedback.mjs
# => P0_D2_DIAGNOSIS_FEEDBACK_OK  exit 0

node web/scripts/check-order-submit-feedback.mjs
# => P0_D3_ORDER_FEEDBACK_OK  exit 0

node web/scripts/check-charge-submit-feedback.mjs
# => P0_D4_CHARGE_FEEDBACK_OK  exit 0
```

覆盖：延迟锁定｜成功含 dispenseNo｜重复点同键不双单（confirm + return）｜4xx｜5xx/网络可恢复｜部分发药文案｜D1–D4 harness 不回归。

## Git

- feat commit：见后续 tip 戳（不以本文件自填 tip）
- 分支：`codex/register-slot-card-20261028`
- `git push origin HEAD`：见回报
- 未混交无关 dirty

`DONE_P0_D5`
