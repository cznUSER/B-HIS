# P1-DIAG-VERSION-DIFF-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 基线 tip：`120a409`（全 `120a409a5b8748fecd8a27e89178fa6455272939`；origin tracking 0/0）
- 岗位：H05 · 本机落盘（优先 Codex；`apply_patch` PowerShell UTF-8 受阻后由执行器最小实现；未用 CloudAgent；未 reset/force）

## 策略（形态 A）

| 项 | 口径 |
| --- | --- |
| 数据来源 | 仅当前页已有 `diagnoses`（`listDiagnoses` 回读） |
| 对比内容 | 结构化六键 + 模板码 / 状态 / 版本号 / 修订原因 |
| 选版 | 历史区「对比版本」；默认优先当前行 `supersedesDiagnoseId` 链，否则列表前两行；面板内可改 A/B |
| 与编辑器 | 独立只读面板；打开/关闭不 `applyDiagnosis` / `resetDraft`；不毁脏态 |
| API / storage | **无** 新 REST；**无** localStorage/sessionStorage |

## 目标

门诊诊断页历史区可对 ≥2 条记录做只读字段 diff；<2 条入口禁用并诚实提示；全同显示「无字段差异」。

## 文件列表

| 文件 | 变更 |
| --- | --- |
| `web/src/workbench/diagnosisVersionDiff.js` | **新建** `pickStructured` / `diffDiagnosisRecords` / `canOpenVersionDiff` / `defaultDiffPair` |
| `web/src/views/DiagnosisView.vue` | **最小接线**：对比入口、A/B 选择、只读面板、极薄 scoped CSS |
| `web/scripts/check-diagnosis-version-diff.mjs` | **新建** 本片 harness |
| `deliverables/h05/p1-diag-version-diff-2026-10-31.md` | 本交付 |

禁区遵守：未改 outpatientContext / Frame / Queue / StatusChain / feedback 语义；未回改 diagnosisDirtyGuard / commonDiagnoses / mockIcdCodes；未触 Order PRINT/FIND/COMMON；未改 Go/OpenAPI/022/Dict；未混交无关 dirty。

## 行为说明

1. **入口**：`diagnosis-version-diff-open`；`diagnoses.length < 2` 时 disabled + `DIFF_NEED_TWO_HINT`。
2. **差异可见**：字段并排 A/B；`changed` 行高亮；testid `diagnosis-version-diff-field-{key}`。
3. **全同空态**：`identical` → `DIFF_NO_CHANGES_HINT`（`diagnosis-version-diff-empty`）。
4. **只读**：对比不调用 save/amend/commit/abate；`openVersionDiff` / `closeVersionDiff` 不触编辑器装载。
5. **不毁脏态**：不改 RESUME 守卫与 `chooseDiagnosis` 确认语义。

## Harness

| 命令 | exit | 摘要 |
| --- | --- | --- |
| `node web/scripts/check-diagnosis-version-diff.mjs` | 0 | `P1_DIAG_VERSION_DIFF_OK` |
| `node web/scripts/check-diagnosis-resume.mjs` | 0 | `P1_DIAG_RESUME_OK` |
| `node web/scripts/check-diagnosis-common.mjs` | 0 | `P1_DIAG_COMMON_OK` |
| `node web/scripts/check-diagnosis-icd-find.mjs` | 0 | `P1_DIAG_ICD_FIND_OK` |
| `node web/scripts/check-diagnosis-submit-feedback.mjs` | 0 | `P0_D2_DIAGNOSIS_FEEDBACK_OK` |

Order 薄包本片未触达，未重跑（声明未改）。

## 残余

- autosave / 诊断打印 / 官方 ICD / 三路以上对比属后续或禁项。
- Codex exec 本机 PowerShell `apply_patch` UTF-8 受阻，本片由执行器落盘收口。
- 工作区无关 dirty 极大；本片仅 stage 上表文件。
- tip / push 以 commit 后回报为准。

`DONE_P1_DIAG_VERSION_DIFF`
