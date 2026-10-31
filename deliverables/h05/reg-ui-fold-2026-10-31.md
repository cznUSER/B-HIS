# REG-UI-FOLD-01 交付记录

- **任务编号：** REG-UI-FOLD-01
- **岗位：** H05
- **完成时间：** 2026-10-31 12:50（Asia/Shanghai，UTC+8）
- **编码方式：** 本机编排 + 验收脚本适配（本机 Chrome/Playwright）
- **另片：** 不并 P0-A-CONTEXT-01（Frame / outpatientContext 另提交）

## 根因

号源卡片版 `RegisterView` 将「今日工作」表放入 `<details class="register-today-fold">`。默认未带 `open` 时，表体处于折叠态，Playwright 对「患者patient-beta」的可见性断言失败。

## 修复（产品默认）

今日工作表 **默认展开**：`RegisterView` 中

`<details class="register-today-fold" data-testid="register-today-panel" open>`

符合「今日工作表默认可见」产品取向；未改全局 CSS / OpenAPI / Go / 022。

## 验收脚本适配（同片）

`web/scripts/check-register-patient-picker.mjs` 对齐当前号源卡片 DOM（非改折叠产品语义）：

- 补齐 `appearance/effective`、`preferences`、`registers/slot-board` mock，避免 site-error 挡道
- 档案区改为 `register-patient-fields`（原 `register-patient-detail-table` 已下线）
- 草稿断言改为确认栏 `pending-register-fee` / `payment-*`；换患后费用与支付偏好重置为产品现状
- 换患确认：兼容 Frame 切患 dialog（`outpatient-switch-confirm`）与原生 `confirm`（`page.on('dialog')` accept）
- 503 重试前先关 `site-error-dialog`

## 验收结果

| 脚本 | 结果 |
| --- | --- |
| `web/scripts/check-register-patient-picker.mjs` | exit 0 |
| `web/scripts/check-register-patient-picker-race.mjs` | exit 0（R4 仍绿） |

## 变更文件（本片）

| 路径 | 说明 |
| --- | --- |
| `web/src/views/RegisterView.vue` | 号源卡片工作台 + 今日表 `details` **默认 `open`** |
| `web/src/components/register/*` | 患者条 / 科室格 / 医生卡 / 号源格 / 确认栏 / 病史（RegisterView 依赖） |
| `web/src/register/slotCardModel.js` | 号源卡片模型 |
| `web/src/views/register-slot-card.css` | 号源卡片样式（非全局换肤） |
| `web/scripts/check-register-patient-picker.mjs` | 全量验收对齐 |
| `deliverables/h05/reg-ui-fold-2026-10-31.md` | 本交付 |

**未纳入本片：** `OutpatientWorkbenchFrame.vue`、`outpatientContext.js`、P0-A 交付与脚本（另片提交）。

## 推送

本机提交后按网络情况推送；若 gitlab/origin/syncproxy 仍不通，提交哈希随汇报给出，推送阻塞另记。
