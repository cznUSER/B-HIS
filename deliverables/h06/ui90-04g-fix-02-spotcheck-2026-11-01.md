# UI90-04G-FIX-02 抽验（患者建档摘要细部/字段网格/底操作条 chrome · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tipFinal（全）：`90d5b1c8478556f5f48fda0e301cf7d866cde152`（短 `90d5b1c`）
- 基线 tip：`9d23097`（`docs(h06): UI90-04G-FIX-01 spotcheck suggest-pass…`；全 `9d23097fc2419add4a8054c1f4da0d0c69051d8b`）
- 实现锚 FIX-01：`f282e09`（全 `f282e09d86b16338d2600d28f283dc1962391072`）
- HEAD 抽验时：`90d5b1c`（与 tipFinal 一致）
- origin：抽验时 `ahead 2 / behind 0`（`rev-list --left-right --count origin/...HEAD` = `0	2`；**本抽验不 push、不 force**）
- 对照：H14 卡 `deliverables/h14/ui90-04g-fix-02-start-card-2026-11-01.md`（SHA256 `8b71345472d9f744ab759d1392137c05b2a73318a3069c2c5b5794258e3411eb`，与期望一致）
- H05：`deliverables/h05/ui90-04g-fix-02-2026-11-01.md` + `deliverables/h05/ui90-04g-fix-02/`
- 性质：结构/禁区抽验 · **未签 90%** · **非 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不签 UI90-04G ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮件 / 禁 qq-mail。

闭合对象：仅 Morph-A **G** 患者建档「摘要细部 + `.field-grid` 分区 + 底操作条」UI-only 最小片（守恒 FIX-01 壳 868 / topH 87；底钮视觉 90×30）。成功 ≠ 90% 签收；几何 ≠ 已真建档写；未改 `createPatient`/`onCreate`；未做列表 Morph / BTN-WIRE；未覆盖 FIX-01/EVID。

## 0.1) tip 字串滞后备注（仍存在）

H05 交付 md「Git tip」仍写 **短 tip：`83dcc83`** / 全 `83dcc83f5b5f908fe2116b0ab8e0b9971a6208ab`。  
抽验以 **HEAD / tipFinal `90d5b1c`** 为准（H01 已明示）。`83dcc83` 与 `90d5b1c` 同父 `9d23097`、同 message；二者 tree 差仅该 md tip 字串两行（历史 tip-fill/amend 链残留；`83dcc83` **非**当前分支祖先）。**代码 / fixture / after-results 以 `90d5b1c` 为准**；本滞后为 docs-only，不构成不通过。

## 1) tip 范围核对

`git show 90d5b1c --stat` / `git diff --name-only 9d23097..90d5b1c`：

| 路径 | 角色 |
|---|---|
| `web/src/views/PatientView.vue` | 仅 template 增量 testid：`patient-summary-grid` / `patient-form-footer` / `patient-panel-actions`（numstat 3/3） |
| `web/src/views/clinical-design.css` | FIX-01 块后新增 **UI90-04G-FIX-02** 注释块；选择器均 `.patient-page` / `.shell-main .patient-page`；footer 钮 `height/min-height:30px` + `min-width:90px` |
| `deliverables/h05/ui90-04g-fix-02-2026-11-01.md` | H05 交付说明（明示不签 90%；**tip 字串仍写 83dcc83，见 §0.1**） |
| `deliverables/h05/ui90-04g-fix-02/_scoped.css` | 隔离 fixture scoped CSS |
| `deliverables/h05/ui90-04g-fix-02/geometry-fixture.html` | 隔离 chrome fixture |
| `deliverables/h05/ui90-04g-fix-02/geometry-probe.cjs` | 三视口几何探针 |
| `deliverables/h05/ui90-04g-fix-02/after-results.json` | `allPass=true`；非活体标注 |
| `deliverables/h05/ui90-04g-fix-02/after-{1440,1024,390}.png` | 三视口截图 |

**无** OrderView / ChargeView / Go / migrations（含 022）/ ICD / Dict / 新 search API / AppShell / styles.css / shell-design / design-system / BTN-WIRE / 真建档写/支付写。  
父 tip=`9d23097`；`9d23097..90d5b1c` = **1** tip · **10** 路径。WT 大量 dirty/untracked **未混入 tip**（禁 force/reset）。

## 2) 生产码核对

| 项 | 结果 |
|---|---|
| CSS 作用域 | tip 新增规则均 `.patient-page` / `.shell-main .patient-page`；无裸选择器扩及 order/register |
| 共享 footer | `:is(.patient-page, .register-page) .clinical-form-footer` 相对基线 **未改**（CSS tip diff 无 `:is(` 命中）；FIX-02 仅加 `.patient-page .clinical-create-panel .clinical-form-footer` 覆盖 |
| 守恒 FIX-01 | `--patient-shell-w:868px` 与 FIX-01 块仍在；`End UI90-04G-FIX-01` → 接 FIX-02 块 |
| footer 钮 | `height:30px` / `min-height:30px` / `min-width:90px`；窄屏 `flex: 1 1 90px` |
| `PatientView.vue` script | tip diff **仅 3 处 template testid**；`createPatient`/`onCreate` 调用体未改；after-results `scriptByteIdentical=true` |
| 真建档写 / 支付写 / 列表 Morph / BTN-WIRE | tip 无；H05 md / after-results note 明示未做/未核验 |
| Order / Charge / Go / 022 | tip `git diff --name-only` 空 |

## 3) 几何 / 横溢（对照 after-results.json）

标注：隔离 fixture · **非活体** · 不得冒充活体 SCORE 前置。`allPass=true`；`pageerrors=[]`；`blockedExternalRequests=[]`。

| 视口 | pageW | topH | footer 钮 | overflow | pass |
|---:|---:|---:|---|---|---|
| 1440 | **868** | **87** | h=30 / min-w=90 / w=90 | false | true |
| 1024 | 944（max-width 100%） | **87** | h=30 / min-w=90 / w=90 | false | true |
| 390 | 310（max-width 100%） | **87** | h=30 / min-w=90 / w=134（窄屏 stretch） | false | true |

invariants：`scriptByteIdentical` / `fix01Kept` / `cssWithoutFix02MatchesTip` / `patientOnlySelectors` / `noSharedFooterRewrite` / `scopedShellWidth` / `newTestIds` **均为 true**。  
H05 报「1440 pageW=868 / topH=87 / allPass + fix01Kept / scriptByteIdentical / noSharedFooterRewrite」与 `after-results.json` **一致**。截图路径存在。  
**无新横溢**：三视口 `overflow=false` / `overflowPass=true`。  
**FIX-01 未回退**：壳 868 / topH 87 守恒。

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| createPatient/onCreate 语义 / 真写患者/支付 | tip 仅 CSS + chrome/testid；script 同基线语义；diff 无 create/onCreate 行变更 |
| 列表 Morph / BTN-WIRE | tip 无；md / after-results note 明示另债未做/未核验 |
| 破坏 FIX-01 | `fix01Kept=true`；token 868 / topH 87 仍在 after-results |
| 共享 footer 改写 | `noSharedFooterRewrite=true`；`:is(patient,register)` 块相对基线未改 |
| Go / 022 / ICD / Dict / 新 search | tip 路径无 |
| 误伤 Order / Charge / styles.css | tip 路径无；新规则均 `.patient-page` 前缀 |
| 签 90% / SCORE / PRELIVE | tip message / md / after-results note 明示不做；本抽验亦不签 |
| force / reset / 脏混 tip | tip 仅 10 交付/chrome 路径；WT dirty 未入 tip |
| qq-mail | 本抽验未发任何用户邮件 |
| DIAG-AUTOSAVE / 公网 / mock 充活体 | tip 无；fixture 标注 non-live / safeFixture |

## 5) 建议

**建议通过** tipFinal `90d5b1c`（全 `90d5b1c8478556f5f48fda0e301cf7d866cde152`）。  
备注：H05 md tip 字串仍滞后写 `83dcc83`（docs-only；以 `90d5b1c` 为准）。  
H06 不定口径、不关片、不签 90%、不 SCORE、不混 PRELIVE、不 push。

证据路径：本文件；H05 md；`deliverables/h05/ui90-04g-fix-02/{geometry-fixture.html,geometry-probe.cjs,after-results.json,after-*.png}`；H14 卡 SHA256 已核。
