import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Providers } from "@/components/Providers";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Noto_Sans_SC, Noto_Serif_SC } from "next/font/google";

// 思源黑体（正文）— 构建时自动下载 Google Fonts WOFF2 文件
const notoSansSC = Noto_Sans_SC({
  subsets: ["latin"],
  weight: ["400", "500", "700"],
  variable: "--font-noto-sans-sc",
  display: "swap",
  fallback: ["PingFang SC", "Microsoft YaHei", "Hiragino Sans GB", "sans-serif"],
});

// 思源宋体（标题）— 构建时自动下载 Google Fonts WOFF2 文件
const notoSerifSC = Noto_Serif_SC({
  subsets: ["latin"],
  weight: ["400", "700"],
  variable: "--font-noto-serif-sc",
  display: "swap",
  fallback: ["STSong", "SimSun", "Songti SC", "serif"],
});

export const metadata: Metadata = {
  title: "起名 - 用东方智慧为宝宝选个好名字",
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
    <html
      lang="zh-CN"
      className={`${notoSansSC.variable} ${notoSerifSC.variable}`}
    >
      <head>
        <meta name="theme-color" content="#8B2323" />
        <meta name="description" content="基于中国传统玄学（八字五行、易经64卦、生肖属相）的智能起名服务" />
      </head>
      <body className={`min-h-screen bg-warm-white font-sans ${notoSansSC.variable} ${notoSerifSC.variable}`}>
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
