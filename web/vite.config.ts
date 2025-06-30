import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/health': 'http://localhost:8080',
      '/node': 'http://localhost:8080',
      '/transactions': 'http://localhost:8080',
      '/queries': 'http://localhost:8080',
      '/rag': 'http://localhost:8080',
      '/metrics': 'http://localhost:8080',
      '/events': 'http://localhost:8080',
      '/wallet': 'http://localhost:8080',
      '/reputation': 'http://localhost:8080',
      '/consensus': 'http://localhost:8080',
      '/staking': 'http://localhost:8080',
      '/explorer': 'http://localhost:8080',
      '/accounts': 'http://localhost:8080',
      '/tokens': 'http://localhost:8080',
      '/economy': 'http://localhost:8080',
      '/nodes': 'http://localhost:8080',
      '/users': 'http://localhost:8080'
    }
  }
})
