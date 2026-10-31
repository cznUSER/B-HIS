# REG-R1-BACKEND-01 · 挂号患者选择后端/OpenAPI 收口

| 项 | 内容 |
|---|---|
| 任务编号 | **REG-R1-BACKEND-01** |
| 执行角色 | H04（业务逻辑开发） |
| 日期 | 2026-10-31（Asia/Shanghai） |
| 结果 | OpenAPI D1/D2 已补齐；限定 Go 测试通过；未运行回填或修改 migration 022 |
| Git | 未提交、未推送；待用户口头确认后再提交本切片 |

## 1. 实际改动文件

- `api/openapi.yaml`：仅对 `POST /patients/registration-search` 补充响应码与 keyword 描述；保持 `info.version: 0.3.11-draft`
- `deliverables/h04/reg-r1-backend-2026-10-31.md`：本交付记录

本任务开始前工作区已有大量未提交变更；`api/openapi.yaml` 亦已 dirty。本轮仅在目标 path 增补内容；未暂存、提交、推送或清理其他文件。

## 2. OpenAPI D1/D2 对照

| # | 依据 | 本次结果 |
|---|---|---|
| D1 错误响应 | 契约错误表：423 `ACCOUNT_LOCKED`、428 `PASSWORD_CHANGE_REQUIRED`、500 `INTERNAL_ERROR` | 三个响应均已加入目标 path；均声明 `application/json` 并引用现有 `#/components/schemas/ErrorBody`，未新增全局 component |
| D2 keyword 长度 | 契约要求最多 64 个 Unicode code point | 保留 `maxLength: 64`，并注明服务端按 Unicode code point（Go rune）计数，最多 64 个 |
| 水位 | 准出 D4 | 仅改描述与响应，仍为 `0.3.11-draft` |

## 3. 代码级复验

- `internal/app/registration_patient_search.go` 使用 `utf8.RuneCountInString(keyword) > 64` 检查非空关键字长度。
- `cmd/patient-search-backfill` 默认拒绝无 `-execute`；运行前及每批事务通过 `DB_NAME()` / 列就绪检查；单测覆盖错库、缺列、探测失败与有效 BHIS。
- migration 022 未修改、未执行；仅按 H06 结论复核守卫与 `TOP (1)` readiness（代码级，非库上执行证据）。

## 4. 测试

```text
go test ./internal/app ./internal/adapter/http ./internal/store/mssql ./internal/store/memory ./cmd/patient-search-backfill -count=1
```

结果：**PASS**（五包均通过）。未运行 `patient-search-backfill -execute`，未连接或写入生产/非隔离库。

## 5. 文档影响评估

检查路径：

- `docs/manuals/网站操作手册.md`
- `docs/manuals/实施部署手册.md`
- `docs/contracts/接口契约.md`

结论：**文档影响=无，无需同步修改**。操作手册已覆盖候选查询与 503 重试；部署手册已记 022/回填安全边界；接口契约已说明 keyword code point 与 423/428/500。本轮未改上述文档。

## 6. 未覆盖风险与后续

- 022 与回填未在本单执行；须由 H01 获准隔离库窗口验证。
- 真库联调 / H06 终审不在本单；本单测试不构成数据库验收证据。
- 按禁止动 `web/` 与非隔离库限制，未跑 Host/网页验收。
- 建议下一步：用户口头确认后仅提交本切片相关文件；H01 另窗跑 022/backfill；H06 真库联调。

## 7. Git 状态

本轮未执行 `git add` / `commit` / `push` / `reset` / `clean`。**待用户口头确认后再提交本任务切片。**