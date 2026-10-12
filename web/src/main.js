// 2026-10-12 P1-H05 登录界面
import { createApp } from 'vue'
import App from './App.vue'
import router from './router/index.js'
import { restoreSession } from './auth/session.js'
import './styles.css'

restoreSession()

createApp(App).use(router).mount('#app')
