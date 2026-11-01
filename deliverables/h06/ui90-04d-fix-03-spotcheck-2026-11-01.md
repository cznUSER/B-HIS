# UI90-04D-FIX-03 抽验（常规 tab 三字段 UI-only · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`dde9183939b1bbd8effa520bdc36a809b0796708`（短 `dde9183`）
- 基线 tip：`d0b5b4b`（`docs(h06): UI90-ERR-CLEAN-02 spotcheck pass (9336943; no 90% sign-off)`）
- HEAD 抽验时：`dde9183`（与目标 tip 一致）
- origin：`0/0`（抽验时）；**本抽验不 push**
- 对照：H14 卡；H05 `deliverables/h05/ui90-04d-fix-03-2026-11-01.md` + `deliverables/h05/ui90-04d-fix-03/`
- 性质：结构/禁区抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **未做 03b/03c** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不把 UI90-04D 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮箱 / 禁 qq-mail。

闭合对象：常规 tab 电话号码 / 负责人 / 助记码三字段 **UI-only**（本地 form；取消丢弃；hint 标明未持久化）。成功 ≠ 90% 签收；编辑区 ≠ 整页。

## 1) tip 范围核对

`git show dde9183 --stat` / `git diff --name-only d0b5b4b..dde9183`：

| 路径 | 角色 |
|---|---|
| `web/src/views/master/DepartmentsPanel.vue` | 常规 tab 增三字段 + cancel/create/edit 清本地三键；**未改** `onSave`→`saveDepartment` 四字段载荷 |
| `deliverables/h05/ui90-04d-fix-03-2026-11-01.md` | H05 交付说明（明示不签 90%） |
| `deliverables/h05/ui90-04d-fix-03/_scoped.css` | 隔离 fixture 样式 |
| `deliverables/h05/ui90-04d-fix-03/geometry-fixture.html` | 隔离几何 fixture |
| `deliverables/h05/ui90-04d-fix-03/geometry-probe.cjs` | 三视口探针 |
| `deliverables/h05/ui90-04d-fix-03/after-results.json` | `allPass=true` |
| `deliverables/h05/ui90-04d-fix-03/after-{1440,1024,390}.png` | 截图证据 |

**零** Go / migrations / 022 / OpenAPI / ICD / Dict 面板 / `web/src/master/api.js` / `store.js` / AppShell / 全局 CSS / 新 search API / nursing API。  
`d0b5b4b..dde9183` 仅 **1** tip；相对 origin **0/0**。WT 大量 dirty/untracked **未混入 tip**。

## 2) 结构验收（对照 tip 树）

| 项 | 结果 |
|---|---|
| 三字段 UI | tip template 有 `dept-phone` / `dept-manager` / `dept-input-code`；标签 电话号码 / 负责人 / 助记码 |
| 未持久化 hint | `data-testid="dept-extra-fields-hint"` 文案含「UI 仅 / 未持久化；不写入 saveDepartment」 |
| `form` 本地三键 | `phone` / `manager` / `inputCode`；注释明示 never in saveDepartment |
| `onSave` 载荷 | **仍仅** `deptCode` / `deptName` / `status` / `sortNo`（tip 源码核对） |
| `departmentBody`（api.js） | tip **未改** api.js；仍只接受四字段 |
| 取消丢弃 | `cancelEdit` 清 `form.phone/manager/inputCode`；探针 `discarded` 三键皆空 |
| create/edit | `startCreate`/`startEdit` 清空三键；edit **不声称**服务端回填 |
| FIX-01 双页签 | `dept-editor-tabs` + 常规 / 护理单元包含科室；`activeTab` 壳仍在 |
| FIX-02 护理 ListBox | `role="listbox"` + 增删 90×30 + UI-only note；本地 `nursingMembers` mock；未扩 nursing API |
| 桌面宽 / 底钮 | CSS `--dept-editor-w: 408px`；`.panel-actions .button` **90×30** |
| 三视口 after-results | 1440/1024 `formW=408`、底钮 90×30、`overflow=false`；390 `formW=342`（max-width:100%）、无横溢；`fieldCount=7`；`savePayloadKeys` 四字段；`allPass=true` |
| formH 增高 | 839（1440/1024）因 4→7 字段，H05 已标可解释偏差；**宽仍 408**，非挡 |
| 未签 90% / 未 SCORE / 未 PRELIVE / 未 03b/03c | tip message + H05 §0 + 本抽验均明示 |

## 3) 禁区自检

- [x] 未 DIAG-AUTOSAVE
- [x] 未官方 ICD/Dict/强制编码/新 search API/022
- [x] 未扩 `saveDepartment` / Go DTO / nursing API
- [x] 未借刀凑分 / 假绿 / 关闭整包 UI90
- [x] 未 mock 充活体（探针为**隔离 fixture**；护理仍标 UI 仅演示）
- [x] 未正式 SCORE / 未混 PRELIVE / 未做满全字段（03b/03c）
- [x] 未改 AppShell / shell-design / design-system / master-design 全局大改
- [x] 未 force/reset；本抽验精确 add 仅本 md；未公网；未 qq-mail
- [x] FIX-01 双页签 + FIX-02 护理壳 **守恒**

## 4) 非挡残留

- 工作区大量 **dirty / untracked**（其它模块/看板/Go 等）。**属 WT WIP，不在 tip 树**；本 tip 与本抽验均未混交。
- formH 因三字段增高属可解释偏差；宽与底钮守恒。
- 本片成功 ≠ 科室整页 90%；03b/03c 后置另派。

## 5) 结论

**建议通过。** tip `dde9183` 相对基线 `d0b5b4b` 仅改 `DepartmentsPanel.vue`（三字段 UI-only + 取消/新建/编辑清本地三键）并交付 H05 md/fixture/probe/after 证据；`saveDepartment` 仍四字段；FIX-01/02 守恒；桌面宽≈408、底钮≈90×30、三视口无新横溢；零 Go/022/ICD/Dict/新 API；未签 90%、未 SCORE、未混 PRELIVE、未做满全字段。本抽验未改业务码、**不 push**。

---

**回交 H01：** 建议通过 · 目标 tip `dde9183` · 本抽验 commit 见同刀 · push=否 · 零业务码。