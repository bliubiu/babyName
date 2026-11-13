import { Route, Routes } from 'react-router-dom';
import { Providers } from '@/components/Providers';
import { NotFound } from '@/components/RoutesFallback';
import HomePage from '@/app/page';
import ResultPage from '@/app/result/page';
import EvaluatePage from '@/app/evaluate/page';
import ComparePage from '@/app/compare/page';
import FavoritesPage from '@/app/favorites/page';
import HistoryPage from '@/app/history/page';
import HuangliPage from '@/app/huangli/page';
import JsonViewerPage from '@/app/json-viewer/page';
import SharePage from '@/app/share/page';
import StatPage from '@/app/stat/page';

// 桌面端路由（HashRouter：Wails 资产服务器对未知路径返回 404，
// hash 路由可免疫 WebView F5 刷新后的 404 空白页）。
// 组件级错误兜底由 Providers 内的 ErrorBoundary 承担。
export default function App() {
  return (
    <Providers>
      <div className="fixed inset-0 bg-texture bg-xiangyun pointer-events-none" />
      <div className="relative z-10 min-h-screen">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/result" element={<ResultPage />} />
          <Route path="/evaluate" element={<EvaluatePage />} />
          <Route path="/compare" element={<ComparePage />} />
          <Route path="/favorites" element={<FavoritesPage />} />
          <Route path="/history" element={<HistoryPage />} />
          <Route path="/huangli" element={<HuangliPage />} />
          <Route path="/json-viewer" element={<JsonViewerPage />} />
          <Route path="/share" element={<SharePage />} />
          <Route path="/stat" element={<StatPage />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </div>
    </Providers>
  );
}
