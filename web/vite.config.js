import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

// API backend target for the dev proxy:
//   1. VITE_API_BASE_URL in web/.env (gitignored, per-environment)
//   2. VITE_API_BASE_URL from the shell
//   3. fallback: http://localhost:8080
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiBase =
    env.VITE_API_BASE_URL ||
    process.env.VITE_API_BASE_URL ||
    'http://localhost:8080'

  return {
    plugins: [react()],
    server: {
      port: 5173,
      host: true,
      proxy: {
        '/api': {
          target: apiBase,
          changeOrigin: true,
        },
      },
    },
    build: {
      outDir: 'dist',
      sourcemap: false,
    },
  }
})