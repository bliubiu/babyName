// 依赖平铺脚本：把 pnpm 虚拟存储（node_modules/.pnpm）里的包平铺到顶层
// node_modules，等价于 npm 的平铺布局。
//
// 为什么需要它：
//   本机（无管理员权限、未开启 Windows 开发者模式）创建符号链接会静默失败 ——
//   os.symlink() 不抛异常，但产出一个不可用的空 junction。pnpm 默认的隔离式
//   （isolated）node_modules 完全依赖符号链接装配，于是包文件都下载到了
//   .pnpm 里，依赖之间的链接却建不起来，运行时报
//       ERR_MODULE_NOT_FOUND: Cannot find package 'std-env'
//   （vitest 首当其冲，前端因此完全跑不了自动化测试）。
//
// 为什么用 junction 而不是 symlink：
//   目录 junction 不需要 SeCreateSymbolicLinkPrivilege，普通用户即可创建，
//   Node 的 fs.symlink(type='junction') 正好支持。
//
// 为什么不用 .npmrc 的 node-linker=hoisted：
//   pnpm 11 已移除该配置项（pnpm config get node-linker → undefined），
//   写了也不生效，见 frontend/.npmrc 的历史记录。
//
// 非 Windows 平台 pnpm 装配正常，脚本直接退出。
// 本脚本挂在 package.json 的 postinstall 上，每次安装后自动执行；也可单独运行：
//   node scripts/hoist-deps.mjs

import fs from 'node:fs';
import path from 'node:path';

if (process.platform !== 'win32') {
  console.log('[hoist] 非 Windows 平台，pnpm 装配正常，跳过平铺');
  process.exit(0);
}

const root = path.resolve(process.cwd(), 'node_modules');
const store = path.join(root, '.pnpm');

if (!fs.existsSync(store)) {
  console.log('[hoist] 未发现 node_modules/.pnpm（可能已是平铺布局），跳过');
  process.exit(0);
}

/** 从 .pnpm 目录名中解析版本号：vitest@5.0.0_@types+node@20_hash → 5.0.0 */
function versionFromDirName(dirName) {
  const at = dirName.lastIndexOf('@');
  if (at < 0) return '0.0.0';
  let v = dirName.slice(at + 1);
  const u = v.indexOf('_');
  if (u >= 0) v = v.slice(0, u);
  return v;
}

/** 版本号比较，无法解析为 SemVer 时退化为字符串比较 */
function compareVersion(a, b) {
  const pa = a.split('.').map(Number);
  const pb = b.split('.').map(Number);
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const x = Number.isFinite(pa[i]) ? pa[i] : -1;
    const y = Number.isFinite(pb[i]) ? pb[i] : -1;
    if (x !== y) return x - y;
  }
  return 0;
}

// 收集候选：包名 → [{ version, source }]
const candidates = new Map();
function addCandidate(name, version, source) {
  if (!candidates.has(name)) candidates.set(name, []);
  candidates.get(name).push({ version, source });
}

for (const dirName of fs.readdirSync(store)) {
  const pkgRoot = path.join(store, dirName, 'node_modules');
  if (!fs.existsSync(pkgRoot)) continue;
  for (const entry of fs.readdirSync(pkgRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const entryPath = path.join(pkgRoot, entry.name);
    // 带作用域的包：@scope 目录下才是真正的包
    if (entry.name.startsWith('@')) {
      for (const sub of fs.readdirSync(entryPath, { withFileTypes: true })) {
        if (!sub.isDirectory()) continue;
        addCandidate(
          `${entry.name}/${sub.name}`,
          versionFromDirName(dirName),
          path.join(entryPath, sub.name)
        );
      }
    } else if (entry.name !== '.bin') {
      addCandidate(entry.name, versionFromDirName(dirName), entryPath);
    }
  }
}

let created = 0;
let skipped = 0;
let multi = 0;

for (const [name, list] of [...candidates.entries()].sort((a, b) => a[0].localeCompare(b[0]))) {
  // 多版本时取最高版本，与 npm 平铺时"先到先得"略有差异但更合理
  const best = list.reduce((acc, c) => (compareVersion(c.version, acc.version) > 0 ? c : acc), list[0]);
  if (list.length > 1) multi++;

  const dest = path.join(root, name);
  if (fs.existsSync(dest)) {
    skipped++;
    continue;
  }
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  try {
    fs.symlinkSync(best.source, dest, 'junction');
    created++;
  } catch (err) {
    console.warn(`[hoist] 平铺失败 ${name}: ${err.message}`);
  }
}

console.log(`[hoist] 平铺完成：新建 ${created} 个，已存在 ${skipped} 个，多版本 ${multi} 个`);
