// 把 Vite 产物同步到 backend/frontend/dist，供 Go 的 //go:embed 嵌入（Wails v3）。
//
// v2 时代由 `wails build` 自动同步，v3 改为显式步骤。挂到 `pnpm build` 末尾，
// 保证 `pnpm e2e`（前置 build）与 `wails3 build` 拿到的都是最新前端，
// 避免「改了 src 却仍服务旧 dist」造成 E2E 假通过（docs/47 S5 的复发坑）。
import { cpSync, existsSync, mkdirSync, readdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const src = path.join(frontendRoot, 'dist');
const dst = path.resolve(frontendRoot, '../backend/frontend/dist');

if (!existsSync(path.join(src, 'index.html'))) {
  console.error('[sync-dist] 缺少 frontend/dist/index.html，请先执行 vite build');
  process.exit(1);
}

mkdirSync(dst, { recursive: true });
for (const name of readdirSync(dst)) {
  // .gitkeep 保证仓库干净检出时 embed 目录非空（否则 go build 直接失败）
  if (name === '.gitkeep') continue;
  rmSync(path.join(dst, name), { recursive: true, force: true });
}
cpSync(src, dst, { recursive: true });
console.log(`[sync-dist] 已同步前端产物 -> ${dst}`);
