/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'export',
  // docs/45 P1-31：构建中间目录必须与静态导出目录分离。
  // 旧值 'out' 会把 .next 产物（dev/、server/、build/、logs/、trace/、
  // *-manifest.json 等）写进交付目录，而 nginx root 正是 out/，
  // 构建日志与服务端代码会被公网读取。
  // 改回 Next 默认 .next：编译中间产物留在 .next/（已被 .gitignore 与
  // .dockerignore 忽略），output:'export' 只把纯静态交付物写入 out/。
  // 联动：tsconfig.json 的 include、next-env.d.ts 的 routes.d.ts 引用
  // 均由 next build 按 distDir 自动重写，勿再手工指回 out/。
  distDir: '.next',
  // Next.js 16 会拦截跨源 dev 资源：E2E 浏览器引擎经 127.0.0.1 打开页面时，
  // next dev 自我标识为 localhost，不加白名单会"渲染但不水合"（React 事件全部失效）。
  allowedDevOrigins: ['127.0.0.1'],
  images: {
    unoptimized: true,
  },
  compiler: {
    // 生产剥除 console 但保留 error/warn：ErrorBoundary 与 app/error
    // 的线上日志全靠 console.error，全剥掉后前端故障无从排查（docs/29 P3）
    removeConsole: process.env.NODE_ENV === 'production' ? { exclude: ['error', 'warn'] } : false,
  },
};

export default nextConfig;
