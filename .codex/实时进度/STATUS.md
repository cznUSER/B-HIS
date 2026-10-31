# B-HIS 运行状态（H01 启动读）

更新说明：权威任务列表以同目录 `任务看板.json` 为准。

- 看板：`E:\B-HIS\.codex\实时进度\任务看板.json`
- 岗位：`E:\B-HIS\.codex\agents\`
- 规则：`E:\B-HIS\AGENTS.md`

## 当前总状态（2026-10-31 12:10 Asia/Shanghai）

- 本机：`http://127.0.0.1:8888/login`（health ok `0.3.11-draft`）；DSN 形 `192.168.1.138:53151?database=BHIS`
- M0：完成 `deliverables\h01\m0-truth-align-2026-10-31.md`
- H14 拆法已确认；不启克隆
- 准出：H02 P0 字段冻结 + REG 产品 R0；H03 REG 正式准出 + P0 形态 A 零新 API（已采纳）
- 022 窗口：已完成（隔离 BHIS）→ `deliverables\h01\reg-r2-022-window-2026-10-31.md`（跳过 up；backfill updated=0 pending=0；错库拒绝已证）
- Git：按任务分批提交推送已授权；禁 reset/混交
- 不做：公网；合同外医保/住院等


