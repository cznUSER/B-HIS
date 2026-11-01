# UI90-ERR-CLEAN-02 抽验（方案 B · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`9336943fd190c3c312f14aef257eae9320be27ff`（短 `9336943`）
- 基线 tip：`6ac2922`（`6ac292212c3e554aa6be02cfead25808ac4f59ec`）
- HEAD 抽验时：`9336943`（与目标 tip 一致）
- origin：`0/0`（抽验时）；**本抽验不 push**
- 对照：H14 方案 B；H05 `deliverables/h05/ui90-err-clean-02-2026-11-01.md` + probe
- 性质：禁区/可解析抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **未做 FIX-03** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务、不把 UI90 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮箱 / 禁 qq-mail。

闭合对象：Diagnosis / Dispense / Order 三临床页不再依赖 tip 树中缺失的 `useErrorDialog.js`（方案 B 关片）。成功 ≠ 90% 签收。

## 1) tip 范围核对

`git show 9336943 --stat` / `git diff 6ac2922..9336943 --name-status`：

| 路径 | 角色 |
|---|---|
| `web/src/views/DiagnosisView.vue` | 删 import + `useErrorDialog([errorMessage, registersError, diagnosesError])`（−6） |
| `web/src/views/DispenseView.vue` | 删 import + 调用（−7） |
| `web/src/views/OrderView.vue` | 删 import + 调用（−9） |
| `deliverables/h05/ui90-err-clean-02-2026-11-01.md` | H05 交付说明 |
| `deliverables/h05/ui90-err-clean-02-probe-2026-11-01.mjs` | 静态 probe |

**零** Go / migrations / 022 / OpenAPI / ICD / Dict / AppShell / DepartmentsPanel / App.vue / main.js / 新 search API。  
未入库 `useErrorDialog.js` / `ErrorDialog.vue` / `formValidation.js`；未改 `SubmitStatusBanner.vue` / `useSubmitFeedback.js` 实现；未 ERR 全包；未做方案 A。

## 2) 方案 B 验收（对照 tip 树）

| 项 | 结果 |
|---|---|
| `git grep useErrorDialog 9336943 --` 三文件 | **空**（DiagnosisView / DispenseView / OrderView 无 match） |
| tip 全树 `git grep useErrorDialog 9336943 -- web/src` | **空**（exit 1） |
| tip 树 helper | **未跟踪**：`git ls-tree` feedback 仅 `SubmitStatusBanner.vue` + `useSubmitFeedback.js`；无 `useErrorDialog.js` / `ErrorDialog.vue` / `formValidation.js` |
| Vue diff | 仅删悬空 import/调用；无其它业务逻辑改动 |
| DiagnosisView 本地反馈 | **仍在**：`role="alert"` / `diagnosis-alert-error` / `errorMessage` / `registersError` / `diagnosesError` / `SubmitStatusBanner` / `createSubmitFeedback` |
| DispenseView | **仍在**：`data-testid="dispense-error"` / `role="alert"` / `errorMessage` / `return-error` / `SubmitStatusBanner` |
| OrderView | **仍在**：`data-testid="order-error"` / `clinical-read-error` / `invalidMessage` / `SubmitStatusBanner` |
| tip `App.vue` / tip-clean `main.js` | **无** ErrorDialog / useErrorDialog / formValidation 挂载 |
| 护理壳 / DepartmentsPanel | tip **未改** |
| SubmitStatusBanner 实现 | tip **未改**（仅消费方仍 import 组件） |
| App 挂载 / ERR 全包 / 方案 A | **未做** |
| 零新 API | tip 无 api/** 变更、无新 endpoint |
| FIX-03 / SCORE / PRELIVE | **未做** |
| 未签 90% | tip message + H05 md §0 + 本抽验均明示 |

## 3) probe

运行：`node deliverables/h05/ui90-err-clean-02-probe-2026-11-01.mjs`

```json
{
  "ok": true,
  "fails": [],
  "cleaned": [
    "web/src/views/DiagnosisView.vue",
    "web/src/views/DispenseView.vue",
    "web/src/views/OrderView.vue"
  ],
  "wtThreePagesClean": true,
  "tipWebSrcGrepAtProbeTime": "(empty)",
  "helperOnDiskButMustStayUntracked": true,
  "helperTracked": false,
  "note": "Plan B close-out: DiagnosisView + DispenseView + OrderView; no 90% sign-off; ERR WIP do-not-commit"
}
```

## 4) 禁区自检

- [x] 未 DIAG-AUTOSAVE
- [x] 未官方 ICD/Dict/强制编码/新 search API/022
- [x] 未借刀凑分 / 假绿 / 关闭整包 UI90
- [x] 未 mock 充活体 / 未正式 SCORE / 未混 PRELIVE / 未做 FIX-03
- [x] 未改业务壳口径 / SubmitStatusBanner 实现
- [x] 未 ERR 全包（ErrorDialog+formValidation+App/main）
- [x] 未 force/reset；本抽验精确 add 仅本 md；未公网；未 qq-mail
- [x] 方案 A 未做；`useErrorDialog.js` 仍 do-not-commit（磁盘 untracked WIP）

## 5) 非挡残留

- 工作区大量 **dirty / untracked**（含其它视图对 `useErrorDialog` 的 WT 接线，以及 untracked `useErrorDialog.js` / `ErrorDialog.vue` / `formValidation.js`）。**属 WT WIP，不在 tip 树**；本刀与本抽验均未入库、未混交。
- tip 树 `web/src` 在 CLEAN-02 后 `useErrorDialog` **已清零**；后续若入库 ERR 全包需另派，不在本片范围。

## 6) 结论

**建议通过。** tip `9336943` 相对基线 `6ac2922` 仅删三临床页悬空 `useErrorDialog` 接线并交付 H05 md/probe；本地 alert / SubmitStatusBanner 路径保留；helper 未入库；未 App 挂载 / 未方案 A / 零新 API；未签 90%、未 SCORE、未混 PRELIVE、未做 FIX-03。本抽验未改业务码、**不 push**。

---

**回交 H01：** 建议通过 · 目标 tip `9336943` · 本抽验 commit 见同刀 · push=否 · 零业务码。
