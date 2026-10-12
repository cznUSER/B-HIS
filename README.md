# B-HIS

医院信息系统（H Phase1）。后端（Go）将放在仓库根目录；登录界面在 `web/`。

## Web 登录界面

```bash
cd web
npm install
npm run dev
```

也可用 pnpm：`pnpm install` 然后 `pnpm dev`。浏览器打开终端里提示的本地地址（默认 http://localhost:5173）。

默认是 **Mock**（`VITE_USE_MOCK=true`），不访问后端。演示账号 `admin` / `admin123`。

切换到 H04 真实 API、接口路径和错误码说明见 [web/README.md](web/README.md)。
