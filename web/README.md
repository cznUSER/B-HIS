# B-HIS Web 登录

Vue 3 + Vue Router + Vite。范围只有登录页和登录后的占位首页，对接 H03 OpenAPI **0.3.2** 的 Auth：

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

口令只放在登录请求体里，不写入控制台。令牌保存在内存和 `sessionStorage`（键 `bhis.session.v1`），刷新后仍停留在首页；退出登录时清除，并调用 `POST /auth/logout`。

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
npm run check   # Mock 成功 / 失败 / 停用 / 限流 / 退出
npm run build   # 生产构建
```

## Mock 与真实 API

默认 **Mock**（仓库里的 `web/.env` 已设 `VITE_USE_MOCK=true`），不访问网络。

| 用户名 | 密码 | 结果 |
|--------|------|------|
| `admin` | `admin123` | 200，显示名「系统管理员」，角色 `SYSTEM` |
| `disabled` | `disabled123` | 403 `ACCOUNT_DISABLED` |
| 其他或密码错误 | | 401 `INVALID_CREDENTIALS`（不区分用户是否存在） |
| 同一用户名连续失败 5 次 | | 之后 429 `AUTH_RATE_LIMITED`（仅当前页面内存计数，刷新清零） |

Mock 的 `accessToken` 是三段 JWT 外形，签名固定为 `mock`，**不是**服务端 HS256，不能拿去调真实接口。`GET /auth/me` 与 `POST /auth/logout` 在 Mock 里同样可用；退出后该 token 的 `jti` 会记在 `sessionStorage`，再调 `me` 得到 `TOKEN_INVALID`。

切换到 H04 真实 API：新建 **`web/.env.local`**（已被 gitignore，勿提交口令或密钥）：

```bash
VITE_USE_MOCK=false
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

保存后重新执行 `npm run dev`。此时浏览器直连该地址，Go API 需要对开发源放开 CORS。

若希望浏览器只访问 Vite，由开发服务器转发：

```bash
VITE_USE_MOCK=false
VITE_API_BASE_URL=/api/v1
VITE_API_PROXY_TARGET=http://localhost:8080
```

`/api` 会被代理到 `VITE_API_PROXY_TARGET`。只认字符串 `false` / `0` / `no` 关闭 Mock，其他值（含未设置）仍走 Mock。

生产构建同样读取构建时的 `VITE_*`。不要把真实环境的密钥放进前端仓库。
