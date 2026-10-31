# REG-DOC-01 — registration-search 手册澄清（H18）

| 项 | 内容 |
|---|---|
| 任务编号 | **REG-DOC-01** |
| 执行岗位 | H18 实施培训与一线支持 |
| 日期 | 2026-10-31（Asia/Shanghai） |
| 派工 | H01；依据 H06 R5「给 H18」建议 |
| 性质 | **仅手册/说明文档**；禁改业务源码；未 git commit / push |
| 文档影响评估 | 有：`实施部署手册`、`网站操作手册`；未改 `api/openapi.yaml` / 业务 Go/Vue（OpenAPI tip `0.3.11-draft` 已含空 keyword→全量分页与 503 边界，本单不扩改契约文件） |

---

## 结论

已按现行 tip `0.3.11-draft` 与 H01 R3 / H06 R5 证据，在两本手册补齐澄清：

1. migration **022** 列就绪且回填对账 **`pending_rows = 0`** 后，`POST /api/v1/patients/registration-search` 应返回 **200 分页**。
2. **空 keyword**（或请求体缺省 keyword）= 全部未删除患者分页，**不是 400**；过短 keyword 合法；超长（>64 Unicode code point）等校验失败才 400。
3. **503 仅表示**助记码列缺失或仍有未回填行（可重试）；回填完成后不应再长期出现 503。
4. INACTIVE 真库样本与 RegisterView 折叠导致的全量 picker UI 脚本结果 **不在本句验收范围**。

---

## 依据（只读）

| 来源 | 路径 |
|---|---|
| H06 R5 | `deliverables/h06/reg-patient-select-r5-review-2026-10-31.md`（§8 给 H18） |
| H01 R3 | `deliverables/h01/reg-r3-true-api-2026-10-31.md`（空 keyword→200 total=125；无 503） |
| 契约 tip | `api/openapi.yaml` path `/patients/registration-search`；health `0.3.11-draft` |

未另开契约变更；未按「空/过短→400」错误简报改文档口径。

---

## 已改文件

| 文件 | 变更摘要 |
|---|---|
| `docs/manuals/实施部署手册.md` | §迁移 022/回填后新增「澄清（REG-DOC-01）」段；更新版次表「本次修订/最近更新」与文末变更说明 |
| `docs/manuals/网站操作手册.md` | 挂号·选患步骤 2/4 与流程更新说明同步 200/空 keyword/503 边界；更新版次表 |

未改：`api/openapi.yaml`、`docs/contracts/接口契约.md`（后者无 registration-search 专节；OpenAPI 已描述空 keyword 与 503）、任何业务源码。

---

## Git

- 工作区：`docs/manuals/实施部署手册.md`、`docs/manuals/网站操作手册.md`、本交付 md 为本任务范围。
- **未提交**：按派工「按任务分批（仅手册相关）需用户/你岗确认后提交；可先改文件回我」。
- 请 H01 / 用户确认后，由本岗或指定岗以**仅手册相关**文件单独 commit（建议信息含 `REG-DOC-01`），再按规程推送。

---

## 非范围 / 声明

- 不关闭 REG-PATIENT-SELECT-01；不代替 H06 有条件通过边界。
- 不宣称其他库或生产已就绪；隔离实例证据仍以 R2–R5 交付为准。
- 未连接数据库、未跑 migration/backfill、未改生产配置、未导出 PII。

## 交付路径

- 主文档：`E:\B-HIS\deliverables\h18\reg-doc-registration-search-2026-10-31.md`
- 手册：`E:\B-HIS\docs\manuals\实施部署手册.md`、`E:\B-HIS\docs\manuals\网站操作手册.md`
