# UI90-04F-FIX-01 抽验（开单壳宽/顶区/编辑区 chrome · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`97462c067cc6a197693dc929ebe6a44e1e5c8b16`（短 `97462c0`）
- 基线 tip：`ec0106b`（`docs(h06): UI90-04F-EVID-01 OrderView Morph evidence pack …`；全 `ec0106be83725342ac3951763bf931fade2a7cf7`）
- HEAD 抽验时：`97462c0`（与实现 tip 一致）
- origin：抽验时 `ahead 1`（相对 origin；**本抽验不 push、不 force**）
- 对照：H14 卡；H05 `deliverables/h05/ui90-04f-fix-01-2026-11-01.md` + `deliverables/h05/ui90-04f-fix-01/`
- 性质：结构/禁区抽验 · **未签 90%** · **非 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不签 UI90-04F ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮件 / 禁 qq-mail。

闭合对象：仅 Morph-A **F** 壳宽/顶区/编辑区 UI-only 最小片（DFM 壳 ~1036 + 顶区 encounter chrome ~105 + `.order-page` 前缀编辑区收口 + 三视口无新横溢）。成功 ≠ 90% 签收；几何 ≠ 已真开单写；未改 `saveOrder`/`commitOrder`/`abateOrder`；未做 FIX-02 / BTN-WIRE。

## 1) tip 范围核对

`git show 97462c0 --stat` / `git diff --name-only ec0106b..97462c0`：

| 路径 | 角色 |
|---|---|
| `web/src/views/clinical-design.css` | `--order-shell-w:1036px` / `--order-top-h:105px`；`.order-page` max-width 壳；`.order-top-chrome` 固定顶区 105；encounter 压紧；`.clinical-order-editor` padding `18px 20px`；`.order-fields` gap 收口；窄屏 `max-width:100%` |
| `web/src/views/OrderView.vue` | **仅** chrome：`data-testid="order-page"` / `order-top-chrome` 包裹 panel-head+encounter / `data-testid="clinical-order-editor"` |
| `deliverables/h05/ui90-04f-fix-01-2026-11-01.md` | H05 交付说明（明示不签 90%） |
| `deliverables/h05/ui90-04f-fix-01/_scoped.css` | 隔离 fixture scoped CSS |
| `deliverables/h05/ui90-04f-fix-01/geometry-fixture.html` | 隔离 chrome fixture |
| `deliverables/h05/ui90-04f-fix-01/geometry-probe.cjs` | 三视口几何探针 |
| `deliverables/h05/ui90-04f-fix-01/after-results.json` | `allPass=true`；非活体标注 |
| `deliverables/h05/ui90-04f-fix-01/after-{1440,1024,390}.png` | 三视口截图 |

**无** Go / migrations（含 022）/ OpenAPI / ICD / Dict / 新 search API / AppShell / shell-design / design-system / ChargeView / FIX-02 / BTN-WIRE / 真开单写/支付写。  
父 tip=`ec0106b`；`ec0106b..97462c0` = **1** tip · **10** 路径。WT 大量 dirty/untracked **未混入 tip**（禁 force/reset）。

## 2) 生产码核对

| 项 | 结果 |
|---|---|
| CSS token | `--order-shell-w:1036px`；`--order-top-h:105px` |
| 壳 / 顶区 | `.shell-main .order-page, .order-page { max-width: var(--order-shell-w) }`；`.order-top-chrome { height/min-height: var(--order-top-h) }` |
| 作用域 | 新增/改动规则均带 `.order-page` / `.shell-main .order-page`；tip diff **无** `+/-` 触碰 `.patient-page` / `.register-page` 既有块 |
| `OrderView.vue` script | **与基线相同**（`<script>…</script>` 全文 `SCRIPT_IDENTICAL=true`） |
| template 变更 | 仅 testid + `order-top-chrome` 包裹；diff 全部在 `</script>` 之后 |
| `saveOrder` / `commitOrder` / `abateOrder` | tip diff **不触及其调用体**；语义未改 |
| 真开单写 / 支付写 / FIX-02 / BTN-WIRE | tip 无；H05 md 明示未做 |

## 3) 几何 / 横溢（对照 after-results.json）

标注：隔离 fixture · **非活体** · 不得冒充活体 SCORE 前置。`allPass=true`；`pageerrors=[]`。

| 视口 | pageW | topH | editorPad | overflow | pass |
|---:|---:|---:|---|---|---|
| 1440 | **1036** | **105** | 18/20 | false | true |
| 1024 | 944（`max-width:100%`） | **105** | 18/20 | false | true |
| 390 | 310 | 139（窄屏自动增高） | 18 | false | true |

Token 探针：`tokenShell=1036px` / `tokenTop=105px`（三视口均 tokenPass）。  
H05 报「1440 pageW=1036 / topH=105 / allPass」与 `after-results.json` **一致**。截图路径存在。

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| saveOrder/commitOrder/abateOrder 语义 / 真写/支付 | tip 仅 CSS + chrome/testid；script 同基线；diff 无 save/commit/abate 行 |
| FIX-02 / BTN-WIRE | tip 无；md 明示另债 |
| Go / 022 / ICD / Dict / 新 search | tip 路径无 |
| 误伤 patient/register | tip CSS diff 无 `patient-page`/`register-page` 增减行；新规则均 `.order-page` 前缀 |
| 签 90% / SCORE / PRELIVE | tip md 与 after-results note 明示不做；本抽验亦不签 |
| force / reset / 脏混 tip | tip 仅 10 交付/chrome 路径；WT dirty 未入 tip |
| qq-mail | 本抽验未发任何用户邮件 |
| DIAG-AUTOSAVE / 公网 / mock 充活体 | tip 无；fixture 标注 non-live |

## 5) 建议

**建议通过** tip `97462c0`（全 `97462c067cc6a197693dc929ebe6a44e1e5c8b16`）。  
H06 不定口径、不关片、不签 90%、不 SCORE、不混 PRELIVE、不 push。

证据路径：本文件；H05 md；`deliverables/h05/ui90-04f-fix-01/{geometry-fixture.html,geometry-probe.cjs,after-results.json,after-*.png}`。
