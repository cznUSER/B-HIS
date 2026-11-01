# UI90-04G-FIX-01 抽验（患者建档壳/摘要/创建面板 chrome · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`f282e09d86b16338d2600d28f283dc1962391072`（短 `f282e09`）
- 基线 tip：`403e8b2`（`docs(h06): UI90-04G-EVID-01 PatientView Morph evidence pack…`；全 `403e8b202842caedf3abc56a0b6e34ff83c03b07`）
- HEAD 抽验时：`f282e09`（与实现 tip 一致）
- origin：抽验时 `ahead 1`（相对 origin；**本抽验不 push、不 force**）
- 对照：H14 卡 `deliverables/h14/ui90-04g-fix-01-start-card-2026-11-01.md`（SHA256 `9361edcec03265a3d2c30a1612d63373d5e6ceba829814be17dd01fa202a8ebb`，与期望一致）
- H05：`deliverables/h05/ui90-04g-fix-01-2026-11-01.md` + `deliverables/h05/ui90-04g-fix-01/`
- 性质：结构/禁区抽验 · **未签 90%** · **非 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不签 UI90-04G ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮件 / 禁 qq-mail。

闭合对象：仅 Morph-A **G** 患者建档壳/顶区/摘要/创建面板 UI-only 最小片（DFM PatientFrm 桌面壳 ~868 + 顶区 chrome + 三视口无新横溢）。成功 ≠ 90% 签收；几何 ≠ 已真建档写；未改 `createPatient`/`onCreate`；未做 FIX-02 / 列表 Morph / BTN-WIRE；未覆盖 EVID。

## 0.1) tip 字串滞后备注（仍存在）

H05 交付 md「Git / 交接」仍写 **本次 commit：`b45445d`**（全 `b45445dc411b60335e4d099904183c97de34f9aa`）。  
抽验以 **HEAD / 实现 tip `f282e09`** 为准（H01/H14 已明示）。`b45445d` 与 `f282e09` 同父 `403e8b2`、同 message；二者 tree 差仅该 md 一行 tip 字串（历史 amend 链残留）。**代码 / fixture / after-results 以 `f282e09` 为准**；本滞后为 docs-only，不构成不通过。

## 1) tip 范围核对

`git show f282e09 --stat` / `git diff --name-only 403e8b2..f282e09`：

| 路径 | 角色 |
|---|---|
| `web/src/views/PatientView.vue` | chrome/template：`data-testid="patient-page"`；`.patient-top-chrome` 包 crumb+BusinessPageHeader；scoped `max-width` → `var(--patient-shell-w, 868px)` |
| `web/src/views/clinical-design.css` | 文件末新增 FIX-01 块：`--patient-shell-w: 868px`；顶区/摘要/表单 spacing；规则均 `.patient-page` / `.shell-main .patient-page` 前缀 |
| `deliverables/h05/ui90-04g-fix-01-2026-11-01.md` | H05 交付说明（明示不签 90%；**tip 字串仍写 b45445d，见 §0.1**） |
| `deliverables/h05/ui90-04g-fix-01/_scoped.css` | 隔离 fixture scoped CSS |
| `deliverables/h05/ui90-04g-fix-01/geometry-fixture.html` | 隔离 chrome fixture |
| `deliverables/h05/ui90-04g-fix-01/geometry-probe.cjs` | 三视口几何探针 |
| `deliverables/h05/ui90-04g-fix-01/after-results.json` | `allPass=true`；非活体标注 |
| `deliverables/h05/ui90-04g-fix-01/after-{1440,1024,390}.png` | 三视口截图 |

**无** `styles.css` / OrderView / ChargeView / Go / migrations（含 022）/ ICD / Dict / 新 search API / AppShell / shell-design / design-system / BTN-WIRE / 真建档写/支付写。  
父 tip=`403e8b2`；`403e8b2..f282e09` = **1** tip · **10** 路径。WT 大量 dirty/untracked **未混入 tip**（禁 force/reset）。

## 2) 生产码核对

| 项 | 结果 |
|---|---|
| CSS token | `--patient-shell-w: 868px`；桌面 `max-width: var(--patient-shell-w)`；`margin-inline: auto` |
| 顶区 / 摘要 / 表单 | `.patient-top-chrome` flex 列；摘要/创建面板 padding 18px（≤640px 16px）；页脚按钮高 30px 视觉 |
| 作用域 | tip 新增选择器均带 `.patient-page` / `.shell-main .patient-page`；裸选择器扫空；**未**触碰 `.order-page` / `.register-page` / Charge |
| `PatientView.vue` script | **与基线相同**（`<template` 之前全文 `scriptByteIdentical=true`） |
| template 变更 | 根 `data-testid="patient-page"`；顶区 wrapper；保留 `patient-summary` / `patient-form` / `patient-mode-switch` |
| `createPatient` / `onCreate` | tip diff **不触及 script 调用体**；语义未改 |
| 真建档写 / 支付写 / BTN-WIRE | tip 无；H05 md / after-results note 明示未做 |
| styles.css / Order / Charge | tip `git diff --name-only` 空 |

## 3) 几何 / 横溢（对照 after-results.json）

标注：隔离 fixture · **非活体** · 不得冒充活体 SCORE 前置。`allPass=true`；`pageerrors=[]`；`blockedExternalRequests=[]`。

| 视口 | pageW | topH | overflow | pass |
|---:|---:|---:|---|---|
| 1440 | **868** | **87** | false | true |
| 1024 | 944（max-width 100%） | **87** | false | true |
| 390 | 310（max-width 100%） | **87** | false | true |

invariants：`scriptByteIdentical` / `existingCssUnchanged` / `patientOnlySelectors` / `rootTestId` / `topChromeTestId` / `preservedTestIds` / `scopedShellWidth` **均为 true**。  
H05 报「1440 pageW=868 / topH=87 / allPass」与 `after-results.json` **一致**。截图路径存在。  
**无新横溢**：三视口 `overflow=false` / `overflowPass=true`。

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| createPatient/onCreate 语义 / 真写患者/支付 | tip 仅 CSS + chrome/testid；script 同基线；diff 无 create/onCreate 行变更 |
| FIX-02 / 列表 Morph / BTN-WIRE | tip 无；md / after-results note 明示另债未做/未核验 |
| Go / 022 / ICD / Dict / 新 search | tip 路径无 |
| 误伤 styles.css / Order / Charge | tip 路径无；新规则均 `.patient-page` 前缀 |
| 签 90% / SCORE / PRELIVE | tip message / md / after-results note 明示不做；本抽验亦不签 |
| force / reset / 脏混 tip | tip 仅 10 交付/chrome 路径；WT dirty 未入 tip |
| qq-mail | 本抽验未发任何用户邮件 |
| DIAG-AUTOSAVE / 公网 / mock 充活体 | tip 无；fixture 标注 non-live / safeFixture |

## 5) 建议

**建议通过** tip `f282e09`（全 `f282e09d86b16338d2600d28f283dc1962391072`）。  
备注：H05 md tip 字串仍滞后写 `b45445d`（docs-only；以 `f282e09` 为准）。  
H06 不定口径、不关片、不签 90%、不 SCORE、不混 PRELIVE、不 push。

证据路径：本文件；H05 md；`deliverables/h05/ui90-04g-fix-01/{geometry-fixture.html,geometry-probe.cjs,after-results.json,after-*.png}`；H14 卡 SHA256 已核。
