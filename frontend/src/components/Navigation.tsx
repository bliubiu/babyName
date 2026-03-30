'use client';

import { useRouter } from 'next/navigation';
import ThemeToggle from './ThemeToggle';

interface NavigationProps {
  showBackButton?: boolean;
  showHistory?: boolean;
  showFavorites?: boolean;
  showExport?: boolean;
  onExport?: () => void;
  onExportPDF?: () => void;
}

export default function Navigation({ showBackButton = true, showHistory = true, showFavorites = true, showExport = false, onExport, onExportPDF }: NavigationProps) {
  const router = useRouter();

  return (
    <nav aria-label="主导航" className="flex flex-wrap justify-between items-center gap-4 mb-3 md:mb-4 lg:mb-6 animate-fadeInUp">
      {showBackButton && (
        <button
          onClick={() => router.push('/')}
          className="flex items-center gap-2 text-teal hover:text-crimson transition-all duration-300 text-sm md:text-base hover-lift"
          aria-label="返回首页"
        >
          ← 返回首页
        </button>
      )}
      <div className="flex flex-wrap gap-2 md:gap-3 text-sm md:text-base">
        {showHistory && (
          <button
            onClick={() => router.push('/history')}
            className="text-teal hover:text-crimson transition-all duration-300 px-2 py-1 rounded-md hover:bg-crimson/10 hover-lift"
            aria-label="查看历史记录"
          >
            📜 历史
          </button>
        )}
        {showFavorites && (
          <button
            onClick={() => router.push('/favorites')}
            className="text-teal hover:text-crimson transition-all duration-300 px-2 py-1 rounded-md hover:bg-crimson/10 hover-lift"
            aria-label="查看我的收藏"
          >
            ❤️ 收藏
          </button>
        )}
        <button
          onClick={() => router.push('/huangli')}
          className="text-teal hover:text-crimson transition-all duration-300 px-2 py-1 rounded-md hover:bg-crimson/10 hover-lift"
          aria-label="查看今日黄历"
        >
          📅 黄历
        </button>
        {showExport && onExport && (
          <button
            onClick={onExport}
            className="text-teal hover:text-crimson transition-all duration-300 px-2 py-1 rounded-md hover:bg-crimson/10 hover-lift"
            aria-label="导出名字为图片"
          >
            📷 导出图片
          </button>
        )}
        {showExport && onExportPDF && (
          <button
            onClick={onExportPDF}
            className="text-teal hover:text-crimson transition-all duration-300 px-2 py-1 rounded-md hover:bg-crimson/10 hover-lift"
            aria-label="导出名字为PDF"
          >
            📄 导出PDF
          </button>
        )}
        <ThemeToggle />
      </div>
    </nav>
  );
}
