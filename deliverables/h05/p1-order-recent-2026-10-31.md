# P1-ORDER-RECENT-01 交付说明（H05）

- 编号：`P1-ORDER-RECENT-01`
- 全名：开单页 · 会话内存「最近使用」itemNo 芯片一键入行（形态 A）
- 日期：2026-10-31（Asia/Shanghai）
- 基线 tip：`e8073b7`
- 分支：`codex/register-slot-card-20261028`

## 范围

- 新建 `web/src/workbench/recentOrderItems.js`：JS 内存 MRU（上限 8）、会话 store、目录 resolve；**零** localStorage/sessionStorage/网络
- `OrderView.vue` 最小接线：常用条下方并列「最近使用」条/空态/禁用矩阵；入行复用 `applyCommonItemToLines`；Finder/COMMON/RECENT 入行成功后 `rememberSessionItemNo`
- 新建 `web/scripts/check-order-recent.mjs`
- `check-order-item-find.mjs` 仅改「最近使用尚未存在」守卫为共存断言（FIND 过滤契约未改）

## 验收

| harness | exit |
|---|---|
| check-order-recent.mjs | 0 |
| check-order-common.mjs | 0 |
| check-order-item-find.mjs | 0 |
| check-order-print-status.mjs | 0 |
| check-order-submit-feedback.mjs | 0 |

## 禁项自检

- 未改 commonOrderItems 常量语义；未改 PRINT/feedback/诊断四刀业务文件
- 未引入 Go/新 API/022/Dict/ICD/DIAG-AUTOSAVE/共享壳「最近记录」
- 未使用 localStorage/sessionStorage；刷新清空诚实空态
- 无关 dirty 未 stage

## 残余风险

- 会话内存不跨标签/刷新；产品若要持久化须另立
- commit 成功后批量 push 本单 itemNo 未做（卡允许估时紧时后置）
- 工作区仍有大量无关 dirty；origin 推送依赖本机 gitlab 可达