# UI90-04B-LEFT-04 抽验（H06）

## 结论：**建议通过**

- 抽验岗：H06（只建议；不定需求口径、不关任务、不签 90%）
- 目标 tip（全）：`9e6dcc1198ad85441bddf8d7ab99a4db4237228b`（短 `9e6dcc1`）
- 开工 tip：`fd358ea`
- 基线：`3954ad3`（全 `3954ad3fa4a934122651876374d4652925a778c4`）
- 抽验时 HEAD：`7c842cf`（tip-fill docs；目标 tip 为祖先，只读抽核）
- 分支：`codex/register-slot-card-20261028`
- origin：抽验时相对 origin **0/0**；本抽验 **未 push**
- 是否改业务：**否**（本抽验仅新增本 md）
- push：**否**
- 时间：2026-11-01（Asia/Shanghai）

## 对照范围

| 项 | 来源 |
|---|---|
| H01 派抽验 | UI90-04B-LEFT-04；DFM 九列 Width 102/76/123/67/67/324/87/85/75±2；扩展列 auto；禁发明 Left；LEFT-01/02/03 守恒 |
| H05 交付 | `deliverables/h05/ui90-04b-left-04-2026-11-01.md` |
| 测点/探针 | `deliverables/h05/ui90-04b-left-04/after-results.json`、`geometry-probe.cjs` |
| 实现 diff | `git show 9e6dcc1 -- web/src/views/access/UserOperationsLogPanel.vue` |
| 文件清单 | `git show --name-only 9e6dcc1` → 仅 Vue + H05 交付（13 files；无 AppShell / 全局 register-table 翻修） |

## 对照项 × 证据

### 1) 桌面九列宽±2；colsPass；视口≥861 — **通过**

| 验收点 | 证据 |
|---|---|
| 具名 Width token | tip scoped：`--uol-col-w-module/type/time/user/dept/content/ip/mac/version` = **102/76/123/67/67/324/87/85/75** px |
| 布局策略 | ≥861：`table-layout:fixed` + `colgroup`/`th`/`td` 具名 class；≤860：`table-layout:auto`，九列 width 回退 auto，时间列 `min-width:155` |
| 测边口径 | H05 / JSON：`th.getBoundingClientRect().width`（border-box）；`measureEdge` 声明与此一致 |
| 总闸 | `after-results.json`：`colsPass=true`、`desktopPass=true`、`errors=[]` |

| 视口 | module | type | time | user | dept | content | ip | mac | version | colsPass |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| 1440 | **102** | **76** | **123** | **67** | **67** | **324** | **87** | **85** | **75** | true |
| 1280 | 102 | 76 | 123 | 67 | 67 | 324 | 87 | 85 | 75 | true |
| 1024 | 102 | 76 | 123 | 67 | 67 | 324 | 87 | 85 | 75 | true |
| 1008 | 102 | 76 | 123 | 67 | 67 | 324 | 87 | 85 | 75 | true |
| 861 | 102 | 76 | 123 | 67 | 67 | 324 | 87 | 85 | 75 | true |
| ≤860 | auto 回退（不锁 DFM） | | | | | | | | | colsPass=true（flow） |

容差：桌面九列相对目标偏差 **0**（≤2）。

### 2) 扩展列仍在（auto）；未发明 Left；未删列 — **通过**

| 列 | th 文案 | DOM | 实测 thW @1440 | 策略 |
|---|---|---|---:|---|
| 结果 | 结果 | th+td 在 | ≈79.66 | `width:auto; min-width:72px` |
| 请求编号 | 请求编号 | th+td 在 | ≈79.66 | 同上 |
| 错误代码 | 错误代码 | th+td 在 | ≈79.69 | 同上 |

- tip template 保留三扩展列；`uol-col-ext-*` 无 DFM Width token。
- 抽核 tip style：全部 `.uol-col-*` 规则 **0** 条含 `left:`（禁发明列 Left 成立）。
- 结果区 `overflow-x:auto`；`resultsScrollable=true`；页级 overflow 未放开（JSON `overflow` 闸与 H05 声明一致）。

### 3) LEFT-01 / LEFT-02 / LEFT-03 回归 — **通过**

**LEFT-03（壳）** — 桌面 ≥1008：

| 视口 | panelW | resultsLeft | stackH | shellPass |
|---:|---:|---:|---:|---|
| 1440/1280/1024/1008 | **1008** | **1** | **126** | true |
| 861 | 861（max-width:100% 收缩） | 1 | 126 | n/a（≥1008 闸） |

token `--uol-shell-w:1008px` 仍在。

**LEFT-02（header）** — 相对 header（桌面；±2；实测精确）：

| 视口 | returnL | titleL | searchR | exportR | headerH | btnH | headerPass |
|---:|---:|---:|---:|---:|---:|---|---|
| ≥861 | **15** | **110** | **105** | **9** | **45** | **30** | true |

**LEFT-01（filters）** — 相对 `.user-operations-filters`（桌面）：

| 目标 | Left/Top | JSON（≥861） |
|---|---:|---|
| label-module / ctrl-module | 108/6 · 183/6 | 合 |
| label-user / ctrl-user | 385/6 · 431/6 | 合 |
| label-begin / ctrl-begin | 108/37 · 183/37 | 合 |
| label-sep / ctrl-end | 385/37 · 431/37 | 合 |

| 视口 | filtersH | 四控件宽 | left01Pass |
|---:|---:|---|---|
| ≥861 | **81** | **196×4** | true |
| ≤860 | 响应式 flow | 不要求 196/81 | left01Pass=true（flow） |

### 4) 未改 AppShell/register-table；无新 search API；未签 90%；未 SCORE — **通过**

| 禁项 | 证据 |
|---|---|
| 未改 AppShell / 全局 register-table | `git show --name-only 9e6dcc1` **无** `AppShell.vue`；仅本页 `register-table` 加 scoped `user-operations-table` class，未改全局语义 |
| script / API 未漂 | `<script>` 相对 `9e6dcc1~1` **逐字节相等**（4159B）；选人/清除/既有 GET/导出语义未动 |
| 无新 search API / 零 Go/022/ICD/Dict | tip 仅 Vue + H05 交付；无 Go/OpenAPI/迁移/Dict |
| 不签 90% / 未 SCORE | commit message：`no 90% sign-off`；H05 md 明示不签/未 SCORE；JSON mode：`not live host or SCORE`；本抽验亦不签 |

### 5) 总闸与指纹

- JSON：`desktopPass=true`、`colsPass=true`、`shellPass=true`、`responsivePass=true`、`errors=[]`
- tip Vue SHA-256（抽核）：`cd9fd3d248b1f117953120000d123954e6762f5a5528384b8eab802f65e9a7d4`
- JSON `sourceSha256`=`135bf4c7…` 与 tip blob **不一致**（见残留备注；不挡测点对齐）

## 越界检查（硬）

- [x] 未改业务实现（抽验侧）
- [x] 未 push
- [x] 未 force / reset / hard
- [x] 未签 90% / 未关任务 / 未改看板口径
- [x] 未发现 AppShell / 全局 register-table 翻修 / 新 search API / Go/022/ICD/Dict
- [x] 未发明列 Left；扩展列未删
- [x] LEFT-01 / LEFT-02 / LEFT-03 几何与 token 未回归
- [x] 无关脏树未夹带（tip 内容干净；本抽验仅 stage 本 md）

## 残留备注（不挡建议通过）

- 本抽验为 **静态 diff + H05 探针 JSON 抽核**，未重跑 Playwright / 真宿主；符合 H06「测量+禁区抽查」薄口径。
- H05 md §7 仍写 amend tip `5edd5f6`，与 H01/git 目标 tip `9e6dcc1` 不一致（文档漂移；内容与 tip diff 一致）。
- JSON `sourceSha256` 与 tip Vue SHA-256 不一致（疑探针取自中间工作树/夹具）；九列实测与 tip token **精确对齐**，故不挡。
- `resultsLeft=1` 来自 panel 1px 边框；落在 ±2。
- 不以局部测点代替 SCORE；不签 90%。

## 建议

**建议通过** UI90-04B-LEFT-04（tip `9e6dcc1`）。是否关单 / 是否另立 SCORE 或后续刀由 H01 定；本岗不定口径、不关任务、不签 90%。
