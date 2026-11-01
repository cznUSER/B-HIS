# UI90-04D-FIX-01 · 双页签壳结构/禁区抽验（H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`2f9057b35d5ace04965173fc56c69710e37ec8a4`（短 `2f9057b`）
- 基线 tip：`7f9c4fd`
- HEAD 抽验时：`2f9057b`（与目标 tip 一致；工作区另有无关 dirty，**未**纳入本抽验 commit）
- origin：任务所述 0/0；**本抽验不 push**
- 性质：结构/禁区抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务、不把 UI90-04D 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀。

## 1) tip 范围核对

`git show 2f9057b --stat` / `git diff 7f9c4fd..2f9057b --name-only`：

| 路径 | 角色 |
|---|---|
| `web/src/views/master/DepartmentsPanel.vue` | 唯一生产源码 |
| `deliverables/h05/ui90-04d-fix-01-2026-11-01.md` | H05 交付说明 |
| `deliverables/h05/ui90-04d-fix-01/*` | 隔离 fixture / probe / JSON / PNG |

**零** Go / migrations / 022 / ICD / Dict / AppShell / shell-design / design-system / master-design 全局 / 新 search API。  
EVID 包仍在 tip `7f9c4fd`：`deliverables/h06/ui90-04d-evid-pack-2026-11-01.md` 与 `ui90-department-evidence/` **未被本 tip 覆盖删除**。

## 2) 结构验收（对照 tip 内 Vue）

| 项 | 结果 |
|---|---|
| 双页签文案 | **常规** / **护理单元包含科室**（`dept-tab-normal` / `dept-tab-nursing`） |
| 默认 | `activeTab = ref('normal')`；`startCreate` / `startEdit` 均重置为 `normal` |
| 四字段范围 | 科室代码 / 名称 / 状态 / 排序 + field-hint **仅在** `v-show="activeTab === 'normal'"` 的 tabpanel |
| 护理 tab | 静态占位「本片未实现护理单元子列表（FIX-02 后置；禁 CRUD）」；**无**子列表 / table / 增删改 CRUD 控件 |
| `saveDepartment` | `onSave` 仍传 `deptCode`/`deptName`/`status`/`sortNo`；toggle 仍仅 `{ status }`；**未扩字段**；`load`/`fetchDepartments` 契约未改 |
| 底钮位置 | 保存/取消仍在页签外 `.panel-actions` |

## 3) 测表（H05 `after-results.json` · 隔离 fixture）

探针：`deliverables/h05/ui90-04d-fix-01/geometry-probe.cjs` → `allPass=true`，`pageerrors=[]`。

| 视口 | formW | tabBarH | save | cancel | overflow | width/btn/tab/nursing/overflow |
|---:|---:|---:|---:|---:|---|---|
| 1440 | **408**（±2 内） | 31（30±2） | **90×30** | **90×30** | false | 全 true |
| 1024 | **408** | 31 | **90×30** | **90×30** | false | 全 true |
| 390 | 342（max-width 收缩；非桌面 408 硬绑） | 31 | **90×30** | **90×30** | false | 全 true（无新横溢） |

护理态抽测（JSON `nursing.*`）：三视口 `nursingCrudButtons=0`、`nursingHasTable=false`、占位文案在场。

截图：`after-1440.png` / `after-1024.png` / `after-390.png`（fixture，非真宿主活体；本刀为 Morph 结构抽验，可接受）。

## 4) 禁区自检

- [x] 仅 `DepartmentsPanel.vue` + h05 deliverables
- [x] 未扩 `saveDepartment`；护理无 CRUD
- [x] 零 Go / 022 / ICD / Dict / DIAG-AUTOSAVE / 新 search API
- [x] 未覆盖 EVID / `ui90-department-evidence`
- [x] 未签 90%；未 SCORE；未混 PRELIVE
- [x] **本抽验是否改业务 = 否**（仅新增本 md）
- [x] **push = 否**

## 5) 残留 / 备注（非否决）

1. **formH≈517** vs DFM ClientHeight 414：页签条 + 护理空壳 `min-height` 可解释增高；本验收口径未把绝对 formH 列为硬门；完整高度像素克隆不在 FIX-01 范围。
2. 几何证据为 **隔离 fixture**，非 `127.0.0.1` 真宿主；结构/禁区抽验足够，正式 SCORE/PRELIVE 后置。
3. 工作区存在大量与 tip 无关的 dirty；抽验仅对照 tip `2f9057b` 树，未把 dirty 当作实现。

## 6) 回报摘要（给 H01）

- **建议：通过**
- 目标 tip：`2f9057b` / `2f9057b35d5ace04965173fc56c69710e37ec8a4`
- 交付：`deliverables/h06/ui90-04d-fix-01-spotcheck-2026-11-01.md`（本文件）
- 是否改业务：否
- push：否
- 简要测表：桌面宽 408、底钮 90×30、三视口无横溢、双页签默认常规、护理空壳无 CRUD、saveDepartment 未扩

---

H06：**只建议通过**；不定口径、不关任务、不签 90%。
