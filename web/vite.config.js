// 2026-10-12 P1-H05 登录界面
// 2026-10-12 mode=demo：Windows exe 同源真实 API（关闭 Mock）
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ mode }) => {
  const demo = mode === 'demo'
  if (demo) {
    process.env.VITE_USE_MOCK = 'false'
    process.env.VITE_API_BASE_URL = '/api/v1'
  }
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_API_PROXY_TARGET || 'http://localhost:8080'

  return {
    plugins: [
      vue(),
      {
        name: 'bhis-demo-marker',
        transformIndexHtml(html) {
          if (!demo) return html
          return html.replace('</head>', '    <meta name="bhis-ui" content="api" />\n  </head>')
        },
      },
    ],
    server: {
      port: 5173,
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
  }
})
