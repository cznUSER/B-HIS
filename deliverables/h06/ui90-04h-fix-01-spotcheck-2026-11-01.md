# UI90-04H-FIX-01 抽验 · H06 spotcheck（2026-11-01）

> **角色**：H06 独立抽验｜只建议通过/不通过｜**不签 UI90 90%**｜不定需求口径｜不关看板｜不改业务码｜不 push｜禁 force/reset｜禁 qq-mail｜不混 SCORE/PRELIVE  
> **本机未重跑 probe**：以 H05 交付产物 + git 证据交叉验证。

## 结论

**建议：通过**

理由：实现 tip `4667d05` 父即为 baseline `1a70c90`；业务路径仅 `RegisterView.vue`（template/chrome）+ `register-slot-card.css`（末尾追加）；独立核 `<script setup>` 归一化后字节一致；`clinical-design.css` blob 未变；CSS 选择器均挂在 `.register-slot-card-page` / `.shell-main .register-page.register-slot-card-page`；probe JSON 三视口 `allPass=true` 且 1440 `pageW=980` / `topH=58` / panels / `overflow=false`。范围未越界到 Patient/Order/Charge/Go/022。C 类主 CORE 仍缺 → **记债不否本 chrome 片**。

## Tip 链（实测）

| 角色 | short | full | parent |
|---|---|---|---|
| tipBaseline | `1a70c90` | `1a70c909f54e9674f78de618341bb6c733424bc8` | `90eb38c…` |
| 实现 tip | `4667d05` | `4667d05fc451b43c0e8249a8183c3bde0822652b` | **`1a70c90`**（吻合） |
| H05 tipFinal（抽验前 HEAD） | `c858330` | `c858330080cb88f43847b81943f97595c7c76dca` | `4667d05` |
| H06 tipFinal | `737aaab` | `737aaab5d936059060820d8adcc48366a58000ac` | `c858330` |

分支：`codex/register-slot-card-20261028`。  
抽验开工时 ahead/behind vs `origin/codex/register-slot-card-20261028`：**ahead 2 / behind 0**（未 push；本片亦不 push）。

> H05 md 行仍写 `tipFinal：20c5dfc`（与真实 HEAD `c858330` 不符；c858330 仅把实现 tip 改成 4667d05，tipFinal 笔误未清）。**以 HEAD/`4667d05` 为准**；记文档债，不因此否 chrome。

## 范围核对（`git diff 1a70c90..4667d05 --name-only`）

业务相关仅：

- `web/src/views/RegisterView.vue`（+data-testid；`.register-top-chrome` 包裹既有 heading）
- `web/src/views/register-slot-card.css`（末尾 +11 行 FIX-01 块）

另有 H05 交付：`deliverables/h05/ui90-04h-fix-01*`。  
**无** clinical-design / Patient / Order / Charge / `*.go` / migration 022。

`clinical-design.css` blob 双侧均为 `b1962a3bc510eda53f00b2d4b7b19c5a0770fe1f`。

## RegisterView script 独立核

从 tip blob 提取 `<script…>…</script>`，CRLF→LF 归一化后：

- `scriptByteIdentical=true`
- len 26546 / 26546
- SHA256（UTF-8）：`189bf699f9b219c2a4ab4595a25f328e753f550efab4f22eff9f7c07bc38ecce`（双侧同）

template 变更仅 chrome 包裹与 testid；未见业务写路径改动（script 未动）。

## CSS 独立核

末尾 FIX-01 块含：

- `--register-shell-w:980px` / `--register-top-h:56px`
- heading / flow-row / card-panels / flow-panel chrome
- `@media(max-width:1280px)` 强制 `max-width:100%`（防窄屏横溢）

既有 CSS 行为：diff 仅为文件末追加（`existingCssUnchanged` 成立）。选择器 register-only。

## Probe 产物交叉核（未本机重跑）

源：`deliverables/h05/ui90-04h-fix-01/after-results.json`（`allPass: true`）

| 视口 | pageW | topH | panels | overflow | pass |
|---:|---:|---:|---|---|---|
| 1440 | 980 | 58 | true | false | true |
| 1024 | 944 | 58 | true | false | true |
| 390 | 310 | 83 | true | false | true |

invariants 自报且与 git 交叉一致：`scriptByteIdentical` / `existingCssUnchanged` / `registerOnlySelectors` / `noClinicalDesignTouch` / `scopedShellWidth`。  
截图存在：`after-1440.png` / `after-1024.png` / `after-390.png`。fixture 标注 UI-only / 非活体。

## Dirty / 债

- **工作树脏**：大量与本片无关的 modified/untracked（Go/web/docs/.codex 等）。只报告；**未** force/reset/restore。
- **C 类主 CORE 仍缺**（记债；不否本 chrome）。
- **未做** FIX-02 / BTN-WIRE / 列表 Morph / SCORE / PRELIVE / 90% 签收。
- H05 md `tipFinal:20c5dfc` 笔误（文档债）。
- push debt：ahead≥2；本抽验 **不 push**。

## 停工项

无（抽验完成；不关看板；不改业务码）。
