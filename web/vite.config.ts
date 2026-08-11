import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 启动前设置：VITE_API_GATEWAY=http://localhost:8088 npm run dev
const gateway = process.env.VITE_API_GATEWAY
const apiTarget = gateway || 'http://localhost:8093'
const useGateway = !!gateway

const proxy: Record<string, object> = {
  '/api': { target: apiTarget, changeOrigin: true },
}

if (!useGateway) {
  proxy['/iam'] = {
    target: 'http://localhost:8091',
    changeOrigin: true,
    rewrite: (path: string) => path.replace(/^\/iam/, '/api/v1'),
  }
}

export default defineConfig({
  base: process.env.VITE_BASE || '/',
  plugins: [vue(),
    {
      name: 'runtime-config-base',
      transformIndexHtml(html) {
        const baseUrl = process.env.VITE_BASE || '/'
        const tag = `<script src="${baseUrl}runtime-config.js"></script>`
        const cleaned = html.replace(/\s*<script src=["'][^"']*runtime-config\.js["']><\/script>/g, '')
        if (cleaned.includes('<head>')) {
          return cleaned.replace('<head>', `<head>\n    ${tag}`)
        }
        return `${tag}\n${cleaned}`
      },
    },
],
  server: { port: 5176, proxy },
})
