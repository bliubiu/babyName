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
      </head>
      <body className="min-h-screen bg-warm-white font-sans">
        <div className="fixed inset-0 bg-texture bg-xiangyun pointer-events-none animate-pattern-fade" />
        <ErrorBoundary>
          <Providers>
            {children}
          </Providers>
        </ErrorBoundary>
      </body>
    </html>
  );
}
