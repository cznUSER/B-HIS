# 收费幂等进程重启验收报告

验收卡：`P1-H06-CHARGE-IDEMPOTENCY-RESTART-20261017`

岗位：H06 独立验收

日期：2026-10-17（Asia/Shanghai）

报告时间：2026-10-17T07:58:58Z

结论：**BLOCKED / 未通过建议**。API 启动命令被当前执行环境策略拒绝，无法取得真实服务进程下的重启重放证据；本报告不把源码核查或既有 SQL Store 测试记作本卡运行验收通过。

## 环境与基线

- 工作区提交 SHA：`eaa9c602ea9028427033bc5ed0cd282e0a271531`。
- 允许目标：Code A `192.168.1.138:53151`，数据库 `BHIS_PHASE2_TEST_20261015`。显式 SQL 连接的只读 `SELECT DB_NAME()` 返回 `BHIS_PHASE2_TEST_20261015`；目标端口此前可达。未连接或写入 `BHIS`、`HospDataV9` 或本机 SQL Server。
- 本机 `MSSQLSERVER` 为停止状态。独立 API 端口 `18081` 和 web 端口 `18082` 在复核时均无监听；原有 API 8080 和 Vite 5175/5176 未触碰。
- 依据 H01《P0 基线与分区》2026-10-17 路线 A：本卡仅在隔离库验证真实 API 进程重启、收费 create/pay/refund 重放、改体冲突、TTL 与一次性副作用；上线和生产库不在授权范围。
- 使用 `go build -o %TEMP%\bhis-h06-charge-20261017.exe ./cmd/host` 构建独立临时可执行文件，构建成功。未改源码、迁移、手册、共享配置或 `.env`，未暂存/提交/推送。
- 计划中的运行 DSN 仅拟通过进程环境变量注入，不落盘；执行环境在进程启动前拒绝了命令。无 API PID、无 API 请求、无 fixture、无 SQL 写入，因而无需执行 fixture 清理。端口复核确认无残留本轮监听。

## 启动阻塞证据

在确认端口空闲并构建临时二进制后，API 启动尝试共三次，均在命令创建阶段被策略拦截；没有命令进入 PowerShell 执行：

| 尝试 | 命令摘要（敏感参数省略） | 结果 |
|---|---|---|
| 1 | `Start-Process -FilePath <临时API.exe> -ArgumentList @('-config','config.example.yaml','-no-browser','start')`，运行配置和连接通过环境变量注入，输出重定向到 `%TEMP%` | `exec_command` 返回 `CreateProcess ... Rejected(...): blocked by policy` |
| 2 | 同一 `Start-Process` 启动命令；凭据先从本机 hyt skill 读取到环境变量，未写文件或回显 | `exec_command` 返回 `CreateProcess ... Rejected(...): blocked by policy` |
| 3 | `& $hostExe -config config.example.yaml -no-browser start *> <TEMP日志文件>`，前台等待 exec 会话，配置和连接通过环境变量注入 | `exec_command` 返回 `CreateProcess ... Rejected(...): blocked by policy` |

拒绝消息中包含拟执行命令文本；本报告不复制其中的连接串或凭据。错误原文的脱敏片段为 `CreateProcess ... Rejected(...): blocked by policy`。由于拒绝发生于进程创建阶段，三次均未获得进程号。最后一次端口/时间复核（UTC `2026-10-17T07:57:26Z`）显示 `18081`、`18082` 均无监听。报告时间为 UTC `2026-10-17T07:58:58Z`。执行环境没有为三次拒绝分别提供可用的时间戳，因此不推定具体秒数。

## 本轮实际测试结果

| 场景 | 请求状态 / 返回 ID | 数据库副作用核对 | 结果 |
|---|---|---|---|
| `charge.create` 同键重启重放 | 未发送；无 HTTP 状态或 `chargeId` | 未创建收费单、明细或幂等记录 | BLOCKED |
| `charge.pay` 同键重启重放 | 未发送；无 HTTP 状态或 `paymentId` | 未创建收款记录 | BLOCKED |
| `charge.refund` 同键重启重放 | 未发送；无 HTTP 状态或 `refundPaymentId` / `abatedChargeId` | 未创建退款、作废/后继收费单 | BLOCKED |
| 三个 scope 的同键改体冲突 | 未发送 | 无幂等冲突请求记录可核对 | BLOCKED |
| TTL 仍有效时的真实跨进程重放 | 未发送；本轮没有创建 TTL 记录 | 无记录可核对 | BLOCKED |
| `expires_at` 严格早于当前时刻后的 API 行为 | 未发送；未调整任何记录 | 无记录需清理或验证余数 | BLOCKED |
| `expires_at` 恰等于应用主机当前 UTC 的边界 | 未运行；精确相等边界不宣称已测 | 无记录可核对 | 未覆盖 |

DB_NAME 只读守卫通过是环境准备证据，不是业务写入或 API 集成测试。未执行数据库聚合查询、未发生 fixture 清理；这两项是因为本轮没有任何写入，而非漏做清理。

## 代码口径核查（非运行证据）

- `internal/app/charge.go` 中 `chargeIdempotencyTTL` 为 24 小时，持久记录到期值从应用服务主机 `time.Now().UTC()` 计算；幂等有效期不取业务 `Service.Now`。
- `internal/store/mssql/charge.go` 的 `readChargeIdempotency` 以 `expiresAt.After(time.Now().UTC())` 判定仍有效；`CreateChargeWithIdempotency` 采用同一比较。故当存储的 `expires_at` 与本次检查使用的应用主机 UTC 瞬间相等时，`After` 返回 false，代码会删除记录并进入正常业务状态校验。这是从比较运算推得的代码语义，不是精确到期时刻的运行实测；应用主机与数据库 UTC 时钟偏差也未测。
- SQL Store 的 pay/refund 查找路径可依据持久幂等记录重建 payment 与退款响应，HTTP 契约分别为 create `201`、pay `200`、refund `200`；本轮没有 API 响应或数据库回读证据来确认这些路径在独立进程重启后实际工作。
- H01 已提供的 SQL 集成结果（MSSQL 包 19 项通过）属于 Store/数据库测试，不能替代本卡要求的真实 API 进程停止/重启验证。本轮未重跑该测试包。

## 未覆盖与准出建议

真实进程重启后 create/pay/refund 同键重放、三类改体冲突、24 小时有效期内重放、强制严格过期后的正常状态结果、支付/收费/幂等副作用计数，以及 fixture 精确清理和余数核验，全部未覆盖。精确 `expires_at == 应用主机 UTC 检查时刻` 也未实测，且不能以近似等待替代。

**给 H01 的建议：未通过（BLOCKED，验收证据不足）。** 这是执行环境未允许本轮启动 API 所致，不是本轮复现到产品缺陷。恢复可启动独立 API 的执行条件后，应重新运行本验收卡；只写入已批准的 Code A 隔离库并在报告中记录请求状态、返回 ID、数据库聚合及清理余数。H01 负责后续文档影响评估、审查通道、单文件提交和 GitLab 推送。
