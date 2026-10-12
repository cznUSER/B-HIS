# B-HIS

医院信息系统（H Phase1）。Go 登录 API 在仓库根目录；Vue 登录界面在 `web/`。

门诊一期当前实现 **登录鉴权**、**患者建档**（`POST/GET /api/v1/patients`）、**挂号**（`POST/GET /api/v1/registers` 与退号）和 `GET /health`。`api/openapi.yaml` 为 **0.3.6-draft**。接诊、完成就诊、诊断，以及 deprecated `/encounters` 仍未注册。

H03 `0.3.0-draft` 没有 Auth。Auth 路径保持不变，患者建档规则仍是 0.3.5（必填 name/sex/birthday/fareIdentity）。成功体按 OpenAPI：`LoginResponse` / `UserDto` / `PatientDto`，不包 `success/data`。失败体按《接口契约》§1.2：`success: false`、`data: null`、`error.code`、`requestId`。

## Web 登录界面

```bash
cd web
npm install
npm run dev
```

也可用 pnpm：`pnpm install` 然后 `pnpm dev`。浏览器打开终端里提示的本地地址（默认 http://localhost:5173）。

登录成功进入首页看板 `/app`（待办、经营概况、两张分布图都是演示占位，数字为 0）。左侧「门诊业务」是建档、挂号、开单、收费、发药、退药/退费；「档案管理」里的患者建档走同一条建档路由。患者建档页可以新建和查询。挂号页可选科室和医生；没有患者时提示先建档。默认仍是 Mock；`web/.env.local` 里把 `VITE_USE_MOCK` 设为 `false` 后，建档走 `POST/GET /api/v1/patients`，挂号走 `POST/GET /api/v1/registers` 与 `POST /api/v1/registers/{id}/cancel`（字段 `departmentId`）。开单及之后的模块页仍是空态。六张入口卡只在 `/app/outpatient`，不是登录后的默认页。

默认是 **Mock**（`VITE_USE_MOCK=true`），不访问后端。Mock 与真实 API 使用同一套演示账号 `system` / `1`。员工默认密码也是 `1`。

切换到 H04 真实 API 的步骤见 [web/README.md](web/README.md)「本机真 API 联调」。联调时先按下一节启动 Go API（默认 `127.0.0.1:8080`）。

## Go 登录 API（默认内存库，不需要 SQL Server）

- Go **1.22+**
- 默认 `Database.Driver: memory`
- 监听 `127.0.0.1:8080`

```bash
go run ./cmd/api -config config.example.yaml
```

`config.example.yaml` 里的 JWT 是占位符。共享环境请用环境变量盖掉，不要把真实口令写进 Git：

| 变量 | 作用 |
|------|------|
| `BHIS_API_ADDR` | 监听地址，默认 `127.0.0.1:8080` |
| `BHIS_JWT_SECRET` | HS256 密钥 |
| `BHIS_ACCESS_TOKEN_TTL` | access token 寿命，默认 `8h`（28800 秒）。Phase1 没有 refresh |
| `BHIS_DB_DRIVER` | `memory`（默认，CI）或 `sqlserver` |
| `BHIS_DB_DSN` | 仅 `sqlserver` 需要。见下方占位形状 |
| `BHIS_LOG_DIR` | 日志目录，默认 `./logs` |
| `BHIS_LOG_LEVEL` | `debug` / `info` / `warn` / `error` |
| `BHIS_CORS_ORIGINS` | 逗号分隔。默认放行 Vite：`http://localhost:5173`、`http://127.0.0.1:5173` |
| `BHIS_CONFIG` | 可选 YAML 路径 |
| `BHIS_OPEN_BROWSER` | `0` / `false` / `no` 时，host 启动后不打开浏览器 |

进程环境变量优先于 YAML。若变量尚未设置，会再读当前目录下 gitignore 的 `.env.local`，然后 `.env`。

也可以：

```bash
go run ./cmd/host -config config.example.yaml start
go run ./cmd/host -config config.example.yaml health
go run ./cmd/host -config config.example.yaml logs
```

Host 在这个构建里是 **Windows 本机演示启动器**（还不是托盘窗）。无参数会启动内存库 API、同源托管登录页，并打开浏览器。验收用的无头进程仍是 `cmd/api`。

## Windows 双击启动

一个 exe 同时提供登录页和 API，运行时不需要 Vite。默认内存库。浏览器打开 http://127.0.0.1:8080/login 。演示账号是 `system` / `1`（员工默认密码；Mock 相同；不是 SQL Server 口令）。

在仓库根目录重新构建网页并打出 exe：

```powershell
powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1
```

产物：`bin\b-his-host.exe`。双击它，或在该目录打开命令行直接运行。窗口要留着；关掉窗口或按 Ctrl+C 即停止。登录页和 `/api/v1` 在同一个进程。

Linux 上交叉编译同样用 `scripts/build-windows.sh`（需要 Node 与 Go），得到同一个 `bin/b-his-host.exe`。

常用启动方式：

```text
bin\b-his-host.exe
bin\b-his-host.exe -no-browser
bin\b-his-host.exe -config config.local.yaml start
bin\b-his-host.exe health
bin\b-his-host.exe logs
```

| 项 | 说明 |
|----|------|
| 不必带配置文件 | 没有 YAML、也没有 `BHIS_JWT_SECRET` 时，用 `config.example.yaml` 里的 JWT 占位符，数据库为 memory。日志会警告这是占位符。 |
| 配置放哪 | 当前目录或 exe 同目录的 `config.local.yaml`、`config.yaml`，其次才是 `config.example.yaml`。也可以 `-config` 或 `BHIS_CONFIG`。 |
| SQL Server | 仍是可选项。在 exe 旁边放 gitignore 的 `config.local.yaml`，或设置 `BHIS_DB_DRIVER=sqlserver` 和 `BHIS_DB_DSN`。不要把 DSN 或口令提交到 Git。 |
| 不打开浏览器 | `-no-browser`，或 `BHIS_OPEN_BROWSER=0`。 |
| 开发时的 Mock | `cd web` 后 `npm run dev` 不变，仍是 Mock（`system` / `1`）。exe 用的是 `npm run build:demo`（关闭 Mock，请求同源 `/api/v1`）。 |

启动失败时，双击打开的窗口会停住并提示按 Enter，避免报错一闪而过。端口默认 `127.0.0.1:8080`；被占用时改 `BHIS_API_ADDR` 或配置里的 `Host.ApiAddr`。日志仍在工作目录的 `./logs`（双击时工作目录一般就是 exe 所在文件夹）。

## 内存演示账号

用户名区分大小写。密码只以 bcrypt 保存。

| 用户名 | 密码 | 角色 | 说明 |
|--------|------|------|------|
| `system` | `1` | `SYSTEM` | 系统管理员，可登录 Go API。显示名「系统管理员」 |
| `disabled` | `1` | 无 | 密码正确时 `403 ACCOUNT_DISABLED`；密码错误仍是 `401 INVALID_CREDENTIALS` |

员工账号的默认密码是 `1`。以后新增的员工种子也用这个口令，不要再写另一套演示密码。Mock、内存库和 SQL 种子一致；SQL 脚本只存 bcrypt。请在正式环境改掉演示口令。这不是 SQL Server 的登录口令。

## curl

登录 `200`，体为 `LoginResponse`（`expiresIn` 默认 `28800`）：

```bash
curl -sS -D - http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"system","password":"1"}'
```

```json
{
  "accessToken": "<jwt>",
  "tokenType": "Bearer",
  "expiresIn": 28800,
  "user": {
    "id": "00000000-0000-4000-8000-000000000001",
    "username": "system",
    "displayName": "系统管理员",
    "roles": ["SYSTEM"]
  }
}
```

当前用户：

```bash
curl -sS http://127.0.0.1:8080/api/v1/auth/me \
  -H "Authorization: Bearer $TOKEN"
```

登出 `204` 空体。服务端把 JWT 的 `jti` 写入黑名单，之后同一 token 再访问为 `401 TOKEN_INVALID`：

```bash
curl -sS -D - -o /dev/null -X POST http://127.0.0.1:8080/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN"
```

失败包络（`X-Request-Id` 可传入，也会出现在 `requestId`）：

| 调用 | HTTP | `error.code` |
|------|------|----------------|
| 用户名或密码错误（两种情况文案相同，不说明是哪一个） | 401 | `INVALID_CREDENTIALS` |
| 不带 `Authorization` | 401 | `UNAUTHORIZED` |
| token 签名坏、格式坏、或已登出 | 401 | `TOKEN_INVALID` |
| token 过期 | 401 | `TOKEN_EXPIRED` |
| 密码正确但账号停用（`disabled`） | 403 | `ACCOUNT_DISABLED` |
| 同一 IP 或用户名失败次数达到上限（默认 8 次 / 15 分钟） | 429 | `AUTH_RATE_LIMITED` |
| 缺字段或 JSON 不合法 | 400 | `VALIDATION_ERROR` |

```bash
curl -sS -D - http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"system","password":"wrong"}'
```

```json
{
  "success": false,
  "data": null,
  "error": { "code": "INVALID_CREDENTIALS", "message": "用户名或密码错误", "details": {} },
  "requestId": "..."
}
```

```bash
curl -sS -D - http://127.0.0.1:8080/api/v1/auth/me
```

`401`，`error.code` 为 `UNAUTHORIZED`。

过期：把 `BHIS_ACCESS_TOKEN_TTL` 设为 `1s`，登录后等待再调 `/api/v1/auth/me`，得到 `401 TOKEN_EXPIRED`。

JWT：HS256。声明 `sub`（userId）、`username`、`roles`、`iat`、`exp`、`jti`。请求头 `Authorization: Bearer <accessToken>`。

## 患者建档

需 Bearer。建档角色为 **SYSTEM / REGISTRAR / DOCTOR / CASHIER**（权限 `patient:create`，SYSTEM 走 `admin:*`）。查询和详情对所有已登录且未停用的角色开放，含 PHARMACIST。角色以用户表为准，与 `GET /auth/me` 相同，不只看 JWT 里的快照。

必填：`name`、`sex`（`M`/`F`/`U`）、`birthday`、`fareIdentity`。`address` 可空（缺省、`""`、`null` 均可）。`idNumber`、`cardNo`、`phone`、`idType` 选填。

生日是纯日期，按 **Asia/Shanghai** 的当天比较：必须 `> 1900-01-01` 且 `≤` 当天。非法为 `400 VALIDATION_ERROR`。库表 CHECK 只保证 `> 1900-01-01`（见已提交的 `002_patient`）；「不能晚于今天」由应用校验。

证件号、卡号非空才查重，冲突 **拒绝且不合并**。`idNumber` 按号码本身唯一（不按 `idType` 组合），`409 DUPLICATE_ID_NUMBER`。`cardNo` 冲突为 `409 DUPLICATE_CARD_NO`。空 `idNumber` 可以多次建档。未发卡 `cardStatus=NOT_SENT`；带了 `cardNo` 则为 `NORMAL`。`status` 默认 `ACTIVE`。`hospPkNo` / `patientNo` 由服务生成（`H`/`P` + 上海日期 + 随机后缀）。

查询参数 `cardNo`、`idNumber`、`hospPkNo` 为精确匹配；`name`、`address` 为包含匹配。不带任何条件时返回 `{"items":[]}`。未知 `patientId` 为 `404 NOT_FOUND`。

`Idempotency-Key` 可选。同一进程内相同键且请求体一致则返回第一次的 `201`；同键不同体为 `409 IDEMPOTENCY_KEY_CONFLICT`。官方 `002_patient` 没有幂等表，键不落库。

成功建档写审计 `patient.create`，`resource_type=Patient`。

```bash
curl -sS -D - http://127.0.0.1:8080/api/v1/patients \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"张三","sex":"M","birthday":"1990-01-15","fareIdentity":"SELF"}'
```

## 挂号

需 Bearer。挂号和退号角色为 **SYSTEM / REGISTRAR**（权限 `encounter:create`、`encounter:cancel`，SYSTEM 走 `admin:*`）。查询和详情对所有已登录且未停用的角色开放。角色以用户表为准，与 `GET /auth/me` 相同。

`POST /api/v1/registers` 返回 `201` `RegisterDto`，初始 `status=NORMAL`。必填 `patientId`、`departmentId`、`doctorId`、`visitDate`。缺字段为 `400 VALIDATION_ERROR`。`registerSort` 可选；`registerFee` 默认 `0`；`registerCharged` 默认 `false`。患者不存在 `404 PATIENT_NOT_FOUND`；科室不存在 `404 DEPARTMENT_NOT_FOUND`；医生不存在 `404 DOCTOR_NOT_FOUND`。`registerId` 与 `registerNo` 由服务生成。`registeredAt` 为创建时间。

内存库种子与 `003_register_seed.up.sql` 一致：`dept-1` 内科、`dept-2` 外科、`doc-1` 张医生、`doc-2` 李医生。

`GET /api/v1/registers` 的查询参数 `patientId`、`registerNo`、`visitDate` 均可选，精确匹配，条件之间为 AND。不带任何条件时返回 `{"items":[]}`，与患者查询相同。未知 `registerId` 为 `404 NOT_FOUND`。

`POST /api/v1/registers/{registerId}/cancel`：仅 `NORMAL` → `BACK`，并写入 `backOperatorId` / `backOperatedAt`（当前用户）。`USED` / `DIAGNOSING` / `DIAGNOSED` / `PAUSED` 为 `409 REGISTER_NOT_CANCELLABLE`，状态不变。已是 `BACK` 再退为 `409 REGISTER_BACK_NOT_ALLOWED`，退号留痕不覆盖。

`Idempotency-Key` 只对挂号创建在进程内生效。同键同体返回第一次的 `201`；同键不同体为 `409 IDEMPOTENCY_KEY_CONFLICT`。官方 `003_register` 没有幂等表，键不落库。

成功挂号写审计 `register.create`，成功退号写 `register.cancel`，`resource_type=Register`。退号拒绝写同一动作的 `FAIL`。

```bash
curl -sS -D - http://127.0.0.1:8080/api/v1/registers \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"patientId":"<patientId>","departmentId":"dept-1","doctorId":"doc-1","visitDate":"2026-10-12"}'
```

## SQL Server（本机 Windows，可选）

CI 和这台云端构建保持 **memory**。要在 Windows 开发机连 SQL Server 时：

1. 不要把主机、账号、口令写进 Git、拍板文档或 `config.example.yaml`。
2. 在 `E:\B-HIS` 自建 gitignore 文件 `.env.local` 或 `config.local.yaml`。
3. 库名 **BHIS**。按顺序执行 `001_identity.up.sql`、`002_patient.up.sql`、`003_register.up.sql`、`003_register_seed.up.sql`（见 `migrations/README.md`）。Windows 身份脚本优先用 `scripts/apply-migration.ps1`（已带 `sqlcmd -I`）。不要改官方 DDL。`sqlcmd` 的口令用环境变量 `SQLCMDPASSWORD`，不要写进脚本。
4. 启动前设置：

```text
BHIS_DB_DRIVER=sqlserver
BHIS_DB_DSN=sqlserver://<user>:<password>@<host>:1433?database=BHIS&encrypt=disable
```

等价 ADO 形状：`server=<host>;user id=<user>;password=<password>;database=BHIS;encrypt=disable`

进程日志只记 `driver=sqlserver`，不记 DSN。驱动报错会先抹掉 URL 用户信息和 `password=`。

## 日志

JSON Lines：`./logs/h-api-YYYYMMDD.log`（日期按 Asia/Shanghai）。字段含 `timestamp`、`level`、`request_id`、`actor_id`、`action`、`outcome`、`error_code`、`duration_ms`。

登录审计动作：`LOGIN_SUCCESS`、`LOGIN_FAILURE`、`LOGOUT`。建档审计动作：`patient.create`（`resource_type=Patient`）。挂号审计动作：`register.create`、`register.cancel`（`resource_type=Register`）。不明文记密码、证件号或 DSN。

`logs/` 不进 Git。

## 测试与构建

```bash
go test ./...
go build -o bin/h-api ./cmd/api
go build -o bin/h-host ./cmd/host
```

Windows 演示包（先构建 `web` 的 demo 静态文件，再打进 exe）见上一节 `scripts/build-windows.ps1`。只执行 `go build -o bin/b-his-host.exe ./cmd/host` 时，用的是仓库里已提交的 `internal/webui/assets`。改了 `web/src` 之后要重新跑脚本，否则 exe 仍是旧页面。

## 布局

`cmd/api`、`cmd/host`、`internal/domain`、`internal/app`、`internal/port`、`internal/adapter/http`、`internal/store/memory`、`internal/store/mssql`、`internal/logging`、`internal/webui`（已构建的登录页）、`api/openapi.yaml`、`migrations/sqlserver/core`、`docs/contracts`、`web/`、`scripts/build-windows.ps1`。
