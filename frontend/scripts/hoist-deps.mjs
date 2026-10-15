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

// --- 第二步：按 semver 就近补链 ---
//
// 背景：pnpm 正常装配时，会在 .pnpm/<pkg>@<ver>/node_modules/<dep> 建依赖链接。
// 本机符号链接建不起来（见文件头），这些链接全部缺失；包只能靠向上查找
// 兜底，而向上找到的顶层 node_modules 里同名包只有**一个**版本（上面取最高版本）。
// 于是「同名多版本」的需求直接爆掉，例如：
//   - eslint-plugin-jsx-a11y@6.10.2 依赖 minimatch ^3.1.2（用 default 导出当函数）
//   - @eslint/config-array@0.23.5 依赖 minimatch ^10.2.4（用 braceExpand 具名导出）
// 顶层只有 10.2.5，两者必有一个被喂错版本：
//   - 喂 3 给 config-array → `TypeError: expand is not a function`（ESLint 启动即崩）
//   - 喂 10 给 jsx-a11y    → `TypeError: (0 , _minimatch.default) is not a function`
//
// 修法：为每个包按其 package.json 里声明的依赖范围，就近补上指向**满足该范围**
// 的已安装版本的 junction。范围判断只取主版本（`^3.1.2` → 3、`~4.0.1` → 4），
// 同一主版本内取已安装的最高版本：对真实依赖范围足够，且不引入 semver 依赖。

/** 从范围串里取主版本：`^10.2.4` → 10；`>=3` → 3；解析不出返回 null */
function majorFromRange(range) {
  const m = /(\d+)/.exec(range);
  return m ? Number(m[1]) : null;
}

// 按包名索引所有已安装版本（来源为顶层平铺目录）
const installed = new Map(); // name → [{ version, source }]
for (const name of candidates.keys()) {
  const dest = path.join(root, name);
  if (!fs.existsSync(dest)) continue;
  let real;
  try {
    real = fs.realpathSync(dest);
  } catch {
    continue;
  }
  let version = '0.0.0';
  try {
    version = JSON.parse(fs.readFileSync(path.join(real, 'package.json'), 'utf8')).version ?? '0.0.0';
  } catch {
    // 读不到 package.json 就退回目录名推断
  }
  installed.set(name, [{ version, source: real }]);
}

// 直接从 .pnpm 目录名重建多版本索引（顶层只留了一个版本，这里要全量）
for (const dirName of fs.readdirSync(store)) {
  const pkgRoot = path.join(store, dirName, 'node_modules');
  if (!fs.existsSync(pkgRoot)) continue;
  for (const entry of fs.readdirSync(pkgRoot, { withFileTypes: true })) {
    if (!entry.isDirectory() || entry.name === '.bin') continue;
    if (entry.name.startsWith('@')) {
      for (const sub of fs.readdirSync(path.join(pkgRoot, entry.name), { withFileTypes: true })) {
        if (!sub.isDirectory()) continue;
        record(`${entry.name}/${sub.name}`, versionFromDirName(dirName), path.join(pkgRoot, entry.name, sub.name));
      }
    } else {
      record(entry.name, versionFromDirName(dirName), path.join(pkgRoot, entry.name));
    }
  }
}

function record(name, version, source) {
  if (!installed.has(name)) installed.set(name, []);
  const list = installed.get(name);
  // 同一版本只留第一个（顶层平铺那份优先，realpath 更短更稳）
  if (!list.some(c => c.version === version)) list.push({ version, source });
}

/** 选出满足范围的最高版本；主版本对不上则返回 null */
function pickVersion(list, range) {
  if (!range || range === '*' || range === 'latest') {
    return list.reduce((a, b) => (compareVersion(b.version, a.version) > 0 ? b : a));
  }
  const wantMajor = majorFromRange(range);
  if (wantMajor === null) return null;
  const sameMajor = list.filter(c => c.version.split('.')[0] === String(wantMajor));
  if (sameMajor.length === 0) return null;
  return sameMajor.reduce((a, b) => (compareVersion(b.version, a.version) > 0 ? b : a));
}

let linked = 0;
let conflict = 0;

for (const dirName of fs.readdirSync(store)) {
  const pkgRoot = path.join(store, dirName, 'node_modules');
  if (!fs.existsSync(pkgRoot)) continue;

  // 找出这个 .pnpm 条目代表的那个包（目录里只有一个真实包目录）
  let selfDir = null;
  for (const entry of fs.readdirSync(pkgRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    if (entry.name.startsWith('@')) {
      for (const sub of fs.readdirSync(path.join(pkgRoot, entry.name), { withFileTypes: true })) {
        if (sub.isDirectory()) selfDir = path.join(pkgRoot, entry.name, sub.name);
      }
    } else if (entry.name !== '.bin') {
      selfDir = path.join(pkgRoot, entry.name);
    }
  }
  if (!selfDir) continue;

  let manifest;
  try {
    manifest = JSON.parse(fs.readFileSync(path.join(selfDir, 'package.json'), 'utf8'));
  } catch {
    continue;
  }
  const deps = { ...manifest.dependencies };
  if (!deps || Object.keys(deps).length === 0) continue;

  for (const [depName, range] of Object.entries(deps)) {
    const dest = path.join(pkgRoot, depName);
    if (fs.existsSync(dest)) continue; // pnpm 建成（或上一轮已补）

    const list = installed.get(depName);
    if (!list) continue; // 该依赖压根没装（可选依赖等），不管
    const pick = pickVersion(list, range);
    if (!pick) {
      conflict++;
      continue;
    }
    fs.mkdirSync(path.dirname(dest), { recursive: true });
    try {
      fs.symlinkSync(pick.source, dest, 'junction');
      linked++;
    } catch (err) {
      console.warn(`[hoist] 补链失败 ${depName}@${range} → ${dirName}: ${err.message}`);
    }
  }
}

console.log(`[hoist] 就近补链完成：新建 ${linked} 个 junction，主版本无匹配跳过 ${conflict} 个`);
