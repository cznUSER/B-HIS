# UI90-ERR-CLEAN-01 抽验（方案 B 首片 · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`ba47553bb993b54501cf3d91c81578d27541999b`（短 `ba47553`）
- 基线 tip：`8111ea8`（`8111ea8cb09b9bedb1b7678e309b94e58b68dd69`）
- HEAD 抽验时：`ba47553`（与目标 tip 一致）
- origin：`0/1` ahead（抽验前）；**本抽验不 push**
- 对照：H14 方案 B 首片；H05 `deliverables/h05/ui90-err-clean-01-2026-11-01.md` + probe
- 性质：禁区/可解析抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **未做 FIX-03** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务、不把 UI90 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮箱 / 禁 qq-mail。

闭合对象：挂号/收费前台 3 文件不再依赖 tip 树中缺失的 `useErrorDialog.js`（方案 B 首片）。成功 ≠ 90% 签收。Diagnosis/Dispense/Order 列为 ERR-CLEAN-02 残留，本片正确未清。

## 1) tip 范围核对

`git show ba47553 --stat` / `git diff 8111ea8..ba47553 --name-only`：

| 路径 | 角色 |
|---|---|
| `web/src/views/RegisterView.vue` | 删 import + `useErrorDialog([...])`（−11） |
| `web/src/components/RegisterPatientPicker.vue` | 删 import + 调用（−4） |
| `web/src/views/ChargeView.vue` | 删 import + 调用（−9） |
| `deliverables/h05/ui90-err-clean-01-2026-11-01.md` | H05 交付说明 |
| `deliverables/h05/ui90-err-clean-01-probe-2026-11-01.mjs` | 静态 probe |

**零** Go / migrations / 022 / OpenAPI / ICD / Dict / AppShell / DepartmentsPanel / App.vue / 新 search API。  
未入库 `useErrorDialog.js`；未改 `SubmitStatusBanner.vue` / `useSubmitFeedback.js` 实现；未 ERR 全包；未做方案 A。

## 2) 方案 B 验收（对照 tip 树）

| 项 | 结果 |
|---|---|
| `git grep useErrorDialog ba47553 --` 三路径 | **空**（RegisterView / RegisterPatientPicker / ChargeView 无 match） |
| tip 树 `useErrorDialog.js` | **未跟踪**；`git ls-files` 无此路径；`git ls-tree` feedback 无此文件 |
| Vue diff | 仅删悬空 import/调用；无其它业务逻辑改动 |
| RegisterView banners | **仍在**：`role="alert"` / `class="banner"` / `loadError` / `SubmitStatusBanner` |
| RegisterPatientPicker | **仍在**：`register-picker-error` / `role="alert"` / `error.value` |
| ChargeView | **仍在**：`data-testid="charge-error"` / `role="alert"` / `errorMessage` / `SubmitStatusBanner` |
| 护理壳 / DepartmentsPanel | tip **未改** |
| SubmitStatusBanner 实现 | tip **未改**（仅消费方仍 import 组件） |
| App 挂载 / ERR 全包 / 方案 A | **未做** |
| 零新 API | tip 无 api/** 变更、无新 endpoint |
| FIX-03 / SCORE / PRELIVE | **未做** |
| 未签 90% | tip message + H05 md §0 + 本抽验均明示 |
| H05 md 列 ERR-CLEAN-02 残留 | **有**：DiagnosisView / DispenseView / OrderView（§4） |
| tip 内残留仍挂债 | `git grep` 确认三视图仍有 import + `useErrorDialog([...])` |

## 3) probe

运行：`node deliverables/h05/ui90-err-clean-01-probe-2026-11-01.mjs`

```json
{
  "ok": true,
  "fails": [],
  "cleaned": [
    "web/src/views/RegisterView.vue",
    "web/src/components/RegisterPatientPicker.vue",
    "web/src/views/ChargeView.vue"
  ],
  "residualForErrClean02": [
    "web/src/views/DiagnosisView.vue",
    "web/src/views/DispenseView.vue",
    "web/src/views/OrderView.vue"
  ],
  "helperOnDiskButMustStayUntracked": true,
  "helperTracked": false,
  "note": "Plan B first slice: RegisterView + RegisterPatientPicker + ChargeView; no 90% sign-off; Diagnosis/Dispense/Order deferred"
}
```

## 4) 禁区自检

- [x] 生产 diff 仅三 Vue（−24 接线）+ h05 deliverables
- [x] tip 三路径无 `useErrorDialog`；helper 未入库
- [x] 本地 banner / SubmitStatusBanner 消费路径保留
- [x] 未改护理壳 / SubmitStatusBanner 实现 / App；未 ERR 全包；未方案 A
- [x] 零 Go / 022 / ICD / Dict / DIAG-AUTOSAVE / 新 search API
- [x] 未签 90%；未 SCORE；未混 PRELIVE；未做 FIX-03
- [x] H05 md 标明 ERR-CLEAN-02 残留三视图；tip 树仍挂债
- [x] **本抽验是否改业务 = 否**（仅新增本 md）
- [x] **push = 否**；禁 qq-mail / 未发用户邮件

## 5) 残留 / 备注（非挡）

1. ERR-CLEAN-02：`DiagnosisView` / `DispenseView` / `OrderView` tip 内仍悬空 `useErrorDialog`（H05 已列；另卡）。
2. 工作区仍有 **untracked** `web/src/components/feedback/useErrorDialog.js`（及 ErrorDialog/formValidation WIP）；方案 B 正确未入库。
3. 工作区大量无关 dirty（含 Go/App/shell 等）；抽验只对照 tip `ba47553` 树，精确 add 仅本 md。
4. origin ahead 债存在；本抽验 **不 push**、不 force/reset。

## 6) 回交摘要

| 项 | 值 |
|---|---|
| 结论 | **建议通过** |
| 目标 tip | `ba47553` / `ba47553bb993b54501cf3d91c81578d27541999b` |
| 本抽验是否改业务 | 否 |
| push | 否 |
