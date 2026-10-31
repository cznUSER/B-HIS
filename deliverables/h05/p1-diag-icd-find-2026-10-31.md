# P1-DIAG-ICD-FIND-01 交付记录

- 日期：2026-10-31（Asia/Shanghai）
- 分支：`codex/register-slot-card-20261028`
- 起始 tip：`929deaf`（全 `929deafa17bdc80acdc733712c926c1a207b2d53`；本机 ahead 5）
- 岗位：H05 · 本机落盘（优先 Codex；本片直接最小实现；未用 CloudAgent；未 reset/force）

## 诚实说明（强制）

| 项 | 口径 |
| --- | --- |
| 码表性质 | **演示 / Mock** 短表 36 条；UI 与模块注释标明「演示码表，非完整官方 ICD」 |
| 非官方全量 | **禁止**暗示完整 ICD-10/国临；**未**拷贝未授权完整 ICD 包 |
| 数据落点 | `web/src/workbench/mockIcdCodes.js` 常量 + 纯函数；**零网络** |
| 写入 assessment | token 格式 `{code} {name}`；空直写；非空「，」整段去重（`appendIcdToken`） |
| API / 主数据 | **零** 新 REST / 新表 / DictType=icd；**不碰 022**；不改 Go |
| 与常用关系 | COMMON 五条芯片保留并列；ICD 为搜索框+结果列表，不吞常用条 |

## 目标

门诊诊断页「初步诊断」旁（常用芯片下方）提供 ICD 检索（演示）：关键词过滤演示码表 → 选中填入/去重追加；busy/只读锁定；脏态联动既有 RESUME。

## 文件列表

| 文件 | 变更 |
| --- | --- |
| `web/src/workbench/mockIcdCodes.js` | **新建** DEMO 短表 + `filterIcdByKeyword` / `formatIcdAssessmentToken` / `appendIcdToken` |
| `web/src/views/DiagnosisView.vue` | **最小胶合**：常用条下 ICD 检索 UI、锁定 disabled、极薄 scoped CSS |
| `web/scripts/check-diagnosis-icd-find.mjs` | **新建** 演示标示/过滤/填入去重/空态/锁定/无 fetch 静态验收 |
| `deliverables/h05/p1-diag-icd-find-2026-10-31.md` | 本交付 |

禁区遵守：未改 outpatientContext / Frame / Queue / StatusChain / feedback 语义；未回改 COMMON / RESUME / PRINT / FIND；未改 diagnosisApi/Contract/Go/022；未改 OrderView / DoctorDiagnosisPanel；未混交无关 dirty。

## 行为说明

1. **演示标示**：标题「ICD 检索（演示）」+ disclaimer「演示码表，非完整官方 ICD」。
2. **检索**：空关键词展示前 12 条；有关键词按 code/name/pinyin/alias 过滤；无匹配显示「无匹配演示码」。
3. **填入**：选中 → `assessment` 写入 `{code} {name}`；去重；不改其它 structured 字段。
4. **脏态**：写入走既有响应式字段，RESUME 「有未暂存修改」自然亮起。
5. **锁定**：`editorLocked || busy` 或已含该 token 时结果按钮 disabled。
6. **常用并存**：COMMON 芯片与 ICD 检索并列；common harness 不回归。

## Harness

| 命令 | exit | 摘要 |
| --- | --- | --- |
| `node web/scripts/check-diagnosis-icd-find.mjs` | 0 | `P1_DIAG_ICD_FIND_OK` |
| `node web/scripts/check-diagnosis-common.mjs` | 0 | `P1_DIAG_COMMON_OK` |
| `node web/scripts/check-diagnosis-resume.mjs` | 0 | `P1_DIAG_RESUME_OK` |
| `node web/scripts/check-diagnosis-submit-feedback.mjs` | 0 | `P0_D2_DIAGNOSIS_FEEDBACK_OK` |

Order PRINT/FIND harness 本片未触达，未重跑（声明未改）。

## 残余

- 完整官方 ICD / 主数据 Dict / 提交强制编码 / 后端检索 API 属后续立项（破形态 A 须升级 H01）。
- origin 推送若失败以本机 tip + 本交付为准（ahead 债保留，勿强推）。
- 工作区仍有大量无关 dirty；本片仅 stage 上表文件。

`DONE_P1_DIAG_ICD_FIND`