import { defineConfig } from '@playwright/test';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

// package.json 为 type:module，Playwright 以 ESM 加载本配置（无 __dirname）
const here = path.dirname(fileURLToPath(import.meta.url));
const backendDir = path.join(here, '../backend');
const baseURL = 'http://127.0.0.1:8080';

// 桌面端 e2e：Wails v3 的 server build（docs/48 阶段 3）。
// `go run -tags server,production .` 不创建原生窗口，由 Go 直接提供静态资源
// 与 /wails/runtime IPC（与桌面 WebView 同一套 HTTPTransport），Playwright
// 用标准 Chromium 直连即可；v2 时代 `wails dev :34115` 浏览器通道的
// `runtime:ready` 握手噪声不复存在。
//
// 运行：pnpm e2e（脚本已前置 pnpm build，后者同步 frontend/dist 到
// backend/frontend/dist 供 //go:embed；改 src 不 build 会假通过）。
//
// 官方边界：server build 不覆盖原生窗口/菜单/托盘/原生对话框，viewport
// 也不等于原生窗口尺寸；本项目 23 例均为页面级用例，可平移。
export default defineConfig({
  testDir: './e2e',
  timeout: 180_000, // 生成链路秒级~十秒级，放宽到 3 分钟
  expect: { timeout: 30_000 },
  fullyParallel: false,
  workers: 1, // 共享单实例后端（收藏/历史跨用例互扰），强制串行
  reporter: [['list']],
  use: {
    baseURL,
    trace: 'retain-on-failure',
  },
  webServer: {
    command: 'go run -tags server,production .',
    url: baseURL,
    reuseExistingServer: true,
    timeout: 300_000, // 首跑含 Go 编译 + 物料播种
    cwd: backendDir,
    env: {
      ...process.env,
      // 隔离库：不污染仓库根 namer.db；固定路径跨次复用 → 二次运行指纹短路快启
      NAMER_DB_PATH: path.join(os.tmpdir(), 'namer-e2e', 'namer.db'),
    },
  },
});
