# P1-DIAG-RESUME-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`8865146`（全 `8865146f0d47d1eccbb475c74362767bebbac98d`；本机 ahead 3）
- 岗位：H05 · 本机落盘（优先 Codex；本片直接最小实现；未用 CloudAgent；未 reset/force）

## 草稿策略（形态 A）

| 项 | 口径 |
| --- | --- |
| 草稿 | 服务端 `status=SAVE`（既有 `saveDiagnose` / `amendDiagnose`） |
| 恢复 | `listDiagnoses` → `applyDiagnosis`；不依赖 localStorage 病历正文 |
| 本片 | 脏快照 + `onBeforeRouteLeave` / `beforeunload` + 切挂号确认 + 本机 `lastLocalSaveAt`（非 PHI）+ 失败不覆盖 |
| 禁 | 新 draft/autosave REST；临床正文/凭据进 storage；DTO 加 `updatedAt`；定时 autosave / 本地病历仓 |

## 目标

门诊诊断页：脏态可感知、离页/刷新守卫、切挂号确认、暂存成功本机时间提示、失败保留编辑器与 SAVE 列表、回页靠服务端 SAVE 恢复。

## 文件列表

| 文件 | 变更 |
| --- | --- |
| `web/src/workbench/diagnosisDirtyGuard.js` | **新建** `snapshotOf` / `isDirty` / 文案常量 / `resumeSaveHint`（零网络；禁止存病历） |
| `web/src/views/DiagnosisView.vue` | **最小胶水**：基线快照、脏提示、离页/beforeunload、切挂号确认、`lastLocalSaveAt`、失败路径不覆盖 |
| `web/scripts/check-diagnosis-resume.mjs` | **新建** 脏/锁定/提示/静态接线验收 |
| `deliverables/h05/p1-diag-resume-2026-10-31.md` | 本交付 |

禁区遵守：未改 outpatientContext / Frame / Queue / StatusChain / useSubmitFeedback / SubmitStatusBanner 反馈语义；未触 Order PRINT/FIND；未改 diagnosisApi/Contract/Go DTO；未混交无关 dirty。

## 行为说明

1. **脏态**：相对上次成功 `applyDiagnosis` / 暂存后的 structured + templateCode + revisionReason + amendSourceId；变更时显示「有未暂存修改」；`editorLocked` 永不脏。
2. **离页**：`onBeforeRouteLeave` 脏时 `confirm`；取消留页不丢内容。
3. **刷新/关页**：脏时 `beforeunload` 原生提示；不脏不拦。
4. **切挂号**：`chooseRegister` / query watch 脏时先确认；取消还原选中与路由；确认后再 `loadDiagnoses`。
5. **暂存提示**：成功 save/amend 后「本机上次暂存成功 · HH:mm:ss」；刷新后仅有服务端 SAVE 时诚实显示「已暂存（服务端）」，不伪造精确服务端时间。
6. **失败不覆盖**：save/amend catch 仅 `submitFeedback.fail` + 错误文案；不 `applyDiagnosis(null)`、不清 `diagnoses` 列表；可复用 D2 幂等重试键。
7. **恢复**：成功暂存后离开再回同挂号 → `listDiagnoses` 可见 SAVE；零 web storage 病历。

## Harness

| 命令 | exit | 摘要 |
| --- | --- | --- |
| `node web/scripts/check-diagnosis-resume.mjs` | 0 | `P1_DIAG_RESUME_OK` |
| `node web/scripts/check-diagnosis-submit-feedback.mjs` | 0 | `P0_D2_DIAGNOSIS_FEEDBACK_OK` |

Order PRINT/FIND harness 本片未触达，未重跑（声明未改）。

## 残余

- ICD / 常用诊断 / 版本对比 / 诊断打印 / 定时 autosave 属后续子片。
- SAVE 态已有 `diagnoseId` 时原页 `canSave` 仍不允许再次「保存诊断」（既有行为）；脏提示仍诚实。
- 要服务端精确 `updatedAt` 属契约形态突破，本片未做。
- origin 推送若失败以本机 tip + 本交付为准。
- 工作区仍有大量无关 dirty；本片仅 stage 上表文件。

`DONE_P1_DIAG_RESUME`
