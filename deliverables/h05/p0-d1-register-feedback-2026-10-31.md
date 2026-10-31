# P0-D1-REGISTER-FEEDBACK-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`192faf6`（全 `192faf63ad56e6dbf4f97a42218d1f312fe8220c`）
- 岗位：H05 · 本机 Codex + 编排修复；未用 CloudAgent；未 reset/force。

## 实现

| 文件 | 变更 |
| --- | --- |
| `web/src/components/feedback/useSubmitFeedback.js` | **新建** idle/submitting/success/error；成功带 registerId/registerNo；不存 token |
| `web/src/components/feedback/SubmitStatusBanner.vue` | **新建** ConfirmRail 默认槽展示；失败仍进 useErrorDialog |
| `web/src/views/RegisterView.vue` | registerAttempt 指纹+调用方持键；registerWriteBusy 互斥锁定；挂号/报到/退号/完成/收费三键统一 begin/succeed/fail；成功清槽，失败保留选择（仅 APPOINTMENT_CONFLICT/SLOT_FULL 清槽） |
| `web/src/api/registerApi.js` | **仅** `createRegister(..., idempotencyKey?)` 调用方持键；空键校验；Mock/真实共用 |
| `web/scripts/check-register-submit-feedback.mjs` | **新建** 专项验收 |
| `RegisterConfirmRail` / `PatientHistory` | **未改**（槽位注入 banner；按钮 `:disabled="registerWriteBusy"`） |

禁区遵守：未动 Frame / OutpatientQueue* / StatusChain* / Go / OpenAPI / 022 / UI90 全局 CSS。

## 验收

```text
node web/scripts/check-register-submit-feedback.mjs
# → P0_D1_REGISTER_FEEDBACK_OK  exit 0
```

覆盖：
1. 提交中 status=submitting（延迟锁定语义 + canSave/registerWriteBusy 门闸）
2. 失败保留选择（onSave catch 不清槽；冲突码可清）
3. 同指纹同键 Mock 二次 createRegister 返回同一 registerId/registerNo（不双单）
4. 409 APPOINTMENT_CONFLICT 可解释
5. 503 / 网络错误可恢复（fail 后仍可 beginAttempt）
6. 成功文案含 registerNo；挂号确认必测；报到/退号/完成/收费动作矩阵抽测

## Git

- feat commit：短 `0036456` / 全 `003645684e7035e5745bd42d14f462f138e8ffc3`
- tip HEAD：本提交（docs tip 戳；以 `git rev-parse HEAD` 为准）
- 分支：`codex/register-slot-card-20261028`
- `git push origin HEAD`：**失败** — `Failed to connect to github.com port 443`；本机 ahead origin **6**（P0-C tip + P0-D1 feat `0036456` + docs stamps；origin 仍停 `ddab6df`）；记债待网络恢复重推。gitlab 仍可能不通。
- 未混交无关 dirty（mockRegisters 预约冲突等既有脏改未 stage）。
- Codex 会话：`01a1906c-c6f3-7572-9f92-b7d89af3c46a`（Hi Token / gpt-6.1-sol）

`DONE_P0_D1`
