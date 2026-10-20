/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'export',
  distDir: 'out',
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
