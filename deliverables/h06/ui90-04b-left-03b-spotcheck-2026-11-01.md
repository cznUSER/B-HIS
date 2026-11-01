# UI90-04B-LEFT-03b 抽验（H06）

## 结论：**建议通过**

- 抽验岗：H06（只建议；不定需求口径、不关任务、不签 90%）
- 目标 tip（全）：`1f1e88e5442d4f10975e61e28c6d0cabab8832bd`（短 `1f1e88e`）
- 基线 tip：`e4af54e`（全 `e4af54e19b6f302eaba01fd4d03172a5885c41e7`）
- 抽验时 HEAD：`1f1e88e`（与目标 tip 一致；只读抽核）
- 分支：`codex/register-slot-card-20261028`
- origin：抽验时相对 origin **ahead 1**（含本 tip；本抽验 **未 push**）
- 是否改业务：**否**（本抽验仅新增本 md）
- push：**否**
- 时间：2026-11-01（Asia/Shanghai）

## 对照范围

| 项 | 来源 |
|---|---|
| H01 派抽验 | UI90-04B-LEFT-03b · 壳高测表；panelH **458±4** / resultsH **332±4**；LEFT-01～04 守恒；禁动 AppShell/shell-design |
| H05 交付 | `deliverables/h05/ui90-04b-left-03b-2026-11-01.md` |
| 测点/探针 | `deliverables/h05/ui90-04b-left-03b/after-results.json`、`geometry-probe.cjs` |
| 实现 diff | `git diff e4af54e..1f1e88e -- web/src/views/access/UserOperationsLogPanel.vue` |
| 文件清单 | `git show --name-only 1f1e88e` → 仅本页 Vue + H05 交付（13 files；**无** AppShell / shell-design） |

## 对照项 × 证据

### 1) 桌面 panelH 458±4；resultsH 332±4 — **通过**

| 验收点 | 证据 |
|---|---|
| 具名高 token | tip scoped：`--uol-shell-h: 458px`；`--uol-results-h: 332px` |
| 布局策略 | ≥861：panel `height:var(--uol-shell-h)` + flex column + `overflow:hidden`；结果区 `height/max-height/flex-basis:var(--uol-results-h)` + `overflow:auto`；≤860：height/auto 解锁 |
| 测边口径 | JSON `measureEdge`：`getBoundingClientRect().height`（border-box） |
| 总闸 | `after-results.json`：`heightPass=true`、`desktopPass=true`、`shellPass=true`、`errors=[]` |

| 视口 | panelW | panelH | resultsH | stackH | stackWithResults | heightPass |
|---:|---:|---:|---:|---:|---:|---|
| 1440 | 1008 | **458** | **332** | 126 | **458** | true |
| 1280 | 1008 | 458 | 332 | 126 | 458 | true |
| 1024 | 1008 | 458 | 332 | 126 | 458 | true |
| 1008 | 1008 | 458 | 332 | 126 | 458 | true |
| 861 | 861（宽收缩） | 458 | 332 | 126 | 458 | true |
| ≤860 | flow | auto（334/451…） | auto | flow | flow | 不要求锁 458（heightPass=true · flow） |

容差：桌面 panelH / resultsH 相对目标偏差 **0**（≤4）。栈 45+81+332=**458**。

### 2) LEFT-01～04 回归 — **通过**

**LEFT-01（filters）** — 相对 `.user-operations-filters`（桌面 ≥861；±2；实测精确）：

| 目标 | Left/Top | JSON |
|---|---:|---|
| label-module / ctrl-module | 108/6 · 183/6 | ✓ |
| label-user / ctrl-user | 385/6 · 431/6 | ✓ |
| label-begin / ctrl-begin | 108/37 · 183/37 | ✓ |
| label-sep / ctrl-end | 385/37 · 431/37 | ✓ |

| 视口 | filtersH | 四控件宽 | left01Pass |
|---:|---:|---|---|
| ≥861 | **81** | **196×4** | true |

**LEFT-02（header）** — 相对 header（桌面；±2；实测精确）：

| 视口 | returnL | titleL | searchR | exportR | headerH | btnH | headerPass |
|---:|---:|---:|---:|---:|---:|---|---|
| ≥861 | **15** | **110** | **105** | **9** | **45** | **30** | true |

**LEFT-03（壳宽）**：

| 视口 | panelW | resultsLeft | resultsW | shellPass |
|---:|---:|---:|---:|---|
| 1440/1280/1024/1008 | **1008** | **1** | 1006 | true |
| 861 | 861（max-width:100% 收缩） | 1 | 859 | true（高闸；宽随视口） |

token `--uol-shell-w:1008px` 仍在。`resultsLeft=1` 落在 ≈0±2（panel 1px 边框）。

**LEFT-04（九列 Width）** — 桌面 th border-box（@1024 抽样；其它 ≥861 同值）：

| module | type | time | user | dept | content | ip | mac | version | colsPass |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| **102** | **76** | **123** | **67** | **67** | **324** | **87** | **85** | **75** | true |

扩展列仍在（auto）：结果 / 请求编号 / 错误代码 · th+td 均 present（@1024 thW≈79.66/79.66/79.69）。未发明列 Left；未删列。

### 3) 变更 scoped 于 UserOperationsLogPanel；未动 AppShell/shell-design — **通过**

| 检查 | 证据 |
|---|---|
| tip 文件清单 | 仅 `UserOperationsLogPanel.vue` + `deliverables/h05/ui90-04b-left-03b*`（13 files） |
| 无全局壳 | `git show --name-only 1f1e88e` **无** `AppShell.vue` / `shell-design.css` / `AccessControlView.vue` / `MasterPanelHeader.vue` |
| script / template 未漂 | 相对基线 `e4af54e`：`<script>` 逐字节相等（4168B）；`<template>` 逐字节相等（848B） |
| 仅 scoped CSS | diff 全为 style：新增 `--uol-shell-h` / `--uol-results-h` 与 ≥861 / ≤860 高度规则（+23 行量级） |
| 工作树脏文件说明 | 工作树另有未提交的 `AppShell.vue` / `shell-design.css` 等脏改；**不属于 tip `1f1e88e`**，本抽验未 stage |

### 4) 零 Go/022/ICD/Dict/新 search API；未签 90%；未 SCORE — **通过**

| 禁项 | 证据 |
|---|---|
| 无 Go/022/ICD/Dict/新 search API | tip 无 Go/OpenAPI/迁移/Dict；script 未改；无新 search 端点 |
| 不签 90% / 未 SCORE | commit message：`no 90% sign-off`；H05 md 明示本片不签 / 未 SCORE；JSON mode：`not live host or SCORE; no 90% sign-off`；本抽验亦不签 |
| FIX-03 选人/清除语义 | 本刀未改 script；探针几何闸未报回归 |

### 5) 总闸与指纹

- JSON：`desktopPass=true`、`heightPass=true`、`shellPass=true`、`colsPass=true`、`responsivePass=true`、`errors=[]`
- tip Vue blob：`58b5797d3ee04f77279b012c2eef232cc65d43ed`（与工作树一致）
- tip Vue SHA-256：`CE43F07CC63903E4223F2B05BF48EBF703A098A5627FC70F172022686987C006`
- JSON `sourceSha256`=`ce43f07c…` 与 tip Vue SHA-256 **一致**

## 越界检查（硬）

- [x] 未改业务实现（抽验侧）
- [x] 未 push
- [x] 未 force / reset / hard
- [x] 未签 90% / 未关任务 / 未改看板口径
- [x] tip 未含 AppShell / shell-design / 新 search API / Go/022/ICD/Dict
- [x] LEFT-01 / LEFT-02 / LEFT-03 / LEFT-04 几何与 token 未回归
- [x] 无关脏树未夹带（本抽验仅 stage 本 md）

## 残留备注（不挡建议通过）

- 本抽验为 **静态 diff + H05 探针 JSON 抽核**，未重跑 Playwright / 真宿主；符合 H06「测量+禁区抽查」薄口径。
- `resultsLeft=1` 来自 panel 1px 边框；落在 ±2（与 LEFT-03/LEFT-04 抽验一致）。
- 视口 861 时 panelW 随 `max-width:100%` 收缩为 861，但 panelH/resultsH 仍锁 458/332；壳宽 1008 闸仍以 ≥1008 视口为准。
- 工作树大量无关 dirty（含 AppShell/Go/Dict 等）**未**进入 tip、**未**被本抽验 stage。
- 不以局部测点代替 SCORE；不签 90%。

## 建议

**建议通过** UI90-04B-LEFT-03b（tip `1f1e88e`）。是否关单 / 是否另立 SCORE 或后续刀（含 03c）由 H01 定；本岗不定口径、不关任务、不签 90%。
