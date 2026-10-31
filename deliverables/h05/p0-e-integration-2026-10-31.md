# P0-E-INTEGRATION-01 / Ea · 门诊共享基础集成复验

- **任务编号**: P0-E-INTEGRATION-01（Ea = H05 主跑）
- **岗位**: H05
- **完成时间**: 2026-10-31 15:32（Asia/Shanghai，UTC+8）
- **基线 tip（开跑前）**: `4e4b85643c122f14d28eb007be8510b9a864bdeb`（= origin tip）
- **分支**: `codex/register-slot-card-20261028`
- **机器**: 华为笔记本 `0b530576-3220-453b-8d1f-d68465218648` / `E:\B-HIS`
- **性质**: 优先只测；发现 A2 浏览器 harness 因 P0-C StatusChain 挂载后打只读列表而失败，做了**最小 harness 胶水**；另增轻量写链 smoke（证据用，无业务功能）。

## 变更文件（本片）

| 路径 | 说明 |
| --- | --- |
| `web/scripts/check-outpatient-context-browser.mjs` | 最小胶水：stub StatusChain 会打的 GET `/patients`（集合）/ `/orders` / `/charges` / `/dispenses` / `/registers/{id}/diagnoses` 空列表，避免 Unexpected endpoint |
| `web/scripts/check-outpatient-write-chain-smoke.mjs` | 新建：mock 写链 register→diagnose→order→charge/pay→dispense，断言同键不双单与单号字段 |
| `deliverables/h05/p0-e-integration-2026-10-31.md` | 本交付 |
| `deliverables/h05/p0-e-integration-2026-10-31/*.log` | 各 harness 原始日志 |

未改：Frame / Queue / StatusChain 业务；Go / OpenAPI；各业务 View；反馈原语；UI90 全局 CSS。无关 dirty 未混交。

## 自动化结果表（终跑）

| ID | 命令 | exit | 摘要 |
| --- | --- | --- | --- |
| A1 | `node web/scripts/check-outpatient-context-p0.mjs` | **0** | `P0_A_CONTEXT_CHECK_OK` |
| A2 | `node web/scripts/check-outpatient-context-browser.mjs`（`BHIS_BROWSER_EXECUTABLE`=本机 Chrome） | **0** | `P0_A_CONTEXT_BROWSER_OK` sixModules=6 writes=0 cancelPreservesDraft/confirmClearsDraft/refreshRefetches=true screenshotSha256=`f2ff16c2d3325d7829bed5014e6ef1e4ce017ee7f43f1c86e73fcfde434bb5e6` |
| B1 | `node web/scripts/check-outpatient-queue-p0.mjs` | **0** | `P0_B_QUEUE_CHECK_OK` |
| C1 | `node web/scripts/check-outpatient-status-chain.mjs` | **0** | `P0_C_STATUS_CHECK_OK` |
| D1 | `node web/scripts/check-register-submit-feedback.mjs` | **0** | `P0_D1_REGISTER_FEEDBACK_OK` |
| D2 | `node web/scripts/check-diagnosis-submit-feedback.mjs` | **0** | `P0_D2_DIAGNOSIS_FEEDBACK_OK` |
| D3 | `node web/scripts/check-order-submit-feedback.mjs` | **0** | `P0_D3_ORDER_FEEDBACK_OK` |
| D4 | `node web/scripts/check-charge-submit-feedback.mjs` | **0** | `P0_D4_CHARGE_FEEDBACK_OK` |
| D5 | `node web/scripts/check-dispense-submit-feedback.mjs` | **0** | `P0_D5_DISPENSE_FEEDBACK_OK` |
| SMOKE | `node web/scripts/check-outpatient-write-chain-smoke.mjs` | **0** | `P0_E_WRITE_CHAIN_SMOKE_OK` 同键不双单；registerNo/diagnoseId/orderNo/chargeId·squareNo/dispenseNo 均有值 |

日志目录：`deliverables/h05/p0-e-integration-2026-10-31/`（`final-*.log`、`harness-results-final.json`）。

## M1 退出四条证据索引

1. **六入口患者上下文一致**  
   - 命令：A2 `check-outpatient-context-browser.mjs` exit 0  
   - 证据：`sixModules:6`（register/diagnosis/order/charge/dispense/refund）；切患取消保留草稿、确认清空；刷新仅 Id 再拉；`screenshotSha256=f2ff16c2…bb5e6`（harness 内截图像素哈希）  
   - 单元补充：A1 `P0_A_CONTEXT_CHECK_OK`

2. **队列可选患者**  
   - 命令：B1 `check-outpatient-queue-p0.mjs` exit 0  
   - 证据：`selectQueueRegister` 写入同一 outpatientContext；403 ≠ 空列表；DIAGNOSED 锚定待开单 Tab

3. **状态链可解释**  
   - 命令：C1 `check-outpatient-status-chain.mjs` exit 0  
   - 证据：正常/部分/阻塞/作废文案路径；分域 403 与 empty 区分；刷新聚合按 registerId 作用域

4. **高风险动作幂等/失败可恢复**  
   - 命令：D1–D5 五套 `check-*-submit-feedback.mjs` 均 exit 0  
   - 串联：SMOKE `check-outpatient-write-chain-smoke.mjs` exit 0（挂号→诊断→开单→收费→发药，同键重放不双单）  
   - 降级说明：既有 `check-usability-mock-chain.mjs` 仍期望发药同键抛 `IDEMPOTENT_REJECT`，与 D5「同键同结果」产品语义冲突，**本片未改该旧脚本**；以新建 P0-E smoke + D1–D5 为准。无额外人工截图（browser harness 已产截图哈希；写链为 mock node）。

## A2 首轮失败与胶水

- 首跑 A2 exit 1：`Unexpected endpoint /api/v1/orders`（随后 `/api/v1/patients`）  
- 根因：P0-C `OutpatientStatusChain` 已挂入 Frame，选患后只读聚合 `orders/charges/dispenses/diagnoses`；合成 patientId（p1/p2）非 UUID 时 `listDispenses` 还会 fan-out 到患者搜索  
- 修复：仅扩 browser harness route stub 空列表；**未改** Frame/StatusChain 业务

## 禁区自检

- [x] 无新业务功能 / 无 Go / OpenAPI / 迁移  
- [x] 未重设计 Frame / Queue / StatusChain / 平行反馈栈  
- [x] 未 git reset / force  
- [x] 无关 dirty 未纳入本片 commit

## Origin

- 开跑：本机 tip = origin `4e4b85643c122f14d28eb007be8510b9a864bdeb`（ahead 0）
- 本片 commit：`edd10e178977e2350c0567ba27b848c23aa4e502`（短 `edd10e1`）`test(h05): P0-E-INTEGRATION-01 Ea harness reverify`
- `git push origin HEAD` → **ok**（`4e4b856..edd10e1` → `codex/register-slot-card-20261028`；ahead 0）
## 残余 / 给 H06·H01

- 旧 `check-usability-mock-chain.mjs` 与 D5 幂等语义不一致（非本片阻塞；可另立清理）  
- H06 独立复核另派；H18 文档门禁另派  
- 形态 A：本片无新 API