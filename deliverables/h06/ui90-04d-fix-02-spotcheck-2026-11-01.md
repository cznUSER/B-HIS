# UI90-04D-FIX-02 抽验（护理列表壳 / 禁区 · H06）

- **结论（顶置）：建议有条件通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`b87e350ef2a7f717ebadedfd8fb4b77c7d3742b1`（短 `b87e350`）
- 基线 tip：`d96b21f`（FIX-01 抽验）
- HEAD 抽验时：`b87e350`（与目标 tip 一致；工作区另有无关 dirty，**未**纳入本抽验 commit）
- origin：`0/0`（抽验时 `rev-list --left-right --count origin/...HEAD`）；**本抽验不 push**
- 对照：H14 `deliverables/h14/ui90-04d-fix-02-start-card-2026-11-01.md`；H05 `deliverables/h05/ui90-04d-fix-02-2026-11-01.md`
- 性质：结构/禁区抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **未做 FIX-03** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「有条件通过」。不定需求口径、不关任务、不把 UI90-04D 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮箱。

**条件（非否决护理壳本身）：** tip 内 `DepartmentsPanel.vue` 新增 `import { useErrorDialog } from '../../components/feedback/useErrorDialog.js'` 与 `useErrorDialog([...])` 调用，但 tip 树 `web/src/components/feedback/` **仅有** `SubmitStatusBanner.vue` / `useSubmitFeedback.js`；`useErrorDialog.js` 仅见于工作区 untracked WIP。干净检出 tip `b87e350` 时科室面板可能模块解析失败。建议 H01 另派极薄清理（删该 import/调用，或授权入库 helper），**不**把本条件当成护理 ListBox/增删验收失败。

## 1) tip 范围核对

`git show b87e350 --stat` / `git diff d96b21f..b87e350 --name-only`：

| 路径 | 角色 |
|---|---|
| `web/src/views/master/DepartmentsPanel.vue` | 唯一生产源码 |
| `deliverables/h05/ui90-04d-fix-02-2026-11-01.md` | H05 交付说明 |
| `deliverables/h05/ui90-04d-fix-02/*` | 隔离 fixture / probe / JSON / PNG / `_scoped.css` |

**零** Go / migrations / 022 / OpenAPI / ICD / Dict / AppShell / shell-design / design-system / master-design 全局 / `web/src/master/api.js` 变更 / 新 search API。  
EVID / `ui90-department-evidence/` / FIX-01 证据 **未被本 tip 覆盖删除**。

## 2) 结构验收（对照 tip 内 Vue + H14 卡）

| 项 | 结果 |
|---|---|
| 护理 listbox | `role="listbox"` · `data-testid="dept-nursing-list"`；空态「（空表 · 未持久化）」；演示行文案含「未持久化 / mock」 |
| 增加 / 删除 chrome | 文案「增加」「删除」；token `--dept-nursing-btn-w/h: 90/30`；层次在列表下、页签内 |
| 列表高 | `--dept-nursing-list-h: 257px`（对齐 DFM `lbSubList` 高） |
| 本地薄状态 | `nursingMembers` / `nursingSelectedIndex`；`addNursingMember` / `delNursingMember` 仅改 ref |
| `resetNursingDraft` | `cancelEdit`、`startCreate`、`startEdit`、`onSave` 成功关闭后均调用 |
| 零新 API / 未触 nursing API | 无新 `fetch*` / 无 nursing endpoint import；`master/api.js` tip 无 diff |
| `saveDepartment` 未扩 | payload 仍 `deptCode`/`deptName`/`status`/`sortNo`；toggle 仍仅 `{ status }` |
| FIX-01 守恒 | 双页签「常规」/「护理单元包含科室」；`activeTab` 默认 `normal`；四字段仅在常规 tabpanel；页签外保存/取消仍在 |
| FIX-03 | **未做**（常规仍四字段，无新增业务字段） |
| 占位移除 | FIX-01「本片未实现…禁 CRUD」占位已替换为列表壳；CRUD **未**落库 |

## 3) 测表（H05 `after-results.json` · 隔离 fixture）

探针：`deliverables/h05/ui90-04d-fix-02/geometry-probe.cjs` → `allPass=true`，`pageerrors=[]`。

| 视口 | formW | listH | 增加 W×H | 删除 W×H | save/cancel | overflow | nursing/local/width/overflow |
|---:|---:|---:|---:|---:|---:|---|---|
| 1440 | **408**（±2） | **257** | **90×30** | **90×30** | **90×30** | false | 全 true |
| 1024 | **408** | **257** | **90×30** | **90×30** | **90×30** | false | 全 true |
| 390 | 342（`max-width:100%` 收缩） | **257** | **90×30** | **90×30** | **90×30** | false | 全 true（无新横溢） |

本地增删抽测（JSON）：加 2 行（标签含「未持久化 / mock」）→ 删 1 行；`localPass=true`；`hasTable=false`；占位文案已去。

截图：`after-1440.png` / `after-1024.png` / `after-390.png`（fixture，非真宿主活体；Morph 结构抽验可接受）。

## 4) 禁区自检

- [x] 生产 diff 仅 `DepartmentsPanel.vue` + h05 deliverables
- [x] 未扩 `saveDepartment`；未新 nursing API；未改 `master/api.js`
- [x] 零 Go / 022 / ICD / Dict / DIAG-AUTOSAVE / 新 search API
- [x] 未覆盖 EVID / `ui90-department-evidence` / FIX-01 证据
- [x] 未签 90%；未 SCORE；未混 PRELIVE；未做 FIX-03
- [x] **本抽验是否改业务 = 否**（仅新增本 md）
- [x] **push = 否**；禁 qq-mail / 未发用户邮件
- [~] 附带 `useErrorDialog` 接线 → 见 §0 条件（非护理壳否决）

## 5) 残留 / 备注（非否决护理壳；条件见 §0）

1. **`useErrorDialog` 未入库**：tip 引用 untracked `web/src/components/feedback/useErrorDialog.js`；干净检出风险。建议极薄清理另派，**不**并入 FIX-03/SCORE。
2. 几何证据为 **隔离 fixture**，非 `127.0.0.1` 真宿主；结构/禁区足够，正式 SCORE/PRELIVE 后置。
3. 护理 list 实测宽随编辑壳（如 1440≈382）小于 DFM `lbSubList` 宽 404；本卡验收以 ListBox 语义 + 高 257 + 增删 90×30 为主，宽偏差可解释，非挡。
4. 工作区大量无关 dirty；抽验只对照 tip `b87e350` 树。

## 6) 回报摘要（给 H01）

- **建议：有条件通过**（条件 = 清理/入库 `useErrorDialog` 接线，见 §0）
- 目标 tip：`b87e350` / `b87e350ef2a7f717ebadedfd8fb4b77c7d3742b1`
- 交付：`deliverables/h06/ui90-04d-fix-02-spotcheck-2026-11-01.md`（本文件）
- 是否改业务：否
- push：否
- 简要测表：桌面宽 408、listH 257、增删 90×30、三视口无横溢、listbox+本地增删+resetNursingDraft、saveDepartment 未扩、FIX-01 守恒
- 残留：useErrorDialog 未入库（条件）；fixture 非活体；list 宽 < DFM 404（可解释）

---

H06：**只建议有条件通过**；不定口径、不关任务、不签 90%。
