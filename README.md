# B-HIS

医院信息系统（H Phase1）。Go 登录 API 在仓库根目录；Vue 登录界面在 `web/`。

门诊一期当前实现 **登录鉴权**（H03 契约 v0.3.2）和 `GET /health`。OpenAPI 里的门诊路径是契约草案，进程不注册这些路由。

H03 `0.3.0-draft` 没有 Auth。`api/openapi.yaml` 已换成 **0.3.2-draft**（含 `POST /auth/login`、`POST /auth/logout`、`GET /auth/me`）。成功体按 OpenAPI：`LoginResponse` / `UserDto`，不包 `success/data`。失败体按《接口契约》§1.2：`success: false`、`data: null`、`error.code`、`requestId`。

## Web 登录界面

```bash
cd web
npm install
npm run dev
```

也可用 pnpm：`pnpm install` 然后 `pnpm dev`。浏览器打开终端里提示的本地地址（默认 http://localhost:5173）。

默认是 **Mock**（`VITE_USE_MOCK=true`），不访问后端。Mock 演示账号 `admin` / `admin123`。

切换到 H04 真实 API、接口路径和错误码说明见 [web/README.md](web/README.md)。联调时先按下一节启动 Go API（默认 `127.0.0.1:8080`）。真实 API 的内存演示账号是 `admin` / `Admin@123`，和 Mock 口令不同。

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

进程环境变量优先于 YAML。若变量尚未设置，会再读当前目录下 gitignore 的 `.env.local`，然后 `.env`。

也可以：

```bash
go run ./cmd/host -config config.example.yaml start
go run ./cmd/host -config config.example.yaml health
go run ./cmd/host -config config.example.yaml logs
```

Host 只是启停 / 健康 / 日志目录的薄封装。验收可以只跑 `cmd/api`。托盘窗不在这个构建里。

## 内存演示账号

用户名区分大小写。密码只以 bcrypt 保存。

| 用户名 | 密码 | 角色 | 说明 |
|--------|------|------|------|
| `admin` | `Admin@123` | `SYSTEM` | 可登录 Go API |
| `disabled` | 同上 | 无 | 密码正确时 `403 ACCOUNT_DISABLED`；密码错误仍是 `401 INVALID_CREDENTIALS` |

请在正式环境改掉演示口令。这不是 SQL Server 的登录口令。Vue Mock 使用另一套口令 `admin` / `admin123`，见上文。

## curl

登录 `200`，体为 `LoginResponse`（`expiresIn` 默认 `28800`）：

```bash
curl -sS -D - http://127.0.0.1:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin@123"}'
```

```json
{
  "accessToken": "<jwt>",
  "tokenType": "Bearer",
  "expiresIn": 28800,
  "user": {
    "id": "00000000-0000-4000-8000-000000000001",
    "username": "admin",
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
  -d '{"username":"admin","password":"wrong"}'
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

## SQL Server（本机 Windows，可选）

CI 和这台云端构建保持 **memory**。要在 Windows 开发机连 SQL Server 时：

1. 不要把主机、账号、口令写进 Git、拍板文档或 `config.example.yaml`。
2. 在 `E:\B-HIS` 自建 gitignore 文件 `.env.local` 或 `config.local.yaml`。
3. 库名 **BHIS**。先执行 `migrations/sqlserver/core/001_identity.up.sql`（见 `migrations/README.md`）。`sqlcmd` 的口令用环境变量 `SQLCMDPASSWORD`，不要写进脚本。
4. 启动前设置：

```text
BHIS_DB_DRIVER=sqlserver
BHIS_DB_DSN=sqlserver://<user>:<password>@<host>:1433?database=BHIS&encrypt=disable
```

等价 ADO 形状：`server=<host>;user id=<user>;password=<password>;database=BHIS;encrypt=disable`

进程日志只记 `driver=sqlserver`，不记 DSN。驱动报错会先抹掉 URL 用户信息和 `password=`。

## 日志

JSON Lines：`./logs/h-api-YYYYMMDD.log`（日期按 Asia/Shanghai）。字段含 `timestamp`、`level`、`request_id`、`actor_id`、`action`、`outcome`、`error_code`、`duration_ms`。

登录审计动作：`LOGIN_SUCCESS`、`LOGIN_FAILURE`、`LOGOUT`。不明文记密码，不记 DSN。

`logs/` 不进 Git。

## 测试与构建

```bash
go test ./...
go build -o bin/h-api ./cmd/api
go build -o bin/h-host ./cmd/host
```

## 布局

`cmd/api`、`cmd/host`、`internal/domain`、`internal/app`、`internal/port`、`internal/adapter/http`、`internal/store/memory`、`internal/store/mssql`、`internal/logging`、`api/openapi.yaml`、`migrations/sqlserver/core`、`docs/contracts`、`web/`。
