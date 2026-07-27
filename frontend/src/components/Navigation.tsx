'use client';

import { useRouter } from 'next/navigation';
import ThemeToggle from './ThemeToggle';
import { IconArrowLeft, IconHistory, IconHeart, IconCalendar, IconCamera, IconFileText } from './Icons';

interface NavigationProps {
  showBackButton?: boolean;
  showHistory?: boolean;
  showFavorites?: boolean;
  showExport?: boolean;
  onExport?: () => void;
  onExportPDF?: () => void;
}

export default function Navigation({
  showBackButton = true,
  showHistory = true,
  showFavorites = true,
  showExport = false,
  onExport,
  onExportPDF,
}: NavigationProps) {
  const router = useRouter();

  return (
    <div className="red-ribbon rounded-none mb-4 md:mb-6 animate-fade-in-up">
      <div className="pt-3" />
      <nav aria-label="主导航" className="flex flex-wrap justify-between items-center gap-3">
      {showBackButton && (
        <button
          onClick={() => router.push('/')}
          className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 text-sm md:text-base hover-lift group"
          aria-label="返回首页"
        >
          <IconArrowLeft className="transition-transform group-hover:-translate-x-0.5" size={16} />
          <span>返回首页</span>
        </button>
      )}

      <div className="flex items-center gap-1 md:gap-2 ml-auto">
        {showHistory && (
          <button
            onClick={() => router.push('/history')}
            className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2.5 py-1.5 rounded-lg hover:bg-crimson/5 text-sm"
            aria-label="查看历史记录"
          >
            <IconHistory size={16} />
            <span className="hidden sm:inline">历史</span>
          </button>
        )}
        {showFavorites && (
          <button
            onClick={() => router.push('/favorites')}
            className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2.5 py-1.5 rounded-lg hover:bg-crimson/5 text-sm"
            aria-label="查看我的收藏"
          >
            <IconHeart size={16} />
            <span className="hidden sm:inline">收藏</span>
          </button>
        )}
        <button
          onClick={() => router.push('/huangli')}
          className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2.5 py-1.5 rounded-lg hover:bg-crimson/5 text-sm"
          aria-label="查看今日黄历"
        >
          <IconCalendar size={16} />
          <span className="hidden sm:inline">黄历</span>
        </button>
        {showExport && onExport && (
          <button
            onClick={onExport}
            className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2.5 py-1.5 rounded-lg hover:bg-crimson/5 text-sm"
            aria-label="导出名字为图片"
          >
            <IconCamera size={16} />
            <span className="hidden sm:inline">导出图片</span>
          </button>
        )}
        {showExport && onExportPDF && (
          <button
            onClick={onExportPDF}
            className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2.5 py-1.5 rounded-lg hover:bg-crimson/5 text-sm"
            aria-label="导出名字为PDF"
          >
            <IconFileText size={16} />
            <span className="hidden sm:inline">导出PDF</span>
          </button>
        )}
        <div className="ml-1 pl-2 border-l border-warm-white/60">
          <ThemeToggle />
        </div>
      </div>
    </nav>
    </div>
  );
}