# UI90-04F-FIX-02 抽验（开单明细/历史/底条 chrome · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`5bd34bee25edafbd2a53c8cdbd2adb3323e4f55d`（短 `5bd34be`）
- 基线 tip：`1e3cf29`（`docs(h06): UI90-04F-FIX-01 spotcheck suggest-pass (97462c0; no 90% sign-off)`；全 `1e3cf294f36c20cfdcfa878d566d039bd294d52a`）
- HEAD 抽验时：`5bd34be`（与实现 tip 一致）
- origin：抽验时 `ahead 1`（相对 origin；**本抽验不 push、不 force**）
- 对照：H14 卡要点；H05 `deliverables/h05/ui90-04f-fix-02-2026-11-01.md` + `deliverables/h05/ui90-04f-fix-02/`
- 性质：结构/禁区抽验 · **未签 90%** · **非 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不签 UI90-04F ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮件 / 禁 qq-mail。

闭合对象：仅 Morph-A **F** 明细表/历史栏/底操作条 UI-only 最小片（DFM lines~263 + history~268 + actionbar 可读 chrome + FIX-01 壳 1036/105 守恒 + `min-width:1070` 保留 + 三视口无新横溢）。成功 ≠ 90% 签收；几何 ≠ 已真开单写；未改 `saveOrder`/`commitOrder`/`abateOrder`；未做 BTN-WIRE；未覆盖 FIX-01/EVID 证据。

## 1) tip 范围核对

`git show 5bd34be --stat` / `git diff --name-only 1e3cf29..5bd34be`：

| 路径 | 角色 |
|---|---|
| `web/src/views/clinical-design.css` | 新增 FIX-02 块：`--order-lines-h:263px` / `--order-history-w:268px`；`.line-scroll` min-height；history max-width 268；actionbar sticky；**守恒** `--order-shell-w:1036px` / `--order-top-h:105px`；**保留** `.line-table { min-width:1070px }`；规则均 `.order-page` / `.shell-main .order-page` 前缀 |
| `web/src/views/OrderView.vue` | **仅** chrome：`data-testid="order-fields"` / `order-line-scroll` / `clinical-order-actionbar` / `clinical-order-history` |
| `deliverables/h05/ui90-04f-fix-02-2026-11-01.md` | H05 交付说明（明示不签 90%） |
| `deliverables/h05/ui90-04f-fix-02/_scoped.css` | 隔离 fixture scoped CSS |
| `deliverables/h05/ui90-04f-fix-02/geometry-fixture.html` | 隔离 chrome fixture |
| `deliverables/h05/ui90-04f-fix-02/geometry-probe.cjs` | 三视口几何探针 |
| `deliverables/h05/ui90-04f-fix-02/after-results.json` | `allPass=true`；非活体标注 |
| `deliverables/h05/ui90-04f-fix-02/after-{1440,1024,390}.png` | 三视口截图 |

**无** Go / migrations（含 022）/ OpenAPI / ICD / Dict / 新 search API / AppShell / shell-design / design-system / ChargeView / BTN-WIRE / 真开单写/支付写。  
父 tip=`1e3cf29`；`1e3cf29..5bd34be` = **1** tip · **10** 路径。WT 大量 dirty/untracked **未混入 tip**（禁 force/reset）。  
`git diff --name-only 1e3cf29..5bd34be -- 'deliverables/h05/ui90-04f-fix-01*' '*evid*'`：**空**（未覆盖 FIX-01 / EVID）。

## 2) 生产码核对

| 项 | 结果 |
|---|---|
| CSS token | `--order-lines-h:263px`；`--order-history-w:268px`；守恒 `--order-shell-w:1036px` / `--order-top-h:105px` |
| 明细 / 历史 / 底条 | `.line-scroll` min-height `var(--order-lines-h)`；history `max-width: var(--order-history-w)`；actionbar sticky 收口；`.line-table { min-width:1070px; width:max(100%,1070px) }` **保留** |
| 作用域 | 新增/改动规则均带 `.order-page` / `.shell-main .order-page`；tip diff **无** 无前缀 `.line-scroll` / `.clinical-order-*` 裸选择器；**无** `+/-` 触碰 `.patient-page` / `.register-page` |
| `OrderView.vue` script | **与基线相同**（`<template` 之前全文 `script_before_template_identical=True`；SHA256 `d367d79a48c25616daa0300244648fa37f9a8c9b767c9b5ff0456a673b45f507`） |
| template 变更 | 仅 4 处 `data-testid`；diff 全部在 template；不触碰 `@click="onSave|onCommit|onAbate"` |
| `saveOrder` / `commitOrder` / `abateOrder` | tip diff **不触及其调用体**；语义未改 |
| 真开单写 / 支付写 / BTN-WIRE | tip 无；H05 md 明示未做 |
| FIX-01 壳未回退 | tip CSS 仍含 FIX-01 块与 `1036/105` token；after-results 1440 `pageW=1036` / `topH=105` |

## 3) 几何 / 横溢（对照 after-results.json）

标注：隔离 fixture · **非活体** · 不得冒充活体 SCORE 前置。`allPass=true`；`pageerrors=[]`。

| 视口 | pageW | topH | lineScrollH | historyW | actionbarH | tableMinWidth | overflow | pass |
|---:|---:|---:|---:|---:|---:|---|---|---|
| 1440 | **1036** | **105** | **265**（token 263±） | **268** | **65** | **1070px** | false | true |
| 1024 | 944 | **105** | 265 | 944（单列堆叠） | 65 | **1070px** | false | true |
| 390 | 310 | 139（窄屏增高） | 249 | 310 | 154 | **1070px** | false | true |

Token 探针：`tokenShell=1036px` / `tokenTop=105px` / `tokenLines=263px` / `tokenHistory=268px`（三视口 tokenPass）。  
H05 报「1440 pageW=1036 / topH=105 / lineScrollH≈265 / historyW=268 / actionbarH≈65 / allPass」与 `after-results.json` **一致**。截图路径存在。  
**无新横溢**：三视口 `overflow=false` / `overflowPass=true`。

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| saveOrder/commitOrder/abateOrder 语义 / 真写/支付 | tip 仅 CSS + chrome/testid；script 同基线；diff 无 save/commit/abate 行变更 |
| BTN-WIRE | tip 无；md / after-results note 明示另债 |
| Go / 022 / ICD / Dict / 新 search | tip 路径无（`git diff --name-only … -- '*.go' 'migrations/**' 'internal/**'` 空） |
| 误伤 patient/register | tip CSS/Vue 路径无 Patient/Register/Charge；新规则均 `.order-page` 前缀 |
| 覆盖 FIX-01 / EVID | tip 未触 `ui90-04f-fix-01*` / EVID；仅新增 `ui90-04f-fix-02*` |
| 签 90% / SCORE / PRELIVE | tip md 与 after-results note 明示不做；本抽验亦不签 |
| force / reset / 脏混 tip | tip 仅 10 交付/chrome 路径；WT dirty 未入 tip |
| qq-mail | 本抽验未发任何用户邮件 |
| DIAG-AUTOSAVE / 公网 / mock 充活体 | tip 无；fixture 标注 non-live |

## 5) 建议

**建议通过** tip `5bd34be`（全 `5bd34bee25edafbd2a53c8cdbd2adb3323e4f55d`）。  
H06 不定口径、不关片、不签 90%、不 SCORE、不混 PRELIVE、不 push。

证据路径：本文件；H05 md；`deliverables/h05/ui90-04f-fix-02/{geometry-fixture.html,geometry-probe.cjs,after-results.json,after-*.png}`。
