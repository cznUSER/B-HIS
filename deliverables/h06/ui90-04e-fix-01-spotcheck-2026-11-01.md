# UI90-04E-FIX-01 抽验（收费壳宽/检索带 chrome · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`6d449104db06280e3d1442499d8b67a0e87231de`（短 `6d44910`）
- 基线 tip：`d53c3d3`（`docs(h06): UI90-04E-EVID-01 outpatient charge evidence pack …`）
- HEAD 抽验时：`3cdfc94`（含 docs 背填 tip；**判定以实现 tip `6d44910` 为主**，对照 `d53c3d3..6d44910`）
- origin：相对 origin ahead（实现 tip 推债并行；**本抽验不 push、不 force**）
- 对照：H14 卡；H05 `deliverables/h05/ui90-04e-fix-01-2026-11-01.md` + `deliverables/h05/ui90-04e-fix-01/`
- 性质：结构/禁区抽验 · **未签 90%** · **非 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不签 UI90-04E ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮件 / 禁 qq-mail。

闭合对象：仅 Morph-A **E** 检索带/页壳/顶区 UI-only 最小首片（DFM 壳 ~1008 + 顶区/检索 chrome ~47 + 三视口无新横溢）。成功 ≠ 90% 签收；壳 ≠ 结算已验；未改 `payCharge`/`refundCharge`；未做 FIX-02 / BTN-WIRE。

## 1) tip 范围核对

`git show 6d44910 --stat` / `git diff --name-only d53c3d3..6d44910`：

| 路径 | 角色 |
|---|---|
| `web/src/views/charge-design.css` | 壳宽 token 1008、顶区/检索带 ~47、candidate-zone 收口 |
| `web/src/views/ChargeView.vue` | **仅** chrome：`data-testid="charge-page"` / `data-testid="charge-workbar"` |
| `deliverables/h05/ui90-04e-fix-01-2026-11-01.md` | H05 交付说明（明示不签 90%） |
| `deliverables/h05/ui90-04e-fix-01/_scoped.css` | 隔离 fixture 用 scoped CSS |
| `deliverables/h05/ui90-04e-fix-01/geometry-fixture.html` | 隔离 chrome fixture |
| `deliverables/h05/ui90-04e-fix-01/geometry-probe.cjs` | 三视口几何探针 |
| `deliverables/h05/ui90-04e-fix-01/after-results.json` | `allPass=true`；非活体标注 |
| `deliverables/h05/ui90-04e-fix-01/after-{1440,1024,390}.png` | 三视口截图 |

**无** Go / migrations（含 022）/ OpenAPI / ICD / Dict / 新 search API / AppShell / shell-design / design-system / FIX-02 / BTN-WIRE。  
父 tip=`d53c3d3`；`d53c3d3..6d44910` = **1** tip · **10** 路径。WT 大量 dirty/untracked **未混入 tip**（禁 force/reset）。

## 2) 生产码核对

| 项 | 结果 |
|---|---|
| `charge-design.css` token | `--charge-shell-w: 1008px`；`--charge-top-h: 47px`；`.charge-page` max-width 桌面壳；窄屏 `max-width:100%` |
| 顶区 / 检索带 | `.charge-workbar` / `.charge-search` 均 `min-height: var(--charge-top-h)`；检索字段单行 flex |
| candidate-zone | padding 收口至 `14px 22px 16px`；gap 收至 12 |
| `ChargeView.vue` script | **与基线相同**（`<template>` 前 script 全文比对一致） |
| template 变更 | 仅两处 `data-testid`；无逻辑/事件/API 改动 |
| `payCharge` / `refundCharge` | tip diff **不触及其调用体**；语义未改 |
| 真支付写 / FIX-02 / BTN-WIRE | tip 无；H05 md 明示未做 |

## 3) 几何 / 横溢（对照 after-results.json）

标注：隔离 fixture · **非活体** · 不得冒充活体 SCORE 前置。`allPass=true`；`pageerrors=[]`。

| 视口 | pageW | workbarH | searchH | overflow | pass |
|---:|---:|---:|---:|---|---|
| 1440 | **1008** | **47** | **47** | false | true |
| 1024 | 944（`max-width:100%`） | **47** | **47** | false | true |
| 390 | 310 | **47** | 84（检索主动换行；探针容差内） | false | true |

Token 探针：`tokenShell=1008px` / `tokenTop=47px`（三视口均 true）。截图路径存在且非空。

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| payCharge/refundCharge 语义 / 真支付写 | tip 仅 CSS + 两 testid；script 同基线 |
| FIX-02 / BTN-WIRE | tip 无；md 明示另债 |
| Go / 022 / ICD / Dict / 新 search | tip 路径无 |
| 签 90% / SCORE / PRELIVE | tip md 与 after-results note 明示不做；本抽验亦不签 |
| force / reset / 脏混 tip | tip 仅 10 交付/chrome 路径；WT dirty 未入 tip |
| qq-mail | 本抽验未发任何用户邮件 |
| DIAG-AUTOSAVE / 公网 / mock 充活体 | tip 无；fixture 标注 non-live |

## 5) 建议

**建议通过** tip `6d44910`（全 `6d449104db06280e3d1442499d8b67a0e87231de`）。  
H06 不定口径、不关片、不签 90%、不 SCORE、不混 PRELIVE、不 push。

证据路径：本文件；H05 md；`deliverables/h05/ui90-04e-fix-01/{geometry-fixture.html,geometry-probe.cjs,after-results.json,after-*.png}`。
