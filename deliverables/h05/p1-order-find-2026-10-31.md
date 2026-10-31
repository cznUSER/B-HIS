# P1-ORDER-FIND-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`c2589d7`（全 `c2589d7f2421d8bcedb05c5da5355e52cdbdcda4`；本机 ahead 2）
- 岗位：H05 · 本机落盘（优先 Codex；本片直接最小实现；未用 CloudAgent；未 reset/force）

## 目标

门诊开单页处方明细「项目」列：可见 **药品 / 治疗** 分类 Tab + 关键词过滤；选中回写当前行 `itemNo`；空态「无匹配项目」≠ 目录加载失败；`locked` 下不可改项。不含常用/最近/模板/密度/皮试/特殊处方。

## 文件列表

| 文件 | 变更 |
| --- | --- |
| `web/src/workbench/orderItemFind.js` | **新建** `FIND_CATEGORIES` / `filterByKeyword` / 空态文案（零网络） |
| `web/src/components/outpatient/OrderItemFinder.vue` | **新建** 触发器+面板：分类 Tab、关键词、列表、空态、清空；`data-testid="line-item-${index}"` |
| `web/src/views/OrderView.vue` | **最小胶水**：行内 `<select>` → `OrderItemFinder`；`findItems`；禁改 print/submit |
| `web/scripts/check-order-item-find.mjs` | **新建** 分类/关键词/回写/空态/静态接线 |
| `deliverables/h05/p1-order-find-2026-10-31.md` | 本交付 |

禁区遵守：未改 outpatientContext / Frame / Queue / StatusChain / useSubmitFeedback / SubmitStatusBanner；未回改 OrderPrintPanel / orderPrintStatus / onPrint*；未改 orderApi / orderContract 业务校验；未混交无关 dirty。

## 行为说明

1. **分类 Tab**：仅 `DRUG`（药品）/ `TREAT`（治疗）；默认药品。
2. **关键词**：对当前分类按 `itemNo` / `itemName` / `spec` 前端过滤（trim + 大小写不敏感）；清空恢复该分类全量。
3. **选中**：emit `update:modelValue` → 行 `itemNo`；规格/单位/单价/金额仍走 `catalogOf` / `lineMoney`；多行各自 finder，不串项。
4. **空态**：列表无命中显示「无匹配项目」；目录加载失败仍走既有 `masterError`，本片不伪装。
5. **锁定**：`:disabled="locked || busy || !canWrite"`；fieldset 禁用与之一致。
6. **PRINT 共存**：finder 面板 `z-index: 40`，不改打印状态机与诚实文案。

## Harness

| 命令 | exit | 摘要 |
| --- | --- | --- |
| `node web/scripts/check-order-item-find.mjs` | 0 | `P1_ORDER_ITEM_FIND_OK` |
| `node web/scripts/check-order-submit-feedback.mjs` | 0 | `P0_D3_ORDER_FEEDBACK_OK` |
| `node web/scripts/check-order-print-status.mjs` | 0 | `P1_PRINT_STATUS_OK` |

## 残余

- 常用/最近/模板/密度/皮试/特殊处方属后续子片。
- 键盘全套快捷键、长名展开策略未纳入。
- origin 若 443 失败以本机 tip + 本交付为准。
- 工作区仍有大量无关 dirty；本片仅 stage 上表文件。

`DONE_P1_ORDER_FIND`