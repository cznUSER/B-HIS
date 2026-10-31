# P1-DIAG-COMMON-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`5fa6b06`（全 `5fa6b06528b7e88cb897ca0d4c12684df4eda16a`；本机 ahead 4）
- 岗位：H05 · 本机落盘（优先 Codex；本片直接最小实现；未用 CloudAgent；未 reset/force）

## 策略（形态 A）

| 项 | 口径 |
| --- | --- |
| 列表来源 | 前端常量 `COMMON_DIAGNOSES`（与医生站五条对齐） |
| 填入 | 点击芯片 → 空则写入 `assessment`；非空则「，」去重追加 |
| storage | **禁** localStorage/sessionStorage 落常用 |
| API | **零** 新 REST/新表；不扩 ICD |
| Order / 医生站 | 本片不改 `OrderView` / `DoctorDiagnosisPanel` |

## 目标

门诊诊断页「初步诊断」旁可见常用诊断芯片；一键填入/去重追加；`busy`/`editorLocked` 下不可点；联动既有 RESUME 脏快照。

## 文件列表

| 文件 | 变更 |
| --- | --- |
| `web/src/workbench/commonDiagnoses.js` | **新建** 常量 + `appendDiagnosisName` / `assessmentIncludesName` |
| `web/src/views/DiagnosisView.vue` | **最小胶合**：assessment 旁芯片条、锁定态 disabled、极薄 scoped CSS |
| `web/scripts/check-diagnosis-common.mjs` | **新建** 填入/去重/锁定/无 storage 静态验收 |
| `deliverables/h05/p1-diag-common-2026-10-31.md` | 本交付 |

禁区遵守：未改 outpatientContext / Frame / Queue / StatusChain / feedback 语义；未回改 diagnosisDirtyGuard / resume harness；未触 Order PRINT/FIND；未改 diagnosisApi/Contract/Go/022；未改 DoctorDiagnosisPanel；未混交无关 dirty。

## 行为说明

1. **芯片可见**：初步诊断字段下「常用诊断」条；五条：上呼吸道感染、急性咽炎、高血压、2型糖尿病、胃炎。
2. **填入/追加**：空 `assessment` 直写；非空按「，」去重追加；不改其它 structured 字段。
3. **脏态**：写入走既有响应式字段，RESUME 「有未暂存修改」自然亮起。
4. **锁定**：`editorLocked || busy` 或已含该名时芯片 disabled。
5. **无 storage/无 ICD**：常量模块零网络；视图无 localStorage/sessionStorage。

## Harness

| 命令 | exit | 摘要 |
| --- | --- | --- |
| `node web/scripts/check-diagnosis-common.mjs` | 0 | `P1_DIAG_COMMON_OK` |
| `node web/scripts/check-diagnosis-resume.mjs` | 0 | `P1_DIAG_RESUME_OK` |
| `node web/scripts/check-diagnosis-submit-feedback.mjs` | 0 | `P0_D2_DIAGNOSIS_FEEDBACK_OK` |

Order PRINT/FIND harness 本片未触达，未重跑（声明未改）。

## 残余

- ICD 检索（`P1-DIAG-ICD-FIND-01`）/个人常用落库/医生站常量抽公共对齐/版本对比/诊断打印属后续。
- origin 推送若失败以本机 tip + 本交付为准。
- 工作区仍有大量无关 dirty；本片仅 stage 上表文件。

`DONE_P1_DIAG_COMMON`
