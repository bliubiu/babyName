import { defineConfig } from '@playwright/test';
import { existsSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

// package.json 为 type:module，Playwright 以 ESM 加载本配置（无 __dirname）
const here = path.dirname(fileURLToPath(import.meta.url));

// 桌面端 e2e：Playwright 驱动 wails dev 的浏览器通道（dev server 对外部
// 浏览器注入 WebSocket IPC，bindings 与原生窗口同源分发，docs/47 S5）。
//
// 运行：pnpm e2e（webServer 自动拉起 wails dev；reuseExistingServer 支持
// 开发者手动 wails dev 后复用）。
//
// 前提：wails CLI 锁定 2.17.0——dev 启动时 CLI 会把 go.mod 重写为自身版本
// （2.17.1 → go 1.26，docs/47 第 2 节）。下面按 显式 env > 固定安装位 >
// PATH 顺序探测二进制。
function resolveWailsCommand(): string {
  const explicit = process.env.NAMER_WAILS;
  if (explicit) return `${explicit} dev -assetdir ../frontend/dist`;
  const pinned = path.join(os.homedir(), 'tools', 'wails-2.17.0', 'wails.exe');
  if (existsSync(pinned)) return `${pinned} dev -assetdir ../frontend/dist`;
  return 'wails dev -assetdir ../frontend/dist';
}

export default defineConfig({
  testDir: './e2e',
  timeout: 180_000, // 生成链路秒级~十秒级，放宽到 3 分钟
  expect: { timeout: 30_000 },
  fullyParallel: false,
  workers: 1, // 共享单实例后端（收藏/历史跨用例互扰），强制串行
  reporter: [['list']],
  use: {
    baseURL: 'http://localhost:34115',
    trace: 'retain-on-failure',
  },
  webServer: {
    command: resolveWailsCommand(),
    url: 'http://localhost:34115',
    reuseExistingServer: true,
    timeout: 300_000, // 首跑含前端安装/构建 + 应用编译 + 物料播种
    cwd: path.join(here, '../backend'),
    env: {
      ...process.env,
      // 隔离库：不污染仓库根 namer.db；固定路径跨次复用 → 二次运行指纹短路快启
      NAMER_DB_PATH: path.join(os.tmpdir(), 'namer-e2e', 'namer.db'),
    },
  },
});
