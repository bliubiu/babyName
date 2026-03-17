import type { Metadata } from "next";
import "./globals.css";
import { ToastProvider } from "@/components/Toast";

export const metadata: Metadata = {
  title: "宝宝起名大师 - 用东方智慧为宝宝选个好名字",
  description: "基于中国传统玄学（八字五行、易经64卦、生肖属相）的智能起名服务",
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
      </head>
      <body className="min-h-screen bg-warm-white">
        <div className="fixed inset-0 bg-texture pointer-events-none" />
        <ToastProvider>
          {children}
        </ToastProvider>
      </body>
    </html>
  );
}
