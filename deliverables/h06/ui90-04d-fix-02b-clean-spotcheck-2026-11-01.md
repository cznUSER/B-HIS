# UI90-04D-FIX-02b CLEAN 抽验（删悬空 useErrorDialog · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`33a81eeeb143fd36b7f04eb37d4a3edaf559370c`（短 `33a81ee`）
- 基线 tip：`6eeb2d1`（FIX-02 抽验 · 有条件通过）
- HEAD 抽验时：`33a81ee`（与目标 tip 一致）
- origin：`0/2`（ahead 债；**本抽验不 push**）
- 对照：H14 方案 B；H05 `deliverables/h05/ui90-04d-fix-02b-clean-2026-11-01.md` + probe
- 性质：禁区/可解析抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **未做 FIX-03** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务、不把 UI90-04D 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮箱 / 禁 qq-mail。

闭合对象：FIX-02 抽验之条件项——`DepartmentsPanel` tip 树不再依赖缺失的 `useErrorDialog.js`。成功 ≠ 90% 签收。

## 1) tip 范围核对

`git show 33a81ee --stat` / `git diff 6eeb2d1..33a81ee --name-status`：

| 路径 | 角色 |
|---|---|
| `web/src/views/master/DepartmentsPanel.vue` | 唯一生产源码（**−5**：删 import + `useErrorDialog([...])` 调用） |
| `deliverables/h05/ui90-04d-fix-02b-clean-2026-11-01.md` | H05 交付说明 |
| `deliverables/h05/ui90-04d-fix-02b-clean-probe-2026-11-01.mjs` | 静态 probe |

**零** Go / migrations / 022 / OpenAPI / ICD / Dict / AppShell / shell-design / `master/api.js` / 新 search API。  
未入库 `useErrorDialog.js`；未覆盖 EVID；未做 ERR 全包（ErrorDialog + formValidation + App/main）。

基线祖先关系：`6eeb2d1` ⊂ `33a81ee`（`merge-base --is-ancestor` OK）。

## 2) 方案 B 验收（对照 tip 树）

| 项 | 结果 |
|---|---|
| `git grep useErrorDialog 33a81ee -- web/src/views/master/DepartmentsPanel.vue` | **空**（no match） |
| tip 树 `useErrorDialog.js` | **无**；`git ls-tree … feedback/` 仅 `SubmitStatusBanner.vue` / `useSubmitFeedback.js` |
| Vue diff | 仅删除 L2 import 与 `useErrorDialog([() => loadError.value, () => noticeError.value ? notice.value : ''])`；无其它业务逻辑改动 |
| loadError / notice 横幅 | **仍在**：`data-testid="dept-notice"`、`data-testid="dept-load-error"`；`loadError` / `notice` / `noticeError` / `errorCode` ref 保留 |
| 干净检出可解析 | tip 内不再引用缺失 helper；悬空 import 条件项闭合 |
| FIX-01 双页签 | 「常规」/「护理单元包含科室」；`activeTab` 默认 `normal`；四字段仅在常规 tabpanel；`--dept-editor-w: 408px` |
| 护理壳（FIX-02） | listbox + 增删 chrome；`dept-nursing-list/add/del/note/empty/actions` 仍在；token `--dept-nursing-list-h: 257`、`--dept-nursing-btn-w/h: 90/30`；本地 `nursingMembers`；`resetNursingDraft` 仍挂 cancel/create/edit/save |
| `saveDepartment` | payload 仍 `deptCode` / `deptName` / `status` / `sortNo`；toggle 仍仅 `{ status }`；**未扩** |
| 零新 API | tip 无新 fetch/nursing endpoint；仅改 Vue −5 + deliverables |
| FIX-03 / SCORE / PRELIVE / ERR 全包 | **未做** |
| 未签 90% | tip message + H05 md + 本抽验均明示 |

## 3) probe

运行：`node deliverables/h05/ui90-04d-fix-02b-clean-probe-2026-11-01.mjs`

```json
{
  "ok": true,
  "fails": [],
  "helperExistsOnDiskButMustStayUntracked": true,
  "note": "Plan B: do not git-add useErrorDialog.js; banners + nursing shell conserved; no 90% sign-off"
}
```

## 4) 禁区自检

- [x] 生产 diff 仅 `DepartmentsPanel.vue`（−5）+ h05 deliverables
- [x] tip 无 `useErrorDialog` 引用 / 未入库 helper
- [x] 横幅恢复保留；护理壳 / FIX-01 / `saveDepartment` 四字段守恒
- [x] 零 Go / 022 / ICD / Dict / DIAG-AUTOSAVE / 新 search API
- [x] 未签 90%；未 SCORE；未混 PRELIVE；未做 FIX-03；未做 ERR 全包
- [x] **本抽验是否改业务 = 否**（仅新增本 md）
- [x] **push = 否**；禁 qq-mail / 未发用户邮件

## 5) 残留 / 备注（非挡）

1. 工作区仍有 **untracked** `web/src/components/feedback/useErrorDialog.js`（及 ErrorDialog/formValidation WIP）；属 ERR-CLEAN 另卡，本片正确未入库。
2. 他页（Register/Charge/Diagnosis 等）预存悬空 `useErrorDialog` import 债 **未**在本 tip 清；另卡处理，非本片否决。
3. 工作区大量无关 dirty；抽验只对照 tip `33a81ee` 树，精确 add 仅本 md。
4. origin ahead `0/2`（含本实现 tip）；本抽验 **不 push**、不 force/reset。

---

**回交 H01：** 建议**通过**；目标 tip `33a81ee`；本抽验 tip 见 commit；业务码未改；push=否。
