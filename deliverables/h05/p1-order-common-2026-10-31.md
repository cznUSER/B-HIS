# P1-ORDER-COMMON-01 · 开单页常用项目芯片

- 日期：2026-10-31（Asia/Shanghai）
- 岗位：H05
- 基线 tip：`4243b19`
- 分支：`codex/register-slot-card-20261028`

## 文件列表

| 路径 | 动作 |
|---|---|
| `web/src/workbench/commonOrderItems.js` | 新建：演示短表常量 + resolve/apply 纯函数 |
| `web/src/views/OrderView.vue` | 接线：常用芯片条 + 锁定禁用 + 缺失诚实提示 |
| `web/scripts/check-order-common.mjs` | 新建：本片 harness |
| `web/scripts/check-order-item-find.mjs` | 共存最小改：OrderView 允许「常用项目」；Finder 仍禁「常用」 |
| `deliverables/h05/p1-order-common-2026-10-31.md` | 本交付 |

## 行为

1. 找项区下方「常用项目」芯片条；明示「演示短表（非个人收藏）」。
2. 短表锚定 mock CATALOG：`DRUG-A` / `DRUG-B` / `TREAT-1`。
3. 点击可用芯片 → 填入首个空行 `itemNo`；无空行则 `addLine` 等价新增后再填；不自动 save/commit。
4. 远程态按 `findItems`（ACTIVE 目录）解析；缺失 → disabled + `order-common-missing` 提示；不写幽灵行。
5. `locked || busy || !canWrite` 时不可点。
6. 零新 API；无 localStorage/sessionStorage 落常用。

## Harness

| 脚本 | 退出码 | 输出标记 |
|---|---|---|
| `node web/scripts/check-order-common.mjs` | 0 | `P1_ORDER_COMMON_OK` |
| `node web/scripts/check-order-item-find.mjs` | 0 | `P1_ORDER_ITEM_FIND_OK` |
| `node web/scripts/check-order-print-status.mjs` | 0 | `P1_PRINT_STATUS_OK` |
| `node web/scripts/check-order-submit-feedback.mjs` | 0 | `P0_D3_ORDER_FEEDBACK_OK` |

## 残余 / 债

- 工作区无关 dirty 极大：未 stage、未混入本片。
- 最近使用 / 模板 / 密度 / 皮试：后置，未做。
- Codex exec 本机 PowerShell apply_patch 受阻，本片由执行器最小落盘收口。
- push 结果见 commit 后回报。
