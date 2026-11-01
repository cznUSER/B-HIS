# UI90-04D-FIX-03b 抽验（自动编码/功能/分类 UI-only 壳 · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`dea804cdb566b2217b73ad3b20194a8ea20295eb`（短 `dea804c`）
- 基线 tip：`fe69bb3`（`docs(h06): UI90-04D-FIX-03 spotcheck pass (dde9183; no 90% sign-off)`）
- HEAD 抽验时：`dea804c`（与目标 tip 一致）
- origin：`0/1`（ahead 1；443 推债并行中）；**本抽验不 push、不 force**
- 对照：H14 卡；H05 `deliverables/h05/ui90-04d-fix-03b-2026-11-01.md` + `deliverables/h05/ui90-04d-fix-03b/`
- 性质：结构/禁区抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **未做 03c** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不把 UI90-04D 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮箱 / 禁 qq-mail。

闭合对象：常规 tab **自动编码（只读流水壳）+ 科室功能（ButtonEdit noop）+ 科室分类（本地 select）**。成功 ≠ 90% 签收；编辑区 ≠ 整页；未接 CodeRules / 真 Dict Lookup / 新 search API。

## 1) tip 范围核对

`git show dea804c --stat` / `git diff --name-only fe69bb3..dea804c`：

| 路径 | 角色 |
|---|---|
| `web/src/views/master/DepartmentsPanel.vue` | 常规 tab 增三壳 + cancel/create/edit 清本地三键；`onFunctionsButtonClick` noop；**未改** `onSave`→`saveDepartment` 四字段载荷；**未改写** `deptCode` |
| `deliverables/h05/ui90-04d-fix-03b-2026-11-01.md` | H05 交付说明（明示不签 90%） |
| `deliverables/h05/ui90-04d-fix-03b/_scoped.css` | 隔离 fixture 样式 |
| `deliverables/h05/ui90-04d-fix-03b/geometry-fixture.html` | 隔离几何 fixture |
| `deliverables/h05/ui90-04d-fix-03b/geometry-probe.cjs` | 三视口探针 |
| `deliverables/h05/ui90-04d-fix-03b/after-results.json` | `allPass=true` |
| `deliverables/h05/ui90-04d-fix-03b/after-{1440,1024,390}.png` | 截图证据 |

**零** Go / migrations / 022 / OpenAPI / ICD / Dict 面板 / CodeRules 接线 / `web/src/master/api.js` / `store.js` / AppShell / 全局 CSS / 新 search API / nursing API。  
`fe69bb3..dea804c` 仅 **1** tip；相对 origin **0/1**。WT 大量 dirty/untracked **未混入 tip**（精确 9 路径；禁 force/reset）。

## 2) 结构验收（对照 tip 树）

| 项 | 结果 |
|---|---|
| edtSerialNo / `dept-auto-code` | `disabled` + `readonly` + `aria-readonly`；绑定 `form.serialNo`；title 标明只读流水壳、不改写科室代码、未接编码规则 |
| 未改 deptCode | `serialNo` 与 `deptCode` 独立；save 仍用 `form.deptCode`；无 CodeRules/强制编码接线 |
| edtFunctions ButtonEdit | 文本 + `dept-functions-btn`；`onFunctionsButtonClick` 空函数（注释：noop / 无 Dict / 无 dialog / 无 search API） |
| edtDepartmentAttr 本地 select | 静态 option：「—」+ 3 占位（`ui-placeholder-*`）；title 标明非主数据 Lookup |
| hint 未持久化 | `dept-extra-fields-hint` 合并声明电话/负责人/助记码/自动编码/功能/分类为 UI 壳、不写 save、功能/分类未接主数据 |
| saveDepartment 四字段 | tip 内 `onSave` 仍 `{ deptCode, deptName, status, sortNo }`；探针 `savePayloadKeys` 同左 |
| 取消丢弃 | cancel/create/edit 清空 `serialNo/functions/departmentAttr`（及 FIX-03 三键）；探针 `afterCancel` 全空 |
| FIX-01 双页签 | `dept-tab-normal` / `dept-tab-nursing` 文案与壳仍在 |
| FIX-02 护理 ListBox | listH=257、增删 90×30、mock note 仍声明未持久化；**未扩** nursing API |
| FIX-03 三字段 | 电话/负责人/助记码仍在 field-grid；fieldCount=10（7→10 含本片三壳） |

## 3) 几何 / 横溢（对照 H05 after-results）

| 视口 | formW | 保存 W×H | overflow | pass |
|---:|---:|---:|---|---|
| 1440 | **408**（±0，落在 408±2） | **90×30** | false | true |
| 1024 | **408** | **90×30** | false | true |
| 390 | 358（max-width:100% 可解释） | **90×30** | false | true |

- `allPass=true`；`pageerrors=[]`
- 功能按钮点后 `dialogAfterClick=false`（noop 成立）
- formH=952 因字段增至 10，属可解释偏差；未动 AppShell/全局 CSS
- 护理壳守恒：listH=257、add/del 90×30

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| DIAG-AUTOSAVE | tip 无涉及 |
| 官方 ICD / Dict / 强制编码 / CodeRules | tip 无接线；功能 noop；分类本地静态 |
| 公网 | 无 |
| force / reset | 本抽验未做；H05 tip 范围干净 |
| 脏混交 | tip 仅 9 路径；WT dirty 未入 tip |
| 借刀凑分 / mock 充活体 | 探针为隔离 fixture；交付明示 UI 壳 |
| 扩 Go/DTO/022/新 search/nursing API | tip `name-only` 无此类文件 |
| 签 90% / SCORE / 混 PRELIVE / 做 03c | tip message 与 H05 md 均明示否；本抽验同否 |

## 5) 本抽验动作

- 读 tip 树 + H05 md + `after-results.json`
- **未改** `DepartmentsPanel.vue` 或其它业务码
- 仅新增本 md；精确 `git add` 该路径；commit；**不 push**

## 6) Git（抽验 commit 回填）

- 分支：`codex/register-slot-card-20261028`
- 实现 tip：`dea804c` / `dea804cdb566b2217b73ad3b20194a8ea20295eb`
- 抽验 commit：见本提交后 `git rev-parse HEAD`
- push：**否**（推债并行；禁 force）

---

H06 抽验岗：**建议通过**。不关片、不签 90%、不做正式 SCORE、不混 PRELIVE、不发 qq-mail。
