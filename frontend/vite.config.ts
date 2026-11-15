import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// Wails 桌面端前端构建配置：
// - dev 端口 5173：仅作为 HMR 可选源（wails dev -frontenddevserverurl
//   http://localhost:5173 临时叠加）；wails dev server 独占 34115，
//   是窗口与外部浏览器（e2e）的统一入口，勿在 vite 侧占用 34115
// - build 产物输出 dist/，wails build 会自动同步进 backend/frontend/dist 作 embed
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
});
