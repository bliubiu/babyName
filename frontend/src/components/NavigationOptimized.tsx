'use client';

import { useRouter, usePathname } from 'next/navigation';
import { memo, useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import NavBackButton from './NavBackButton';
import NavButton from './NavButton';
import NavMobileMenu from './NavMobileMenu';
import ThemeToggle from './ThemeToggle';

const NavExportButtons = dynamic(
  () => import('./NavExportButtons'),
  {
    loading: () => <div className="animate-pulse text-sm">加载中...</div>,
    ssr: false
  }
);

interface NavigationProps {
  showBackButton?: boolean;
  showHistory?: boolean;
  showFavorites?: boolean;
  showExport?: boolean;
  onExport?: () => void;
  onExportPDF?: () => void;
}

const NAV_BUTTONS = [
  { icon: '📜', label: '历史', path: '/history' },
  { icon: '❤️', label: '收藏', path: '/favorites' },
  { icon: '📅', label: '黄历', path: '/huangli' },
] as const;

function NavigationComponent({ 
  showBackButton = true, 
  showHistory = true, 
  showFavorites = true, 
  showExport = false, 
  onExport, 
  onExportPDF 
}: NavigationProps) {
  const router = useRouter();
  const pathname = usePathname();
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  const visibleButtons = useMemo(() => {
    return NAV_BUTTONS.filter(btn => {
      if (btn.path === '/history' && !showHistory) return false;
      if (btn.path === '/favorites' && !showFavorites) return false;
      return true;
    });
  }, [showHistory, showFavorites]);

  const handleMobileMenuToggle = () => {
    setIsMobileMenuOpen(prev => !prev);
  };

  const handleMobileMenuClose = () => {
    setIsMobileMenuOpen(false);
  };

  return (
    <>
      <nav className="flex flex-wrap justify-between items-center gap-4 mb-3 md:mb-4 lg:mb-6 animate-fadeInUp" role="navigation" aria-label="主导航">
        <div className="flex items-center gap-4">
          {showBackButton && <NavBackButton />}
        </div>

        <div className="hidden md:flex flex-wrap gap-2 md:gap-3 text-sm md:text-base">
          {visibleButtons.map((btn) => (
            <NavButton
              key={btn.path}
              icon={btn.icon}
              label={btn.label}
              onClick={() => router.push(btn.path)}
              isActive={pathname === btn.path}
            />
          ))}
          {showExport && onExport && onExportPDF && (
            <NavExportButtons onExport={onExport} onExportPDF={onExportPDF} />
          )}
          <ThemeToggle />
        </div>

        <button
          className="md:hidden p-2 text-teal hover:text-crimson transition-colors"
          onClick={handleMobileMenuToggle}
          aria-label="打开菜单"
          aria-expanded={isMobileMenuOpen}
        >
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            {isMobileMenuOpen ? (
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            ) : (
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            )}
          </svg>
        </button>
      </nav>

      <NavMobileMenu
        isOpen={isMobileMenuOpen}
        onClose={handleMobileMenuClose}
        buttons={visibleButtons}
        showExport={showExport}
        onExport={onExport}
        onExportPDF={onExportPDF}
      />
    </>
  );
}

const Navigation = memo(NavigationComponent);
Navigation.displayName = 'Navigation';

export default Navigation;
