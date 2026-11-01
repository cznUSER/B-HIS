# UI90-04D-FIX-03c 抽验（四单选/接口/导诊/激活 UI-only 壳 · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`5e16be6f0f0a09600b4f445a6b992ec2ce164734`（短 `5e16be6`）
- 基线 tip：`03b8528`（`docs(h06): UI90-04D-FIX-03b spotcheck pass (dea804c; no 90% sign-off)`）
- HEAD 抽验时：`5e16be6`（与目标 tip 一致）
- origin：`0/0`（`03b8528..5e16be6` ahead/behind 据核 1/0 相对基线；相对 origin 0/0）；**本抽验不 push、不 force**
- 对照：H14 卡；H05 `deliverables/h05/ui90-04d-fix-03c-2026-11-01.md` + `deliverables/h05/ui90-04d-fix-03c/`
- 性质：结构/禁区抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不把 UI90-04D 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮箱 / 禁 qq-mail。

闭合对象：常规 tab **科室类型四单选（普通/儿科/急诊/中医）+ 接口(一)(二) + 导诊提示 + 激活标记**。成功 ≠ 90% 签收；编辑区 ≠ 整页；激活标记 ≠ 启停 status；未接 CodeRules / 真 Dict Lookup / 新 search API / nursing API。

## 1) tip 范围核对

`git show 5e16be6 --stat` / `git diff --name-only 03b8528..5e16be6`：

| 路径 | 角色 |
|---|---|
| `web/src/views/master/DepartmentsPanel.vue` | 常规 tab 增五键壳 + cancel/create/edit 清本地五键；四单选互斥默认普通；接口/导诊本地文本；激活 checkbox 独立于 status；**未改** `onSave`→`saveDepartment` 四字段载荷；**未改写** `deptCode` |
| `deliverables/h05/ui90-04d-fix-03c-2026-11-01.md` | H05 交付说明（明示不签 90%） |
| `deliverables/h05/ui90-04d-fix-03c/_scoped.css` | 隔离 fixture 样式 |
| `deliverables/h05/ui90-04d-fix-03c/geometry-fixture.html` | 隔离几何 fixture |
| `deliverables/h05/ui90-04d-fix-03c/geometry-probe.cjs` | 三视口探针 |
| `deliverables/h05/ui90-04d-fix-03c/after-results.json` | `allPass=true` |
| `deliverables/h05/ui90-04d-fix-03c/after-{1440,1024,390}.png` | 截图证据 |

**零** Go / migrations / 022 / OpenAPI / ICD / Dict 面板 / CodeRules 接线 / `web/src/master/api.js` / `store.js` / AppShell / 全局 CSS / 新 search API / nursing API。  
`03b8528..5e16be6` 仅 **1** tip；相对 origin **0/0**。WT 大量 dirty/untracked **未混入 tip**（精确 9 路径；禁 force/reset）。

## 2) 结构验收（对照 tip 树）

| 项 | 结果 |
|---|---|
| 四单选 rbtnNormal/EK/JZ/ZY | `name="deptKind"` + `v-model="form.deptKind"`；testid `dept-kind-normal/ek/jz/zy`；默认 `normal`；探针 `mutexPass`（选儿科后仅 ek，deptCode 仍 D001） |
| 未改 deptCode | 四单选不写 `form.deptCode`；save 仍用原 `form.deptCode`；无 CodeRules/强制编码 |
| 接口一二 | `dept-iface-1/2` 绑 `expandInterface1/2`；title 标明 UI 壳 / 未接主数据 Dict |
| 导诊 | `dept-guide-info` 绑 `guidInfo`；本地文本；未接主数据 |
| 激活 chkIsActive | `dept-active-mark` 绑 `isActiveMark`（默认 false）；**独立于** `form.status`；hint/title 明示 ≠ status；探针 `activeNeStatusPass`（勾选后 status 仍 ACTIVE） |
| hint 未持久化 | `dept-extra-fields-hint` 合并声明电话/负责人/助记码/自动编码/功能/分类/四单选/接口/导诊/激活为 UI 壳、不写 save、激活≠status、接口导诊未接主数据 |
| saveDepartment 四字段 | tip 内 `onSave` 仍 `{ deptCode, deptName, status, sortNo }`；探针 `savePayloadKeys`/`saveKeys` 同左 |
| 取消丢弃 | cancel/create/edit 清空五键（及 FIX-03/03b 键）并 `deptKind='normal'`；探针 `afterCancel` 全空/默认 |
| FIX-01 双页签 | `dept-tab-normal` / `dept-tab-nursing` 文案与壳仍在 |
| FIX-02 护理 ListBox | listH=257、增删 90×30、mock note 仍声明未持久化；**未扩** nursing API |
| FIX-03 三字段 | 电话/负责人/助记码仍在 |
| FIX-03b 三壳 | 自动编码 disabled+readonly；功能 ButtonEdit + 生产 `onFunctionsButtonClick` 空 noop；分类本地静态 select；fieldCount=15 |

**证据注**：隔离 fixture 点功能钮会把 `__state.dialogOpened=true`（`dialogAfterClick=true`），**未入 pass 门禁**；生产 Vue 仍为 FIX-03b 空 noop，**不据此降档**。

## 3) 几何 / 横溢（对照 H05 after-results）

| 视口 | formW | 保存 W×H | overflow | pass |
|---:|---:|---:|---|---|
| 1440 | **408**（±0，落在 408±2） | **90×30** | false | true |
| 1024 | **408** | **90×30** | false | true |
| 390 | 358（max-width:100% 可解释） | **90×30** | false | true |

- `allPass=true`；`pageerrors=[]`
- formH=1211（桌面）/1247（390）因字段增至 15，属可解释偏差；未动 AppShell/全局 CSS
- 护理壳守恒：listH=257、add/del 90×30
- scoped 仍 `--dept-editor-w: 408px`；底钮 `width: 90px; height: 30px`

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| DIAG-AUTOSAVE | tip 无涉及 |
| 官方 ICD / Dict / 强制编码 / CodeRules | tip 无接线；四单选不改 deptCode；接口/导诊本地壳 |
| 激活绑 status | `isActiveMark` 与 `form.status` 分立；无赋值串联；探针 status 勾选前后不变 |
| 公网 | 无 |
| force / reset | 本抽验未做；H05 tip 范围干净 |
| 脏混交 | tip 仅 9 路径；WT dirty 未入 tip |
| 借刀凑分 / mock 充活体 | 探针为隔离 fixture；交付明示 UI 壳 |
| 扩 Go/DTO/022/新 search/nursing API | tip `name-only` 无此类文件 |
| 签 90% / SCORE / 混 PRELIVE | tip message 与 H05 md 均明示否；本抽验同否 |

## 5) 本抽验动作

- 读 tip 树 + H05 md + `after-results.json` + Vue diff
- **未改** `DepartmentsPanel.vue` 或其它业务码
- 仅新增本 md；精确 `git add` 该路径；commit；**不 push**

## 6) Git（抽验 commit 回填）

- 分支：`codex/register-slot-card-20261028`
- 实现 tip：`5e16be6` / `5e16be6f0f0a09600b4f445a6b992ec2ce164734`
- 抽验 commit：见本提交后 `git rev-parse HEAD`
- push：**否**

---

H06 抽验岗：**建议通过**。不关片、不签 90%、不做正式 SCORE、不混 PRELIVE、不发 qq-mail。
