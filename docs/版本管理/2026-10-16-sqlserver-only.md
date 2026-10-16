# B-HIS 数据库运行边界变更

| 项 | 内容 |
|---|---|
| 版本 | `2026.10.16-sqlserver-only` |
| 时间 | 2026-10-16 17:51（Asia/Shanghai） |
| 基线提交 | `73b50bbfb081b457c11de2475207a58fecd8e166` |
| 本次提交 | `38f227b` (`feat: require sqlserver runtime and reserve dameng adapter`) |
| 分支 | `codex/website-usability-2026-10-14` |
| 需求 | 网站与启动器移除内存库运行模式；SQL Server 为唯一启用的运行时数据库；保留达梦切换接口；文件改动纳入 Git 并通知 H01 |

## 变更边界

- `internal/config`：默认驱动改为 `sqlserver`，允许 `sqlserver` 与预留的 `dameng`，拒绝 `memory` 运行配置。
- `internal/store`：运行时工厂删除 memory 分支；SQL Server 通过 `port.StoreSet` 驱动适配器接入；达梦适配器明确失败闭锁，不回退到其他数据库。
- `cmd/host`、`config.example.yaml`、根 README：启动、配置和故障语义改为 SQL Server 必需。
- `web/.env`、`web/src/api/config.js`：产品构建固定真实 HTTP API；仅测试进程可通过 `BHIS_WEB_TEST_MODE=1` 使用隔离契约替身。
- `web/scripts/check-real-only.mjs`、`check-suite.mjs`：增加运行模式守卫并隔离既有契约脚本。
- 测试：验证 memory 运行入口被拒绝、Dameng 预留接口显式失败、嵌入 UI 无 DSN 不启动。

## 文件版本规则

1. 每次源码、配置、脚本或文档改动先记录基线提交和本文件版本号。
2. 改动完成后运行针对性测试，并在本文件追加验证证据或新建下一版本记录。
3. 通过 Git 提交保存可回滚版本；不得覆盖工作树中未归属本次目标的既有改动。
4. 交审前将需求名、改动文件清单、测试结果和当前版本通知 H01。

## 验证证据

- `go test ./internal/config ./internal/store ./internal/store/mssql ./internal/bootstrap ./cmd/host`：通过。
- `npm run check`（`web/`）：通过，真实 API 守卫和既有前端契约检查均通过。
- `go test ./...`：现有 `internal/app/TestChargePayAndRefundIdempotencySurvivesServiceRestart` 仍失败，错误为 `ILLEGAL_STATE_TRANSITION`；该测试未由本次数据库边界改动触及，已保留为待修项。
