# UI90-04D-FIX-03c-FIXTURE-01 抽验（隔离 fixture noopPass 对齐 · H06）

- **结论（顶置）：建议通过**
- 岗位：H06（只建议；不定需求、不关任务、不签 90%）
- 抽验时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`（华为笔记本 / `E:\B-HIS`）
- 分支：`codex/register-slot-card-20261028`
- 目标 tip（全）：`da84e464f48719b95b05819fce14ce2a7554bfa0`（短 `da84e46`）
- 基线 tip：`751d070`（`docs(h06): UI90-04D-FIX-03c spotcheck pass (5e16be6; no 90% sign-off)`；实现参考 `5e16be6`）
- HEAD 抽验时：`da84e46`（与目标 tip 一致）
- origin：相对 origin **ahead 1**（本 tip 待推并行债，**本抽验不 push、不 force**）
- 对照：H14 卡；H05 `deliverables/h05/ui90-04d-fix-03c-fixture-01-2026-11-01.md` + `deliverables/h05/ui90-04d-fix-03c/`
- 性质：结构/禁区抽验 · **未签 90%** · **未 SCORE** · **未混 PRELIVE** · **本抽验未改业务码**

## 0) 声明

本 md 仅建议「通过」。不定需求口径、不关任务/看板、不把 UI90-04D 标 ≥90%、不做正式 SCORE、不与 PRELIVE 混刀、不发用户邮箱 / 禁 qq-mail。

闭合对象：仅 **03c 隔离 fixture/probe** 拉回与生产/03b 一致的功能钮 noop + `noopPass` 门禁。成功 ≠ 90% 签收；未改生产 `DepartmentsPanel`；未做真 Dialog / Dict / 新 search。

## 1) tip 范围核对

`git show da84e46 --stat` / `git diff --name-only 751d070..da84e46`：

| 路径 | 角色 |
|---|---|
| `deliverables/h05/ui90-04d-fix-03c-fixture-01-2026-11-01.md` | H05 交付说明（明示不签 90%） |
| `deliverables/h05/ui90-04d-fix-03c/geometry-fixture.html` | 功能钮 click → `/* noop */`；不再 `dialogOpened=true` |
| `deliverables/h05/ui90-04d-fix-03c/geometry-probe.cjs` | 新增 `noopPass` 并入 `pass`/`checks` |
| `deliverables/h05/ui90-04d-fix-03c/after-results.json` | 三视口重跑；`dialogAfterClick=false`；`noopPass=true`；`allPass=true` |

**零** `web/src/**`；**零** Go / migrations / 022 / OpenAPI / ICD / Dict / CodeRules / `master/api.js` / `store.js` / AppShell / 全局 CSS / 新 search API / nursing API。  
父 tip=`751d070`；`751d070..da84e46` 仅 **1** tip、**4** 路径。WT 大量 dirty/untracked **未混入 tip**（禁 force/reset）。

## 2) 验收要点

| 项 | 结果 |
|---|---|
| 三视口 `dialogAfterClick` | 1440/1024/390 均为 **false** |
| `noopPass` | 三视口均为 **true**，并入 `pass` |
| `allPass` | **true**；`pageerrors=[]` |
| 03c 既有门禁未回退 | `mutexPass` / `activeNeStatusPass` / `savePass`（仍四字段）/ `nursingPass` / `hintPass` / `discardPass` / `conservePass` / `controlsPass` / `formWPass` / `btnPass` / `overflowPass` 全 true |
| tip 无 web/src | `git diff --name-only 751d070..da84e46 -- web/src` 为空 |
| 生产 `onFunctionsButtonClick` | 仍为空 noop（FIX-03b 注释）；tip 未触 `DepartmentsPanel.vue`；`saveDepartment` 仍 `{deptCode,deptName,status,sortNo}` |

fixture 关键句：`dept-functions-btn` click → `/* noop — FIX-03c-FIXTURE-01: align production/03b; never set dialogOpened */`。  
probe 关键句：`noopPass = dialogAfterClick === false` 并入 `pass`。

## 3) 几何 / 横溢（对照 after-results）

| 视口 | formW | dialogAfterClick | noopPass | save W×H | overflow | pass |
|---:|---:|---|---|---:|---|---|
| 1440 | 408 | false | true | 90×30 | false | true |
| 1024 | 408 | false | true | 90×30 | false | true |
| 390 | 358 | false | true | 90×30 | false | true |

护理壳守恒：listH=257、增删 90×30；未扩 nursing API。

## 4) 硬禁核对

| 禁区 | 抽验 |
|---|---|
| 改 web/src / 生产 Dialog | tip 无；生产仍空 noop |
| Go / 022 / ICD / Dict / 新 search | tip 无 |
| 扩 save / nursing API | tip 无；探针 save 仍四字段 |
| 签 90% / SCORE / PRELIVE | tip md 明示不做；本抽验亦不做 |
| force / reset / 脏混交 | tip 仅 4 交付路径；WT dirty 未入 tip |
| qq-mail | 本抽验未发任何用户邮件 |

## 5) 建议

**建议通过** tip `da84e46`（全 `da84e464f48719b95b05819fce14ce2a7554bfa0`）。  
H06 不定口径、不关片、不签 90%、不 SCORE、不混 PRELIVE、不 push。

证据路径：本文件；H05 md；`deliverables/h05/ui90-04d-fix-03c/{geometry-fixture.html,geometry-probe.cjs,after-results.json}`。