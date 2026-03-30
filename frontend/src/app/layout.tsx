import type { Metadata, Viewport } from "next";
import "./globals.css";
import { Providers } from "@/components/Providers";

export const metadata: Metadata = {
  title: "宝宝起名大师 - 用东方智慧为宝宝选个好名字",
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
        <link
          href="https://fonts.googleapis.com/css2?family=Noto+Sans+SC:wght@400;500;700&family=Noto+Serif+SC:wght@400;700&display=swap"
          rel="stylesheet"
        />
        <meta name="theme-color" content="#8B2323" />
        <meta name="description" content="基于中国传统玄学（八字五行、易经64卦、生肖属相）的智能起名服务" />
      </head>
      <body className="min-h-screen bg-warm-white">
        <div className="fixed inset-0 bg-texture pointer-events-none" />
        <Providers>
          {children}
        </Providers>
      </body>
    </html>
  );
}
