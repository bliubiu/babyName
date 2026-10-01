/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'export',
  distDir: 'out',
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
