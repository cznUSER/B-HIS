# P1-PRINT-STATUS-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`649cb13`（全 `649cb1388d355e6dfce5013bcc56332f1575d8db`；本机 ahead 1）
- 岗位：H05 · 本机实现（Codex 补丁编码受阻后改本机落盘；未用 CloudAgent；未 reset/force）

## 目标

门诊开单页（`OrderView`）为已提交/可输出处方增加可追踪输出状态：预览 + `IDLE|PREVIEW|PRINTING|PRINTED|FAILED` + 失败重试/重打 UI。无真实打印服务时使用 `window.print()` +「确认已输出」占位，**诚实声明未对接打印服务、未回写服务端打印流水**。

## 文件列表

| 文件 | 变更 |
| --- | --- |
| `web/src/workbench/orderPrintStatus.js` | **新建** 纯前端状态机 + `buildPreviewModel`（防串患）+ 诚实文案常量 |
| `web/src/components/outpatient/OrderPrintPanel.vue` | **新建** 预览面板 + 状态/按钮互斥 + scoped CSS |
| `web/src/views/OrderView.vue` | **最小胶水**：挂面板；COMMIT/PROC 可预览；切换处方/新开 reset；可选 `recordAction('打印'|'重打')` |
| `web/scripts/check-order-print-status.mjs` | **新建** 状态迁移 / 重试 / 重打不改 order.status / 不串患 / 诚实文案 / 静态接线 |
| `deliverables/h05/p1-print-status-2026-10-31.md` | 本交付 |

禁区遵守：未改 `outpatientContext` 写入语义、Frame/Queue/StatusChain、`useSubmitFeedback`/`SubmitStatusBanner`、他岗 View/harness、DoctorOrderPanel、Go/OpenAPI。

## 状态机

```text
IDLE → PREVIEW（打开预览，绑定当前 order）
PREVIEW → PRINTING（本机打印占位 / window.print）
PRINTING → PRINTED（用户确认已输出；记 lastPrintedAt）
PRINTING → FAILED（占位异常）
FAILED → PRINTING（失败重试；不新建 order）
PRINTED → PRINTING（重打；order.status 快照不变）
任意非 PRINTING → IDLE（收起 / 切换处方 / 新开 / 刷新后会话未持久化）
```

中文文案：未输出 | 预览中 | 正在输出… | 本机已确认输出 | 输出失败。

## 诚实占位说明

- UI / 常量明示：`本机占位输出：未对接打印服务，未回写服务端打印流水。刷新后打印态不持久化。`
- `SESSION_PERSISTENCE_NOTE`：`本机会话未持久化打印态；刷新后回到未输出。`
- **禁止**文案：已对接打印服务 / 已回写服务端打印流水 / 服务端已打印。
- `window.print()` 仅本机占位；最终以用户「确认已输出」进入 PRINTED。
- 重打只增加前端尝试与可选 `recordAction`，**不**改业务 `order.status`、不重新 commit。

## Harness

| 命令 | exit | 摘要 |
| --- | --- | --- |
| `node web/scripts/check-order-print-status.mjs` | 0 | `P1_PRINT_STATUS_OK` |
| `node web/scripts/check-order-submit-feedback.mjs` | 0 | `P0_D3_ORDER_FEEDBACK_OK`（D3 不回归） |

## 残余

- 无真实打印微服务 / 设备选择 / 服务端打印审计（属后续形态突破，本片不作准出）。
- Charge/Refund/医生站套打未纳入。
- origin 若 443 失败以本机 tip + 本交付为准。
- 工作区仍有大量无关 dirty，本片仅 stage 上表文件。

`DONE_P1_PRINT_STATUS`