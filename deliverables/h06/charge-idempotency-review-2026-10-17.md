# 收费幂等回归独立复核

日期：2026-10-17（Asia/Singapore）
复核人：H06
范围：收费创建、现金支付、退费的持久化幂等 TTL 与服务重启重放

## 结论

代码层通过，整体暂不准出。`Service.Now` 仅用于业务单据时间；收费幂等记录的有效期统一使用服务端 UTC 墙上时钟计算 24 小时。内存存储与 MSSQL 存储的过期判断口径一致，未过期的同键不同请求仍返回幂等冲突。

## 已通过证据

```text
go test ./internal/app -run 'TestCharge(PayAndRefundIdempotencySurvivesServiceRestart|CreateChargeIdempotencySurvivesServiceRestart)' -count=1
go test ./internal/store/memory -run 'TestCharge.*Idempotency' -count=1
go test ./internal/app -run 'Test.*(Charge|Idempotency).*' -count=1
go test ./internal/store/memory -run 'Test.*Idempotency.*' -count=1
go test ./... -count=1
go vet ./internal/app ./internal/store/memory ./internal/store/mssql
npm run check
npm run build
```

以上命令在本轮工作区通过；前端构建保留既有单个 JS chunk 超过 500 KB 的提示。`go test -race` 未执行，当前 Go 环境未启用 CGO，race 运行条件不满足。

## 未完成证据

1. SQL Server 集成测试要求 `BHIS_DASHBOARD_TEST_DSN` 指向 `BHIS_PHASE2_TEST_20261015`。当前代码 A 的 `MSSQLSERVER` 服务为停止状态，当前账号无法启动服务；使用 `sqlserver://192.168.1.138:53151?database=BHIS_PHASE2_TEST_20261015` 连接时返回受信任域登录失败。
2. 当前通过的重启回归复用了同一个 memory store，仅重建 `Service`，不能替代真实 SQL 进程重启证据。
3. 尚未形成 24 小时前后边界的独立 SQL 断言。过期记录删除后会进入正常业务状态校验，支付可能因收费单已支付而返回状态冲突。

## 准出条件

在隔离库完成 SQL Server 连接和进程重启后，复跑收费创建/支付/退费三类同键重放、同键改请求、过期边界及清理检查，并记录目标库名、主机 UTC 时间、提交 SHA 和退出码。完成前只能标记“代码层通过，SQL 验收待补”。
