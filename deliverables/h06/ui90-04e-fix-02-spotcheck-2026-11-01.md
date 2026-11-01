# UI90-04E-FIX-02 抽验（收费明细区/结算区 chrome · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`74318dea1e661e999abd503d8ec2bc58e103fd15`（短 `74318de`）
- 基线 tip：`af35e66`（`docs(h06): UI90-04E-FIX-01 spotcheck …`）
- HEAD 抽验时：`74318de`（与实现 tip 一致）
- origin：`0/0`（相对 origin；**本抽验不 push、不 force**）
- 对照：H14 卡；H05 `deliverables/h05/ui90-04e-fix-02-2026-11-01.md` + `deliverables/h05/ui90-04e-fix-02/`
- 性质：结构/禁区抽验 · **未签 90%** · **非 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不签 UI90-04E ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮件 / 禁 qq-mail。

闭合对象：仅 Morph-A **E** 明细区/结算区 UI-only 最小片（preview~329 + settle~340 + FIX-01 壳 1008/47 守恒 + 三视口无新横溢）。成功 ≠ 90% 签收；几何 ≠ 已真支付；未改 `payCharge`/`refundCharge`；未做 BTN-WIRE。

## 1) tip 范围核对

`git show 74318de --stat` / `git diff --name-only af35e66..74318de`：

| 路径 | 角色 |
|---|---|
| `web/src/views/charge-design.css` | `--charge-preview-h:329px` / `--charge-settle-w:340px`；workspace 侧栏 token；preview result-wrap min-height；settle 可读收口；**守恒** `--charge-shell-w:1008px` / `--charge-top-h:47px` |
| `web/src/views/ChargeView.vue` | **仅** chrome：`data-testid="charge-workspace"` / `charge-preview` / `charge-pending-scroll-wrap`（3 处） |
| `deliverables/h05/ui90-04e-fix-02-2026-11-01.md` | H05 交付说明（明示不签 90%） |
| `deliverables/h05/ui90-04e-fix-02/_scoped.css` | 隔离 fixture scoped CSS |
| `deliverables/h05/ui90-04e-fix-02/geometry-fixture.html` | 隔离 chrome fixture |
| `deliverables/h05/ui90-04e-fix-02/geometry-probe.cjs` | 三视口几何探针 |
| `deliverables/h05/ui90-04e-fix-02/after-results.json` | `allPass=true`；非活体标注 |
| `deliverables/h05/ui90-04e-fix-02/after-{1440,1024,390}.png` | 三视口截图 |

**无** Go / migrations（含 022）/ OpenAPI / ICD / Dict / 新 search API / AppShell / shell-design / design-system / BTN-WIRE / 真支付写。  
父 tip=`af35e66`；`af35e66..74318de` = **1** tip · **10** 路径。WT 大量 dirty/untracked **未混入 tip**（禁 force/reset）。  
**未覆盖** `deliverables/h05/ui90-04e-fix-01*`（`git diff` 对该目录为空；目录仍在 tip 树内）。

## 2) 生产码核对

| 项 | 结果 |
|---|---|
| CSS token | `--charge-shell-w:1008px`；`--charge-top-h:47px`；`--charge-preview-h:329px`；`--charge-settle-w:340px` |
| workspace / settle | `grid-template-columns: minmax(0,1fr) var(--charge-settle-w, 340px)`；`.charge-pay` 可读收口 |
| preview 区 | `.charge-preview .result-wrap` `min-height: var(--charge-preview-h, 329px)`；表列宽/sticky 头 |
| `ChargeView.vue` script | **与基线相同**（`<template>` 前 script 全文 SHA256 一致：`fc126b1b…ef8c`） |
| template 变更 | 仅三处 `data-testid`；diff 无 pay/refund 行 |
| `payCharge` / `refundCharge` / `onPay` / `onRefund` | tip diff **不触及其调用体**；语义未改 |
| 真支付写 / BTN-WIRE | tip 无；H05 md 明示未做 |

## 3) 几何 / 横溢（对照 after-results.json）

标注：隔离 fixture · **非活体** · 不得冒充活体 SCORE 前置。`allPass=true`；`pageerrors=[]`。

| 视口 | pageW | workbarH | wrapH | settleW | overflow | pass |
|---:|---:|---:|---:|---:|---|---|
| 1440 | **1008** | **47** | **370** | **340** | false | true |
| 1024 | 944 | **47** | 370 | 898（单列堆叠） | false | true |
| 390 | 310 | **47** | 370 | 272 | false | true |

Token 探针：`tokenShell=1008px` / `tokenTop=47px` / `tokenPreview=329px` / `tokenSettle=340px`（三视口均 tokenPass）。  
FIX-01 壳守恒：本 tip after 1440 `pageW=1008`/`workbarH=47`；FIX-01 证据目录未改，其 after 仍 1008/47。截图路径存在且非空。

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| payCharge/refundCharge 语义 / 真支付写 | tip 仅 CSS + 三 testid；script 同基线；diff 无 pay/refund 行 |
| BTN-WIRE | tip 无；md 明示另债 |
| Go / 022 / ICD / Dict / 新 search | tip 路径无 |
| 签 90% / SCORE / PRELIVE | tip md 与 after-results note 明示不做；本抽验亦不签 |
| 覆盖 FIX-01 证据 | tip 未触碰 `ui90-04e-fix-01*` |
| force / reset / 脏混 tip | tip 仅 10 交付/chrome 路径；WT dirty 未入 tip |
| qq-mail | 本抽验未发任何用户邮件 |
| DIAG-AUTOSAVE / 公网 / mock 充活体 | tip 无；fixture 标注 non-live |

## 5) 建议

**建议通过** tip `74318de`（全 `74318dea1e661e999abd503d8ec2bc58e103fd15`）。  
H06 不定口径、不关片、不签 90%、不 SCORE、不混 PRELIVE、不 push。

证据路径：本文件；H05 md；`deliverables/h05/ui90-04e-fix-02/{geometry-fixture.html,geometry-probe.cjs,after-results.json,after-*.png}`。
