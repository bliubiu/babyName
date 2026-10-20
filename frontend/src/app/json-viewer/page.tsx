'use client';

import Navigation from '@/components/Navigation';

// JSON 美化查看器宿主页
//
// 查看器本体是自包含的原生实现（public/json4matter.html），自带全局 body 样式，
// 若直接挂载会污染本应用布局，故用 iframe 承载实现样式隔离；其逻辑与离线能力不变。
// 路由入口统一收到主导航「JSON」按钮，随前端静态导出（out/）一起发布。
export default function JsonViewerPage() {
  return (
    <div className="flex flex-col min-h-screen cloud-bg">
      <Navigation />
      <main className="flex-1 min-h-0 px-3 md:px-6 py-4 flex flex-col">
        <div className="flex-1 min-h-0 rounded-2xl border border-paper/40 overflow-hidden shadow-sm bg-warm-white-light">
          <iframe
            src="/json4matter.html"
            title="JSON 美化查看器"
            className="w-full h-full border-0"
          />
        </div>
      </main>
    </div>
  );
}
