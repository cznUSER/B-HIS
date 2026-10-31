# UI90-04B-LEFT-01 抽验（H06）

## 结论：**建议通过**

- 抽验岗：H06（只建议；不定需求口径、不关任务、不签 90%）
- 目标 tip（全）：`de00f3e0a7a5de084ef49318eb3bb87b09ab5cca`（短 `de00f3e`）
- 基线：`159a309`（全 `159a3096caf2fd96ddbdef57ca1a39f023433252`）
- 看板开工 tip：`2fb9c4a`
- 分支：`codex/register-slot-card-20261028`
- origin：抽验时相对 origin **0/0**；本抽验 **未 push**
- 是否改业务：**否**（本抽验仅新增本 md）
- push：**否**
- 时间：2026-11-01（Asia/Shanghai）

## 对照范围

| 项 | 来源 |
|---|---|
| H14 开工卡 | `deliverables/h14/ui90-04b-absolute-left-01-start-card-2026-11-01.md` |
| H05 交付 | `deliverables/h05/ui90-04b-left-01-2026-11-01.md` |
| 测点/探针 | `deliverables/h05/ui90-04b-left-01/after-results.json`、`behavior-results.json`、`component-build-results.json`、`scope-results.json` |
| 实现 diff | `git diff 159a309..de00f3e -- web/src/views/access/UserOperationsLogPanel.vue` |
| 文件清单 | `git show --stat de00f3e` → 仅 Vue + H05 交付目录（23 files） |

## 对照项 × 证据

### 1) H14：相对 filters Left±2 / R2 Top±2；196±2；81±2；FIX-03 保留 — **通过**

| 验收点 | 证据 |
|---|---|
| 具名 `--uol-left-*` / Top token | tip scoped：`--uol-left-label-module:108px`、`--uol-left-ctrl-module:183px`、`--uol-left-label-user:385px`、`--uol-left-ctrl-user:431px`、begin/end 同行 Left、`--uol-top-r1:6px`、`--uol-top-r2:37px`；filters `position:relative` |
| 桌面相对 filters 绝对位 | `@media (min-width:861px)`：命名 `uol-abs-*` `position:absolute` + `left/top: var(--uol-*)`；field/date-range `display:contents`；窄屏 ≤860 回退既有 flow/grid |
| 主控件宽 196±2 / filters 高 81±2 | `after-results.json` 1440/1024/861：四主控宽均为 **196**；`filtersHeight=81`；`desktopPass=true` |
| FIX-03 保留 | tip 相对基线 **script 逐字节相等**；template 仅加 `uol-abs-*` 定位 class（剥离后与基线一致）。仍含 `readonly`、省略号/`aria-label="选择用户"`、`fetchUsers`、`filters.userId`、`openUserPicker`/`clearUser`、`role="dialog"`。`behavior-results.json`：`readonly/ellipsisSelect/clear=true`；选人后 audit 含 `userId`，清除后仅去掉 `userId` |

### 2) 禁全页/header absolute；禁表列 Width；禁新 search API — **通过**

| 禁项 | 证据 |
|---|---|
| 无全页 / header absolute | tip 仅两处 `position:absolute`：(1) 条件区内命名 `uol-abs-*`（本片准许，含 legend `left:3/top:13`，H14 允许守恒动）；(2) 既有 `.user-operations-user-picker`（FIX-03 薄弹层）。无 AppShell / header chrome / 整页 1008 壳 absolute |
| 无表列 Width 矩阵 | Vue 无 `colWidth`/`column-width`；commit 未触结果表列 |
| 无新 search API | commit 无 Go/OpenAPI/迁移；behavior 请求仅既有 `GET /api/v1/access/modules`、`/api/v1/audit/user-operations`、`/api/v1/users` |

### 3) 不签 90%；未 SCORE；零 Go/022/ICD/Dict — **通过**

| 项 | 证据 |
|---|---|
| 不签 90% / 未 SCORE | commit message：`no 90% sign-off`；H05 md 明示「本片不签 UI90-04C / UI90-04B 90%；未 SCORE」；本抽验亦不签 |
| 文件边界 | `de00f3e` 仅：`UserOperationsLogPanel.vue` + `deliverables/h05/ui90-04b-left-01*` |
| 零越界模块 | 无 Go / 022 / ICD / Dict / 全局 design-system / access-control / master-design 改动 |
| 业务语义 | script 不变；筛选/分页/导出/选人语义未漂；`sourceSha256` 与 tip Vue SHA-256 一致：`532fd4fa8e680e0999d22e596657da0f1b88e6ffe3d53ae39c90f8d6d264ad9c` |

## 测点抽核（相对 `.user-operations-filters`；对照 H05 声称 / JSON）

H05 报 1024 与 `after-results.json` viewport=1024 `targets[]` **逐项一致**（Left/Top 精确相等，容差 ±0 ≪ ±2）：

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

桌面宽高守恒抽核：

| 视口 | filters 高 | module/user/begin/end 宽 | absolutePass | 抽核 |
|---:|---:|---|---|---|
| 1440 | 81 | 196/196/196/196 | true | 合 81±2 / 196±2；Left/Top 全合 |
| 1024 | 81 | 196/196/196/196 | true | 同上 |
| 861 | 81 | 196/196/196/196 | true | 同上（Top 浮点噪声 ≪1） |
| ≤860 | 自适应 | 弹性 | flowPass=true | 与 H14「窄屏不要求绝对 Left」一致；`responsivePass=true` |

`component-build-results.json`：隔离组件生产构建 `pass=true`。H05 如实披露全量 `build:demo` 因范围外 `RegisterView`/`registerApi` 导出失败——**不构成本片布局抽验否决**。

## 越界检查（硬）

- [x] 未改业务实现（抽验侧）
- [x] 未 push
- [x] 未 force / reset / hard
- [x] 未签 90% / 未关任务 / 未改看板口径
- [x] 未发现全页/header absolute 冒充本刀 / 表列 Width 矩阵 / 新 search API
- [x] 未泄漏 04B-LEFT-02（header/整页壳）为本片必做
- [x] FIX-03 行为标记与 behavior JSON 一致保留
- [x] 无关脏树未夹带（tip 内容干净；本抽验仅 stage 本 md）

## 残留备注（不挡建议通过）

- 本抽验为 **静态 diff + H05 相对面板测点/行为 JSON 抽核**，未重跑 Playwright / 真宿主；符合 H14「H06 测量+禁区抽查」薄口径。
- 全量应用构建失败属范围外基线债，交编排器；不以局部测点代替 SCORE。
- legend 桌面 absolute `3/13` 属 H14 允许的守恒动，非 header chrome。

## 建议

**建议通过** UI90-04B-LEFT-01（tip `de00f3e`）。是否关单 / 是否另立 04B-LEFT-02 或 SCORE 由 H01 定；本岗不定口径、不关任务、不签 90%。