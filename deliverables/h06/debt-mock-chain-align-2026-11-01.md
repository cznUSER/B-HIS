# DEBT-MOCK-CHAIN-ALIGN-01 · 旧 mock-chain 对齐 D5 同键语义（H06）

- 日期：2026-11-01（Asia/Shanghai）
- 基线 tip：`42f2d09` (`42f2d094efe85dc4c017b6e7b7d1533a55c0d570`)
- 分支：`codex/register-slot-card-20261028`（origin ahead 1 债挂；本片 **未 push**）
- 机：华为笔记本-代码 `0b530576-3220-453b-8d1f-d68465218648` / 仓库 `E:\B-HIS`
- H14 卡：`deliverables/h14/debt-mock-chain-align-01-start-card-2026-11-01.md`

## 1) 路径选择（写死）

**对齐**（首选；未走废弃）。

依据：现行 `mockDispenses` 同键同指纹返回先验结果（`prior.result`），指纹不一致抛 `IDEMPOTENCY_KEY_CONFLICT`；D5 `check-dispense-submit-feedback.mjs` 已按此断言。旧 `check-usability-mock-chain.mjs` 仍期望 `IDEMPOTENT_REJECT`，与 D5/mock 冲突。本片只改正 harness 断言，**不回退** mock 同键语义。

## 2) 改动文件

| 文件 | 说明 |
|---|---|
| `web/scripts/check-usability-mock-chain.mjs` | 发药/退药同键同指纹重放改为先验 `dispenseId`/`dispenseNo` 与 `returnId`/`dispenseId`/`returnedQty`；同键异指纹仍期望 `IDEMPOTENCY_KEY_CONFLICT`。工作区既有患者建档字段补全（idType/idNumber/phone/nationality）一并保留于该准写文件。 |
| `deliverables/h06/debt-mock-chain-align-2026-11-01.md` | 本交付 |

**未改**：`web/src/**`、Go、D5 断言、`mockDispenses`、`check-suite.mjs`、`package.json`。

## 3) suite / CI 引用复验

- `web/scripts/check-suite.mjs`：**未引用** `check-usability-mock-chain`
- `web/package.json`：**未引用** `check-usability-mock-chain`
- 默认勿改 suite（本片遵守）

## 4) 自测（cwd `E:\B-HIS`）

| 命令 | exit |
|---|---|
| `node web/scripts/check-usability-mock-chain.mjs` | **0**（memory + sessionStorage 双 PASS） |
| `node web/scripts/check-dispense-submit-feedback.mjs` | **0**（`P0_D5_DISPENSE_FEEDBACK_OK`） |

## 5) 禁区自检

- [x] 未改 Vue/Go/022/ICD/Dict/AUTOSAVE
- [x] 未回退 mock 同键语义
- [x] 未放宽 D5 断言
- [x] 未 force/reset；未 `git add -A`
- [x] 本片 **未 push**（ahead 1 债仍挂）
- [x] 夹带 dirty：否（若 commit，仅精确 add 本片两文件）

## 6) tip / 提交 / push

- 基线：`42f2d09`
- 是否 commit：见回报（精确 add 本片文件成功则新 tip；否则仍 `42f2d09` + 工作区改动）
- 是否 push：**否**