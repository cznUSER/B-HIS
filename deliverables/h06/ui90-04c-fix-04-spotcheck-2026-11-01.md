# UI90-04C-FIX-04 测量抽验（H06）

## 结论：**建议通过**

- 抽验岗：H06（只建议；不定需求口径、不关任务、不签 90%）
- 目标 tip（全）：`4b2e605dacd81854330f9520b10cb516c2a941a3`（短 `4b2e605`）
- 基线：`9443941`（全 `94439410fc056e6fab94becfc4fc1c504371f2ad`）
- 分支：`codex/register-slot-card-20261028`
- origin：抽验时工作区显示相对 origin **0/0**；本抽验 **未 push**
- 是否改业务：**否**（本抽验仅新增本 md）
- push：**否**
- 时间：2026-11-01（Asia/Shanghai）

## 对照范围

| 项 | 来源 |
|---|---|
| H14 开工卡 | `deliverables/h14/ui90-04c-fix-04-start-card-2026-11-01.md`（磁盘；未入 tip 树） |
| H05 交付 | `deliverables/h05/ui90-04c-fix-04-2026-11-01.md` |
| 测宽/探针 | `deliverables/h05/ui90-04c-fix-04/after-results.json`、`behavior-results.json`、`component-build-results.json` |
| 实现 diff | `git diff 9443941..4b2e605 -- web/src/views/access/UserOperationsLogPanel.vue` |
| 文件清单 | `git show --stat 4b2e605` → 仅 Vue + H05 交付目录（21 files；+815/−8） |

## 对照项 × 证据

### 1) H14：桌面两行；token；主控件 196±2；filters 高 81±2；保留 FIX-03 — **通过**

| 验收点 | 证据 |
|---|---|
| 桌面两行网格 | tip scoped：`grid-template-columns` 两组 `label+gap+ctrl`；`grid-template-rows: 17px 25px 25px`；`record-date` `grid-column: 1 / -1`。`after-results.json`：1440/1024 均 `twoRows=true`，R1 两控件同 top、R2 同 top 且低约 28px |
| 命名 token | `--uol-filters-h: 81px`、`--uol-ctrl-w: 196px`，以及 label/gap/padding 变量均在本组件 scoped 条件区选择器内 |
| 主控件宽 196±2 | 1440/1024：module select / user button-edit / begin / end 宽均为 **196**（±0） |
| filters 高 81±2 | 1440/1024：`filtersHeight=81`；`desktopPass=true` |
| FIX-03 保留 | tip 相对基线 **script/template 逐字节相等**（script 4308、template 5816）；仍含 `readonly`、省略号/`选择用户`、`fetchUsers`、`filters.userId`、`openUserPicker`/`clearUser`、`role="dialog"`。`behavior-results.json`：`readonly/ellipsisSelect/clear=true`；选人后 audit query 含 `userId`，清除后仅去掉 `userId` |

### 2) 禁全页 absolute；禁表列 Width 矩阵；禁新 search API；未软拆泄漏 04b 必做 — **通过**

| 禁项 | 证据 |
|---|---|
| 无全页 absolute | tip 仅 1 处 `position:absolute`：`.user-operations-user-picker`（`top:100%; left:0`），属 FIX-03 既有薄选人层，非整页布局；无 Left/Top 像素追齐（`leftPxHard=false`） |
| 无表列 Width 矩阵 | Vue 无 `colWidth`/`column-width`；commit 未触表列 |
| 无新 search API | commit 无 Go/OpenAPI/迁移；behavior 请求路径仅既有 `GET /api/v1/access/modules`、`/api/v1/audit/user-operations`、`/api/v1/users` |
| 未把 04b 当必做 | 实现用 grid/token 近似两列组；H05 md 明示「没有新增 absolute 定位」「不做 Left/Top 像素追齐」；符合 H14「绝对坐标默认后置 04b」 |

### 3) 不签 90%；未 SCORE；零 Go/022/ICD/Dict/新 search API — **通过**

| 项 | 证据 |
|---|---|
| 不签 90% / 未 SCORE | commit message：`Does NOT sign UI90-04C 90%; no SCORE; no absolute Left chase.`；H05 md §6 诚实声明同口径；本抽验亦不签 |
| 文件边界 | `4b2e605` 仅：`UserOperationsLogPanel.vue` + `deliverables/h05/ui90-04c-fix-04*` |
| 零越界模块 | 无 Go / 022 / ICD / Dict / 全局 design-system 改动 |
| 业务语义 | script/template 相对 `9443941` 不变；筛选/分页/导出/选人语义未漂；`sourceSha256` 与 tip Vue SHA-256 一致：`05186b8acb2f2b70233b9773ef8c5938f688c9736a6a8916143f6a4a87f656c1` |

### 4) commit 范围 / 是否仅 scoped — **通过**

- diff 仅触及 `<style scoped>`（style 6257→8282）；script/template 不变，与 H05「仅 scoped」声明一致。
- 工作区脏树巨大，但目标 tip 内容干净；本抽验仅 stage 本 md。

## 测宽抽核（H05 JSON）

| 视口 | filters 高 | module/user/begin/end 宽 | twoRows | 抽核 |
|---:|---:|---|---|---|
| 1440 | 81 | 196/196/196/196 | true | 合 81±2 / 196±2 |
| 1024 | 81 | 196/196/196/196 | true | 同上 |
| 860 / 520 / 390 | 窄屏自适应（不要求 81/196） | 弹性收缩 | 860 twoRows=true；520/390 false | 与 H14「窄屏不要求像素」一致；`responsivePass=true`、无横向溢出 |

`component-build-results.json`：隔离组件生产构建 `pass=true`。H05 如实披露全量 `build:demo` 因范围外 `RegisterView`/`registerApi` 导出失败——**不构成本片布局抽验否决**（H14 明确非真宿主/SCORE）。

## 越界检查（硬）

- [x] 未改业务实现（抽验侧）
- [x] 未 push
- [x] 未 force / reset / hard
- [x] 未签 90% / 未关任务 / 未改看板口径
- [x] 未发现全页 absolute / 表列 Width 矩阵 / 新 search API
- [x] 未把 FIX-04b 绝对坐标泄漏为本片必做
- [x] FIX-03 行为标记与 behavior JSON 一致保留

## 残留备注（不挡建议通过）

- 本抽验为 **静态 diff + H05 测宽/行为 JSON 抽核**，未重跑 Playwright / 真宿主；符合 H14「H06 测量+禁区抽查」薄口径。
- 全量应用构建失败属范围外基线债，交编排器；不以局部测宽代替 SCORE。

## 建议

**建议通过** UI90-04C-FIX-04（tip `4b2e605`）。是否关单 / 是否另立 FIX-04b 或 SCORE 由 H01 定；本岗不定口径、不关任务、不签 90%。
