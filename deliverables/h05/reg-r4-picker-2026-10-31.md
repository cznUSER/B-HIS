# REG-R4-PICKER-01 交付

- 任务编号：REG-R4-PICKER-01
- 执行机：华为笔记本-代码（machineId `0b530576-3220-453b-8d1f-d68465218648`）
- 时间：2026-10-31 12:30 Asia/Shanghai
- 编码：本机 Codex CLI（中断后由 H05 执行器补齐竞态脚本契约字段与本交付）

## 变更文件列表

- `web/src/components/RegisterPatientPicker.vue`（Esc 关闭；既有 `requestSeq` 迟到响应丢弃保持；`close()` 递增 seq）
- `web/scripts/check-register-patient-picker-race.mjs`（新建：390 视口 + Esc + 迟到响应不覆盖）
- `deliverables/h05/reg-r4-picker-2026-10-31.md`（本文件）

未改：`web/scripts/check-register-patient-picker.mjs`、全局 CSS、OpenAPI、Go、migrations、其它 Vue/View。

## 竞态策略说明

1. `loadPage` 每次递增本地 `requestSeq`，仅当 `seq === requestSeq` 时写入 `items/total/offset/error/loading`。
2. `close()`（含 Esc / 遮罩 / 取消）再递增 `requestSeq`，使进行中的迟到响应全部失效。
3. 重新打开弹窗由 `watch(open)` 再次 `loadPage`，只展示最新一次请求结果。
4. Esc：`window` keydown 监听，仅在 `open===true` 且 `Escape` 时调用 `close()`，不改动当前已选患者（本组件只 `emit('close')`）。

## 检查命令与结果

环境：`BHIS_BROWSER_EXECUTABLE=C:\Program Files\Google\Chrome\Application\chrome.exe`（本机 ms-playwright chromium 未安装）。

```text
cd E:\B-HIS\web
node .\scripts\check-register-patient-picker-race.mjs
# exit 0
# register picker race check passed at 390px after 2 requests
```

```text
cd E:\B-HIS\web
node .\scripts\check-register-patient-picker.mjs
# exit 1
# locator.waitFor Timeout：register-today-table 内「患者patient-beta」resolved to hidden
# 失败点在打开 picker 之前；RegisterView 将今日工作表包在 <details class="register-today-fold"> 默认折叠，属本任务禁止修改的 View。
```

## 未完成项 / 风险

- 原 `check-register-patient-picker.mjs` 全量用例未能跑通，被边界外 `RegisterView.vue` 的 `<details>` 折叠挡住；取消/换人断言未在本轮复验。
- 专项竞态脚本已 exit 0，覆盖：Esc 关闭、重开后 fresh 渲染、迟到 stale 不覆盖。
- 本步未 git commit/push（由 H05 编排器切片提交）。

## Codex 备注

- 首次 `codex exec` 约 634s 后异常退出（exit -1），已写出 Vue Esc 与 race 脚本初稿，但未写交付 md、未打印 `DONE_REG_R4`。
- race 初稿 mock 体缺少契约字段（`patientNo/sex/idType/...`），`assertRegistrationPatientList` 拒绝导致失败；执行器已在允许文件内补齐后复跑通过。