# B-HIS Web 登录

Vue 3 + Vue Router + Vite。登录页对接 H03 OpenAPI **0.3.2** 的 Auth；登录成功进入首页看板 `/app`（侧栏进入门诊六步，模块页为空态）：

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/auth/login` | 匿名。请求 `LoginRequest` `{ username, password }`，成功 `LoginResponse` |
| `POST` | `/api/v1/auth/logout` | `Authorization: Bearer <accessToken>`。成功 `204` 或 `200 { "success": true }` |
| `GET` | `/api/v1/auth/me` | 同上 Bearer。成功 `UserDto` |

`LoginResponse` 字段：`accessToken`（JWT）、`tokenType: "Bearer"`、`expiresIn`（秒，默认 28800）、`user`（`id`、`username`、`displayName`、`roles`）。

失败使用 `ErrorBody`：`{ success, error: { code, message, details }, requestId }`。客户端也接受多一个 `data` 字段的包络，以及没有 JSON 的 HTTP 错误。

| HTTP | code | 界面文案（服务端未给 message 时） |
|------|------|-----------------------------------|
| 401 | `INVALID_CREDENTIALS` | 用户名或密码错误 |
| 403 | `ACCOUNT_DISABLED` | 账号已停用 |
| 429 | `AUTH_RATE_LIMITED` | 登录尝试过于频繁，请稍后再试 |
| 401 | `UNAUTHORIZED` / `TOKEN_EXPIRED` / `TOKEN_INVALID` | 按码提示重新登录 |

口令只放在登录请求体里，不写入控制台。令牌保存在内存和 `sessionStorage`（键 `bhis.session.v1`），刷新后仍停留在首页 `/app`；退出登录时清除，并调用 `POST /auth/logout`。

## 登录后的壳

本地预览：

```bash
cd web
npm install
npm run dev
```

浏览器打开 http://localhost:5173 。未登录访问 `/` 或 `/app` 会到登录页；登录成功进入 `/app` 首页看板。

顶栏：产品名 B-HIS（回到首页）、环境标签（Mock 为「演示」，真实 API 为「开发」）、机构「演示机构」（占位，没有机构码）、人员显示名和角色、通知铃铛（角标固定 0，占位）、用户菜单、「退出登录」、本机时钟。用户菜单里「个人设置」为二期，不能点。显示名和角色来自登录结果或 `GET /auth/me`，页面不写死账号。没有患者搜索框。

左侧栏：首页；档案管理（患者建档进入 `/app/outpatient/patient`，就诊卡/账户为二期灰项）；门诊业务下建档、挂号、开单、收费、发药、退药/退费，以及一条「六步总览」；系统设置里的用户与权限、基础字典都是二期灰项，不打开空白页。无权限的门诊项灰显，原因是「无权限」。没有住院、病历、报表、采购这些一级菜单。

首页主区是工作台：待办一行（待收费、待发药为 0，可进入对应空态；进销存未上传、未贯标提醒为二期，不能点；库存预警为 0 的弱占位）、经营概况（今天 / 昨天 / 本周 / 本月、日期按钮二期、刷新）、收费行和客流行、两张等分圆环（收入来源、支付方式，图例都是 0）。页上标明「演示 / 占位」。切换时间和刷新不会改数字，也不请求经营接口。

`/app/outpatient` 仍是六张卡的总览，不是登录落地页。顺序不变：患者建档、挂号、开单、收费、发药、退药/退费。门诊接口还没接入，徽标是「开发中」；有权限时「进入」打开空态「接口就绪后开放」，没有假列表。无权限的卡保持原位、灰显。顶上有一条可关闭的闭环提示。

## 本地运行

在 `web/` 目录：

```bash
npm install
npm run dev
```

或：

```bash
pnpm install
pnpm dev
```

浏览器打开 http://localhost:5173 。提交时按钮禁用并显示「登录中…」；再次提交会清掉上一条错误。

```bash
npm run check       # Mock、工作台入口、以及本机假 HTTP（ErrorBody / 204）。不访问 Go API
npm run check:api   # 仅当 VITE_USE_MOCK=false 时打真 API；否则跳过。CI 不跑
npm run build       # 生产构建
```

## Mock

默认 **Mock**（仓库里的 `web/.env` 已设 `VITE_USE_MOCK=true`），不访问网络。

| 用户名 | 密码 | 结果 |
|--------|------|------|
| `system` | `1` | 200，显示名「系统管理员」，角色 `SYSTEM` |
| `disabled` | `1` | 403 `ACCOUNT_DISABLED` |
| 其他或密码错误 | | 401 `INVALID_CREDENTIALS`（不区分用户是否存在） |
| 同一用户名连续失败 5 次 | | 之后 429 `AUTH_RATE_LIMITED`（仅当前页面内存计数，刷新清零） |

Mock 的 `accessToken` 是三段 JWT 外形，签名固定为 `mock`，**不是**服务端 HS256，不能拿去调真实接口。`GET /auth/me` 与 `POST /auth/logout` 在 Mock 里同样可用；退出后该 token 的 `jti` 会记在 `sessionStorage`，再调 `me` 得到 `TOKEN_INVALID`。

只认字符串 `false` / `0` / `no` 关闭 Mock，其他值（含未设置）仍走 Mock。生产构建同样读取构建时的 `VITE_*`。

## 本机真 API 联调

Mock 关掉之后，页面不再出现「本地演示模式」，改为「本机 API 账号」`system` / `1`。登录口令只在表单里输入，不要写进 `web/.env.local`。员工默认密码是 `1`。

### 1. 启动 Go API

在**仓库根目录**（不是 `web/`）。进程默认监听 **`127.0.0.1:8080`**（`config.example.yaml` 的 `Host.ApiAddr`，与 `internal/config` 的 Defaults 相同）。契约前缀是 `/api/v1`。

`go run ./cmd/api` 会再找 `BHIS_CONFIG`、`config.local.yaml`、`config.yaml`，以及 gitignore 的根目录 `.env.local` / `.env`。JWT 为空会直接退出。本机演示用仓库里的占位配置即可（日志会警告密钥是占位符；共享环境改用 `BHIS_JWT_SECRET`，不要把真实密钥提交进 Git）：

```bash
go run ./cmd/api -config config.example.yaml
```

已设置 `BHIS_JWT_SECRET`，或根目录已有 gitignore 的本地配置时，可以不带 `-config`：

```bash
go run ./cmd/api
```

确认端口：

```bash
curl -sS http://127.0.0.1:8080/health
```

应看到 `"status":"ok"` 且 `"driver":"memory"`（未改数据库驱动时）。监听地址若被 `BHIS_API_ADDR` 改掉，下面的 `VITE_API_BASE_URL` 要跟着改。

内存演示账号以仓库根 [README](../README.md)「内存演示账号」为准，和 Mock 是同一套口令。员工默认密码是 `1`。

| 用户名 | 密码 | 结果 |
|--------|------|------|
| `system` | `1` | 200，显示名「系统管理员」，角色 `SYSTEM` |
| `disabled` | `1` | 密码正确时 403 `ACCOUNT_DISABLED`；密码错误仍是 401 `INVALID_CREDENTIALS` |

同一 IP 或用户名失败次数默认 **8 次 / 15 分钟** 后 429 `AUTH_RATE_LIMITED`（`config.example.yaml` 的 `Auth.RateLimit`）。这和 Mock 的「连续 5 次、刷新清零」不同。正式环境请改掉演示口令。这不是 SQL Server 的登录口令，也不要写进前端仓库。

### 2. 关掉 Mock

在 `web/` 新建 **`web/.env.local`**（把 `.env.example` 复制过去最省事）。该文件已被 gitignore。**不要提交 `.env.local`，不要把口令、JWT、数据库连接串写进任何会进 Git 的文件。** 前端这几个变量里也不需要密钥。

保存后重新执行 `npm run dev`（Vite 只在启动时读环境变量）。浏览器打开 http://localhost:5173 。

**方式 A — 浏览器直连**（地址就是上面的默认端口）：

```bash
VITE_USE_MOCK=false
VITE_API_BASE_URL=http://127.0.0.1:8080/api/v1
```

页面在 `http://localhost:5173` 或 `http://127.0.0.1:5173` 时，默认 CORS 已放行（`config.example.yaml` 的 `CORS.AllowedOrigins`，另含 preview 的 `4173`）。API 对 `POST` / `GET` 带 `Content-Type`、`Authorization` 会先回 `OPTIONS` 204。用了别的主机名或端口时，设置 `BHIS_CORS_ORIGINS`（逗号分隔）后重启 API。客户端跨源请求使用 `credentials: omit`，因为服务端不回 `Access-Control-Allow-Credentials`，登录靠 `Authorization: Bearer`，不靠 Cookie。

**方式 B — Vite 代理**（浏览器只访问 5173，不走跨源）：

```bash
VITE_USE_MOCK=false
VITE_API_BASE_URL=/api/v1
VITE_API_PROXY_TARGET=http://127.0.0.1:8080
```

开发服务器把以 `/api` 开头的请求转到 `VITE_API_PROXY_TARGET`。`npm run preview`（默认 4173）没有这段代理，预览构建请用方式 A，或保证页面源在 CORS 白名单里。

### 3. 验收

| 步骤 | 期望 |
|------|------|
| 成功 | `system` / `1`。进入 `/app` 首页看板，有「经营概况」和「演示 / 占位」。顶栏显示名「系统管理员」，角色「系统」（来自 `UserDto`，不是页面写死的账号）。侧栏「门诊业务」可展开六步 |
| 失败 | 故意错口令。展示服务端 `error.message`「用户名或密码错误」，以及 `INVALID_CREDENTIALS`。未知用户同一个码，页面不说明用户是否存在 |
| 停用 | `disabled` 且口令正确。`ACCOUNT_DISABLED`，「账号已停用」。该用户口令错误仍是 `INVALID_CREDENTIALS` |
| 退出 | 工作台用户菜单「退出登录」。`POST /api/v1/auth/logout` 成功为 **204 空体**。回到登录页；刷新停在登录页。同一 token 再访问为 401 `TOKEN_INVALID` |
| 连不上 | API 没起来，或直连时页面源不在 CORS 白名单。提示「无法连接登录服务」。先看 `/health`，或改用方式 B |

失败包络与 Go API 一致：`success: false`、`data: null`、`error.code` / `error.message`、`requestId`。登录成功体是不包 `success` 的 `LoginResponse`。界面在服务端带了 `message` 时用服务端文案，并显示错误码。

可选，在 `web/` 且 API 已启动、`.env.local` 里 `VITE_USE_MOCK=false` 时：

```bash
npm run check:api
```

脚本从环境以及 `web/.env.local`、`web/.env` 读取 `VITE_USE_MOCK`、`VITE_API_BASE_URL`、`VITE_API_PROXY_TARGET`（已有环境变量优先）。它打印状态码和错误码，不打印 token 或口令。`VITE_USE_MOCK` 不是 `false` / `0` / `no` 时直接跳过并退出 0。**CI 只跑 `npm run check`，不要求本机有 API。** 相对路径的 `VITE_API_BASE_URL` 会拼到 `VITE_API_PROXY_TARGET`（缺省 `http://127.0.0.1:8080`）上，因为脚本跑在 Node 里，不经过 Vite。

## Windows 演示 exe

`npm run dev` 保持 Mock。给双击 exe 用的是另一条构建，关闭 Mock，接口走同源 `/api/v1`，登录页带仓库里的全屏背景：

```bash
npm run build:demo
```

仓库根目录的 `scripts/build-windows.ps1` 会执行这条命令，把 `web/dist` 拷进 `internal/webui/assets`，再打出 `bin/b-his-host.exe`。账号与启动说明见仓库根目录 [README.md](../README.md) 的「Windows 双击启动」。演示账号是 `system` / `1`。
