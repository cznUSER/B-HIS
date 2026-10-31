# P0-D2-DIAGNOSIS-FEEDBACK-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`32718ed`（全 `32718edae87998e1cf3d654b73ddd480e66b668d`）
- 岗位：H05 · 本机实现（Codex exec 卡在 Windows apply_patch 后改由编排器本机落盘）；未用 CloudAgent；未 reset/force。

## 实现

| 文件 | 变更 |
| --- | --- |
| `web/src/components/feedback/useSubmitFeedback.js` | **薄加法** `diagnoseId`；诊断动作默认成功文案含诊断号；挂号 registerId/registerNo 语义保留 |
| `web/src/components/feedback/SubmitStatusBanner.vue` | 保留 `data-testid=register-submit-feedback`；诊断动作附加 `data-diagnosis-testid` |
| `web/src/api/diagnosisApi.js` | save/amend/commit/abate **调用方可选幂等键**；空键 VALIDATION_ERROR；补 `assertAmendDiagnoseRequest` 导入 |
| `web/src/api/mockDiagnoses.js` | idempotency 表；同键同指纹同 `diagnoseId`/同结果；冲突 409 |
| `web/src/views/DiagnosisView.vue` | 复用 D1 `createSubmitFeedback` + banner；save/amend/commit/abate 指纹持键；busy 含 submitting；失败保留草稿 |
| `web/scripts/check-diagnosis-submit-feedback.mjs` | **新建** 专项验收 |

禁区遵守：未改 Frame / OutpatientQueue* / StatusChain* / Register 业务接线 / DoctorDiagnosisPanel / Order 等他页 / Go / OpenAPI / diagnosisContract 业务语义。

## 验收

```text
node web/scripts/check-diagnosis-submit-feedback.mjs
# => P0_D2_DIAGNOSIS_FEEDBACK_OK  exit 0

node web/scripts/check-register-submit-feedback.mjs
# => P0_D1_REGISTER_FEEDBACK_OK  exit 0
```

覆盖：延迟锁定｜成功含 diagnoseId｜重复点同键不双单（save/amend/commit/abate）｜4xx ILLEGAL_STATE_TRANSITION / DIAGNOSE_NOT_AMENDABLE｜5xx/网络可恢复｜Register harness 不回归。

## Git

- feat commit：短 `5159de4` / 全 `5159de42b99f52d103982329f348892e63759f86`
- tip HEAD：以 `git rev-parse HEAD` 为准
- 分支：`codex/register-slot-card-20261028`
- `git push origin HEAD`：见提交后实测
- 未混交无关 dirty

`DONE_P0_D2`