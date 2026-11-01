# UI90-04B-LEFT-03 抽验（H06）

## 结论：**建议通过**

- 抽验岗：H06（只建议；不定需求口径、不关任务、不签 90%）
- 目标 tip（全）：`f3558a8c9fe3cf82f8bfbd7238967c6510fc0463`（短 `f3558a8`）
- 开工 tip：`2304058`
- 基线：`8695f2a`（全 `8695f2a3a6df1965f620c14f126aad381b45a6aa`）
- 抽验时 HEAD：`f3558a8`（与目标 tip 一致）
- 分支：`codex/register-slot-card-20261028`
- origin：抽验时相对 origin **ahead 2**（443 债，H01 在推；本抽验 **未 push**）
- 是否改业务：**否**（本抽验仅新增本 md）
- push：**否**
- 时间：2026-11-01（Asia/Shanghai）

## 对照范围

| 项 | 来源 |
|---|---|
| H01 派抽验 | UI90-04B-LEFT-03；壳宽 1008±2；结果区 left≈0/宽随壳；LEFT-01/02 守恒 |
| H05 交付 | `deliverables/h05/ui90-04b-left-03-2026-11-01.md` |
| 测点/探针 | `deliverables/h05/ui90-04b-left-03/after-results.json`、`geometry-probe.cjs` |
| 实现 diff | `git show f3558a8 -- web/src/views/access/UserOperationsLogPanel.vue` |
| 文件清单 | `git show --name-only f3558a8` → 仅 Vue + H05 交付（13 files；无 AppShell/MasterPanelHeader） |

## 对照项 × 证据

### 1) 桌面 panelW=1008±2；resultsW≈clientW；resultsLeft≈0±2；stackH≈126±4 — **通过**

| 验收点 | 证据 |
|---|---|
| 具名壳宽 token | tip scoped：`--uol-shell-w: 1008px`；`box-sizing:border-box`；`@media (min-width:861px)` `width:var(--uol-shell-w); max-width:100%`；≤860 `width:auto; max-width:100%` |
| 结果区填壳 | 同媒体查询：`.user-operations-results` / `.result-empty` / `.shell-status` → `position:relative; left:0; width:100%; max-width:none` |
| 测边口径 | H05 / 探针：`getBoundingClientRect().width`（border-box）=1008±2；`resultsLeft` 相对 panel；空态测 `.result-empty` |
| 桌面壳宽/结果区 | `after-results.json`：见下表；`shellPass=true`（闸：viewport≥1008） |
| stackH 可选 | 1440/1280/1024/1008：headerH45+filtersH81=**126**（目标 126±4） |

| 视口 | panelW | clientW | resultsW | resultsLeft | stackH | shellPass |
|---:|---:|---:|---:|---:|---:|---|
| 1440 | **1008** | 1006 | **1006** | **1** | **126** | true |
| 1280 | 1008 | 1006 | 1006 | 1 | 126 | true |
| 1024 | 1008 | 1006 | 1006 | 1 | 126 | true |
| 1008 | 1008 | 1006 | 1006 | 1 | 126 | true |
| 861 | 861（max-width:100% 收缩；不锁 1008） | 859 | 859 | 1 | 126 | n/a（≥1008 闸） |
| ≤860 | flow / auto | — | — | 1 | flow | flow 过 |

容差核对：panelW 相对 1008 偏差 **0**（≤2）；resultsLeft=**1**（≤2）；resultsW=clientW；stackH=126（≤4）。

### 2) LEFT-02 header 守恒 — **通过**

相对 header（桌面 >860；容差 ±2；实测精确相等）：

| 视口 | returnL | titleL | searchR | exportR | headerH | btnH | headerPass |
|---:|---:|---:|---:|---:|---:|---|---|
| 1440 | **15** | **110** | **105** | **9** | **45** | 30/30/30 | true |
| 1280 | 15 | 110 | 105 | 9 | 45 | ≈30 | true |
| 1024 | 15 | 110 | 105 | 9 | 45 | 30/30/30 | true |
| 1008 | 15 | 110 | 105 | 9 | 45 | ≈30 | true |
| 861 | 15 | 110 | 105 | 9 | 45 | 30/30/30 | true |

源码 token 仍在：`--uol-hdr-left-return/title/search/export` + right 105/9；本 tip **仅加壳宽 CSS**，未改 header 定位块。

### 3) LEFT-01 filters 守恒 — **通过**

相对 `.user-operations-filters`（桌面；JSON 1024/1440 与 H05 声称一致）：

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
| ≥861 | **81** | **196/196/196/196** | true |
| ≤860 | 响应式 flow | 不要求 196/81 | left01Pass=true（flow） |

源码 token 仍在：`--uol-left-*` 108/183/385/431；`--uol-filters-h:81px`；`--uol-ctrl-w:196px`。

### 4) 未改 AppShell/MasterPanelHeader；无表列 Width；无新 search API；未签 90%；未 SCORE — **通过**

| 禁项 | 证据 |
|---|---|
| 未改 AppShell / MasterPanelHeader | `git diff-tree --name-status -r f3558a8` **无** 二者；仅 `UserOperationsLogPanel.vue` + H05 交付 |
| 无表列 Width 矩阵 | tip Vue diff 仅 panel 壳宽 + 结果区 left/width；未触结果表列 |
| 无新 search API / 零 Go/022/ICD/Dict | commit 无 Go/OpenAPI/迁移/Dict；script/template 相对基线前 212 行 **逐字节相等**（仅 `<style scoped>` 变更） |
| 不签 90% / 未 SCORE | commit message：`no 90% sign-off`；H05 md 明示「本片不签…；未 SCORE」；JSON mode 写明 not SCORE；本抽验亦不签 |
| FIX-03 / 业务语义 | 选人/清除/既有 GET 路径未改（script 未动） |

### 5) 总闸与指纹

- JSON：`desktopPass=true`、`shellPass=true`、`responsivePass=true`、`errors=[]`
- `sourceSha256` = `d2be7f0061a1ab455e4d31f3f694846f1edbf4d5f50c2641bf81ddfd0eb8856a` 与 tip 工作区 Vue SHA-256 **一致**

## 越界检查（硬）

- [x] 未改业务实现（抽验侧）
- [x] 未 push
- [x] 未 force / reset / hard
- [x] 未签 90% / 未关任务 / 未改看板口径
- [x] 未发现 AppShell / 全局 MasterPanelHeader / 表列 Width / 新 search API / Go/022/ICD/Dict
- [x] LEFT-01 / LEFT-02 几何与 token 未回归
- [x] 无关脏树未夹带（tip 内容干净；本抽验仅 stage 本 md）

## 残留备注（不挡建议通过）

- 本抽验为 **静态 diff + H05 探针 JSON 抽核**，未重跑 Playwright / 真宿主；符合 H06「测量+禁区抽查」薄口径。
- `resultsLeft=1` 来自 panel 1px 边框（相对 border-box）；H05 已声明，落在 ±2。
- vp=861 因 `max-width:100%` 缩至 861；壳宽硬闸按探针仅要求 ≥1008，与 H05「不要求锁 1008」一致。
- 不以局部测点代替 SCORE；不签 90%。

## 建议

**建议通过** UI90-04B-LEFT-03（tip `f3558a8`）。是否关单 / 是否另立 LEFT-03b/04 或 SCORE 由 H01 定；本岗不定口径、不关任务、不签 90%。
