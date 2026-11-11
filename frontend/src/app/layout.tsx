import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Providers } from "@/components/Providers";
import { ErrorBoundary } from "@/components/ErrorBoundary";

// 字体：由 globals.css 的 :root --font-noto-* 变量指向系统字体栈
// （本机离线构建时 next/font/google 无法下载字体，改用系统字体无需联网）

export const metadata: Metadata = {
  title: "起名 - 为宝宝选个好名字",
  description: "基于中国传统玄学（八字五行、易经64卦、生肖属相）的智能起名服务",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  themeColor: "#8B2323",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN">
      <head>
        <meta name="theme-color" content="#8B2323" />
        <meta name="description" content="基于中国传统玄学（八字五行、易经64卦、生肖属相）的智能起名服务" />
        {/* 防闪烁（FOUC）：静态导出首屏 HTML 不含 .dark，此脚本在 <body> 渲染前
            同步执行，按 localStorage('theme') 或服务端 prefers-color-scheme 给 <html>
            挂 .dark，避免深色用户在 JS 加载完成前先看到浅色闪一下。与 ThemeToggle 共用
            同一 key 与 class 约定（docs/07 §0.3）。 */}
        <script
          dangerouslySetInnerHTML={{
            __html:
              "(function(){try{var t=localStorage.getItem('theme');var d=t==='dark'||(!t&&window.matchMedia('(prefers-color-scheme: dark)').matches);if(d){document.documentElement.classList.add('dark');}}catch(e){}})();",
          }}
        />
      </head>
    <body className="min-h-screen bg-warm-white font-sans">
      {/* 宣纸纹理 / 祥云暗纹：常驻背景层，opacity 由 CSS 控制（0.03~0.04），
          不可挂 animate-pattern-fade——该动画 100% 为 opacity:1 且 forwards，
          会把这层 fixed 全屏遮罩变成不透明，盖住所有内容（暗色下尤甚）。 */}
      <div className="fixed inset-0 bg-texture bg-xiangyun pointer-events-none" />
      {/* 内容层抬到纹理之上：relative z-10 确保文字/卡片始终在背景纹理上方 */}
      <div className="relative z-10 min-h-screen">
        <ErrorBoundary>
          <Providers>
            {children}
          </Providers>
        </ErrorBoundary>
      </div>
    </body>
    </html>
  );
}
