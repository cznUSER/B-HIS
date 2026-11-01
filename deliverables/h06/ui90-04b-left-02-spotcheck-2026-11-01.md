# UI90-04B-LEFT-02 抽验（H06）

## 结论：**建议通过**

- 抽验岗：H06（只建议；不定需求口径、不关任务、不签 90%）
- 目标 tip（全）：`ef4628631225c1ff37ed606ebb9b29e04524c832`（短 `ef46286`）
- 基线：`e673055`（全 `e6730551f8d04fd2cafd54b29019940237ea4375`）
- 抽验时 HEAD：`87ef44f`（含 H01 开工 docs；tip 之后 1 个 docs commit）
- 分支：`codex/register-slot-card-20261028`
- origin：抽验时相对 origin **ahead 2**（H01 在推；本抽验 **未 push**）
- 是否改业务：**否**（本抽验仅新增本 md）
- push：**否**
- 时间：2026-11-01（Asia/Shanghai）

## 对照范围

| 项 | 来源 |
|---|---|
| H14 开工卡 | `deliverables/h14/ui90-04b-left-02-start-card-2026-11-01.md` |
| H05 交付 | `deliverables/h05/ui90-04b-left-02-2026-11-01.md` |
| 测点/探针 | `deliverables/h05/ui90-04b-left-02/after-results.json`、`geometry-probe.cjs` |
| 实现 diff | `git diff e673055..ef46286 -- web/src/views/access/UserOperationsLogPanel.vue` |
| 文件清单 | `git show --stat ef46286` → 仅 Vue + H05 交付目录（12 files） |

## 对照项 × 证据

### 1) H14：相对 header returnL/titleL/searchR/exportR；headerH45；btnH30；LEFT-01 守恒 — **通过**

| 验收点 | 证据 |
|---|---|
| 具名 header token | tip scoped `:deep(.master-panel-header)`：`--uol-hdr-left-return:15px`、`--uol-hdr-left-title:110px`、`--uol-hdr-left-search:813px`、`--uol-hdr-left-export:909px`；实测策略 `--uol-hdr-right-search:105px`、`--uol-hdr-right-export:9px`（H14 允许 right 等价） |
| 桌面相对 header 绝对位 | `@media (min-width:861px)`：`uol-hdr-return/search/export` + `.master-panel-copy` `position:absolute`；header `position:relative`；≤860 回退 `position:static` / flex |
| returnL=15 / titleL=110 / searchR=105 / exportR=9（±2） | `after-results.json` 1440/1024/861：return.left=**15**、title.left=**110**、search.right=**105**、export.right=**9**；`headerPass=true` |
| headerH=45±2；btnH=30±2 | 同三视口：header.height=**45**；return/search/export height=**30**；既有 `.button` `height:30px` 守恒 |
| LEFT-01 filters 守恒 | 同三视口：filters 目标 Left/Top **108/6、183/6、385/6、431/6、108/37、183/37、385/37、431/37**；四主控宽 **196**；filtersH≈**81**；`left01Pass=true` |
| FIX-03 / 业务语义 | tip 相对基线 **script 逐字节相等**；template 仅加 `uol-hdr-*` 定位 class（剥离后与基线一致）。未改选人/清除/fetchUsers/filters.userId |

### 2) 无全局 MasterPanelHeader；无整页壳 absolute；无表列 Width；无新 search API — **通过**

| 禁项 | 证据 |
|---|---|
| 未改 MasterPanelHeader.vue | `git diff --name-only e673055..ef46286` **无** `MasterPanelHeader`；仅本页 `:deep` + actions class |
| 无整页壳 absolute | absolute 仅限本页 header 命名 chrome（本片准许）+ LEFT-01 filters `uol-abs-*` + 既有 user-picker；无 AppShell / 1008 整页壳 |
| 无表列 Width 矩阵 | Vue/commit 未触结果表列 Width |
| 无新 search API | commit 无 Go/OpenAPI/迁移；script 未改；仍用既有检索/选人路径 |

### 3) 不签 90%；未 SCORE；零 Go/022/ICD/Dict — **通过**

| 项 | 证据 |
|---|---|
| 不签 90% / 未 SCORE | commit message：`no 90% sign-off`；H05 md 明示「本片不签 UI90-04C / UI90-04B 90%；未 SCORE」；本抽验亦不签 |
| 文件边界 | `ef46286` 仅：`UserOperationsLogPanel.vue` + `deliverables/h05/ui90-04b-left-02*` |
| 零越界模块 | 无 Go / 022 / ICD / Dict / 全局 design-system / access-control / master-design |
| sourceSha256 | JSON `0179372849c669c4b1ee93f35b418446047b9f0d5fa50e5296f121905ffad500` 与 tip Vue SHA-256 **一致** |

## 测点抽核（相对 header；对照 H05 / JSON）

桌面 header（容差 ±2；实测精确相等）：

| 视口 | headerH | returnL | titleL | searchR | exportR | btnH | headerPass |
|---:|---:|---:|---:|---:|---:|---|---|
| 1440 | 45 | 15 | 110 | 105 | 9 | 30/30/30 | true |
| 1024 | 45 | 15 | 110 | 105 | 9 | 30/30/30 | true |
| 861 | 45 | 15 | 110 | 105 | 9 | 30/30/30 | true |
| ≤860 | flow | flow | flow | flow | flow | 30 | headerFlowPass=true |

LEFT-01 回归（1024；与 H05 声称一致）：

| 目标 | 期望 Left/Top | JSON Left/Top | 抽核 |
|---|---:|---:|---|
| label-module | 108 / 6 | 108 / 6 | 合 |
| ctrl-module | 183 / 6 | 183 / 6 | 合 |
| label-user | 385 / 6 | 385 / 6 | 合 |
| ctrl-user | 431 / 6 | 431 / 6 | 合 |
| label-begin | 108 / 37 | 108 / 37 | 合 |
| ctrl-begin | 183 / 37 | 183 / 37 | 合 |
| label-sep | 385 / 37 | 385 / 37 | 合 |
| ctrl-end | 431 / 37 | 431 / 37 | 合 |

| 视口 | filtersH | 四控件宽 | left01Pass |
|---:|---:|---|---|
| 1440/1024/861 | 81±2 | 196/196/196/196 | true |
| ≤860 | 响应式 flow | 不要求 196/81 | left01Pass=true（flow） |

总闸：`desktopPass=true`、`responsivePass=true`、`errors=[]`。

## 越界检查（硬）

- [x] 未改业务实现（抽验侧）
- [x] 未 push
- [x] 未 force / reset / hard
- [x] 未签 90% / 未关任务 / 未改看板口径
- [x] 未发现全局 MasterPanelHeader / 整页壳 absolute / 表列 Width / 新 search API
- [x] LEFT-01 filters token/几何未回归
- [x] 无关脏树未夹带（tip 内容干净；本抽验仅 stage 本 md）

## 残留备注（不挡建议通过）

- 本抽验为 **静态 diff + H05 相对 header/filters JSON 抽核**，未重跑 Playwright / 真宿主；符合 H14「H06 测量+禁区抽查」薄口径。
- 检索/导出采用 **right=105/9**（非 left=813/909）；与 H14「DFM Anchors akRight 等价」一致，交付已明示。
- `component-build.cjs` 内 task 字符串仍写 LEFT-01（探针脚手架残留）；目录内无 `component-build-results.json`——**不构成本片布局抽验否决**。
- 不以局部测点代替 SCORE；不签 90%。

## 建议

**建议通过** UI90-04B-LEFT-02（tip `ef46286`）。是否关单 / 是否另立 LEFT-03 或 SCORE 由 H01 定；本岗不定口径、不关任务、不签 90%。
