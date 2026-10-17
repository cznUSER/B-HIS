# 收费幂等回归独立复核

日期：2026-10-17（Asia/Shanghai）
复核人：H06
范围：收费创建、现金支付、退费的持久化幂等 TTL 与服务重启重放

## 结论

收费持久化幂等集成通过；TTL 实现与同键冲突检查通过独立复核。`Service.Now` 仅用于业务单据时间；收费幂等记录的有效期由应用服务主机 UTC 墙上时钟计算 24 小时。内存存储与 MSSQL 存储均以 UTC 墙上时钟判断过期，未过期的同键不同请求仍返回幂等冲突。

收费创建、支付、退费的 SQL Store 集成现已在隔离库通过。真实服务进程重启后的端到端重放和 24 小时前后边界仍未实测，因此结论限定为“SQL 持久化 Store 集成通过”，不代表跨进程重启及过期边界已验收。

## 已通过证据

```text
go test ./internal/app -run 'TestCharge(PayAndRefundIdempotencySurvivesServiceRestart|CreateChargeIdempotencySurvivesServiceRestart)' -count=1
go test ./internal/store/memory -run 'TestCharge.*Idempotency' -count=1
go test ./internal/app -run 'Test.*(Charge|Idempotency).*' -count=1
go test ./internal/store/memory -run 'Test.*Idempotency.*' -count=1
go test ./internal/store/mssql -count=1 -v
go test ./... -count=1
go vet ./internal/app ./internal/store/memory ./internal/store/mssql
npm run check
npm run build
```

应用与 memory 相关命令、`go test ./... -count=1`、`go vet ./internal/app ./internal/store/memory ./internal/store/mssql`、`npm run check`、`npm run build` 在本轮工作区通过；前端构建保留既有单个 JS chunk 超过 500 KB 的提示。`go test -race` 未执行，当前 Go 环境未启用 CGO，race 运行条件不满足。

最新完整 MSSQL 包复测结果由 H01 提供：命令 `go test ./internal/store/mssql -count=1 -v` 在隔离库 `BHIS_PHASE2_TEST_20261015` 共 19 项全部通过，包含 charge concurrency、access control、user atomic、bundle 以及原 13 项 SQL 命名用例。迁移 016 的执行由 H01 从源文件提取 21 条幂等列变更语句，在内存拼成隔离专用批次，并加入 `DB_NAME()` 必须等于 `BHIS_PHASE2_TEST_20261015`、`dbo.patient` 必须存在的守卫及事务包装；连接也显式指定该隔离库。该受守卫批次成功执行，未写回仓库。源迁移 `016_patient_registration_detail.up.sql` 未修改，仍含 `USE BHIS`，没有 `DB_NAME()` 守卫，不能直接视为隔离库安全迁移脚本。上述 SQL Store 用例通过不等价于真实服务进程重启或 24 小时过期边界验收。

## 未完成证据

1. SQL Server 持久化集成现已通过；`TestChargeCreateIdempotencySQL` 在同一隔离库上重建 Store 并验证创建、支付、退费重放及请求冲突，但没有停止并重新启动服务进程。因此真实跨进程重启后的端到端证据仍缺。
2. 尚未形成 24 小时前后边界的独立 SQL 断言。过期记录被删除后会进入正常业务状态校验；例如已支付收费单不会因幂等记录过期而再次支付成功。
3. `网站操作手册.md` 原第 6 条曾描述为“按新请求流程处理”，可能被理解为请求会成功；H01 已改为“进入正常业务状态校验，结果由当前收费单状态决定，不保证成功”。`网站数据结构文档.md` 原文已说明过期键删除后进入正常业务状态校验、结果由当前收费单状态决定且不保证成功，内容合格，无需修改。本次 H06 只更新本复核报告，未修改手册。
4. 迁移 016 源文件内的 `USE BHIS` 会将执行上下文切到 `BHIS`；源文件仍不适合直接对隔离库执行。部署手册已明确隔离库必须使用目标专用守卫副本，并提示 down 会删除新增列及其数据，因此执行安全边界已有文档覆盖。三本手册的相关幂等与迁移口径已完成一致性复核。

## 准出条件

收费 SQL Store 集成可标记通过。三本手册相关表述的一致性复核已完成。若要对跨进程耐久性及完整 24 小时语义准出，还需在隔离库停止并重启 API 进程后验证支付/退费重放，并覆盖有效期内、恰好到期及到期后的同键请求，记录目标库名、应用主机 UTC 时间、提交 SHA、退出码和 fixture 清理结果。执行迁移 016 时须遵循部署手册，使用目标专用守卫副本；不得原样执行含 `USE BHIS` 的源文件，down 不作为默认回滚步骤。
