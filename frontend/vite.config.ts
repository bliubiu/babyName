import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// Wails 桌面端前端构建配置：
// - dev 固定 34115 端口（wails dev 经 frontend:dev:serverUrl 反向代理窗口资源）
// - build 产物输出 dist/，wails build 会自动同步进 backend/frontend/dist 作 embed
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 34115,
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
