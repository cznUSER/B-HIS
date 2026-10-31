# UI90-04A-FIX-01 · H06 抽验（borderWidth / transform）· **不签 90%**

- 岗位：H06（独立验证；只建议通过/不通过）
- 时间：2026-11-01（Asia/Shanghai）
- 机：`0b530576-3220-453b-8d1f-d68465218648`；仓库 `E:\B-HIS`
- 目标 tip：`be229e452532b8e1121420ab05d6be6d1c9de4e2`（`be229e4 fix(h05): UI90-04A-FIX-01 password change 2px borders no motion`）
- HEAD 确认：`git rev-parse HEAD` = 上述 tip；`origin/...` left-right = `0 0`（已 push）
- 依据：H01 请抽验；H05 交付 `deliverables/h05/ui90-04a-fix-01-2026-11-01.md` + `deliverables/h05/ui90-04a-fix-01/`
- 方法：`git show be229e4:web/src/views/PasswordChangeView.vue` 源码核对 + 只读重跑 `node deliverables/h05/ui90-04a-fix-01/token-probe.cjs`（exit 0）
- **本抽验不签 UI90-04A 90%**；禁改业务码（本抽验零改 Vue/Go）

## 结论

**建议通过**

验收点 A/B（边框 2px + hover/active transform=none）与三视口无新横溢均成立；H05 交付 md 已明确声明不签 90%。残余 SCORE / HYT 截图 / 几何微调不阻塞本薄修复片通过建议。

## 验收点核对

| # | 验收点 | 结果 | 证据 |
|---|---|---|---|
| 1 | dialog / button `borderWidth=2px` | **PASS** | 源码：`.ui2-password-dialog { border: 2px ... }`；`.button { border: 2px ... }`。probe base：dialog/button 四边均为 `2px`（1440/1024/390） |
| 2 | form 左右/底 `2px`（顶接 header 可为 0） | **PASS** | 源码：`border: 2px solid #a5a5a5; border-top: 0`。probe：`borderTopWidth=0px`，左右/底 `2px` |
| 3 | hover / active `transform=none` | **PASS** | 源码：基态与 `:hover`/`:active`（含 blocked）均显式 `transform: none; transition: none`。本机重跑 probe：hover/active `transform=none`，`borderTopWidth=2px` |
| 4 | 三视口无新横溢 | **PASS** | 本机重跑：`overflows [[1440,false],[1024,false],[390,false]]`；与交付 `token-probe-results.json` / `settled-states.json` 一致 |
| 5 | 交付 md 声明不签 90% | **PASS** | H05 md 标题与 §0「明确声明：本片不签 90%」；commit message 亦含 `Does NOT sign UI90-04A 90%` |

## tip / 改动范围抽核

- commit 路径仅：`web/src/views/PasswordChangeView.vue` + `deliverables/h05/ui90-04a-fix-01*`（无 Go/迁移/全局 design-system）
- diff 实质：dialog/form/button `1px→2px`；按钮 hover/active 覆盖全局 motion 为 `transform:none`
- 工作区虽脏，本抽验未 `reset`/`checkout`；以 `git show be229e4` + 交付目录探针为准

## 本机重跑摘要（只读）

```
node deliverables/h05/ui90-04a-fix-01/token-probe.cjs  → exit 0
ok: true; fail: []
settled: dialogBorder=2px; formBorderTop=0px; formBorderLeft=2px;
         normal/hover/active borderWidth=2px, transform=none
overflows: 1440/1024/390 均为 false
```

说明：探针为隔离 HTML（全局 `.button` 泄漏 + 本页 scoped CSS 拷贝），非全量 Vue mount——与 H05 交付 note 一致；对本片 A/B token 闭合足够。

## 是否改业务

**否**（H06 本片仅抽验 + 本文档；未改 PasswordChangeView / CSS / Go / harness 业务断言）。

## 对看板 / 90%

**不得**因本通过建议将 UI90-04A 标为视同 90% done；须另立 SCORE。本抽验亦不关闭整项 UI90。
