import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

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
  plugins: [vue()],
  server: { port: 5176, proxy },
})
