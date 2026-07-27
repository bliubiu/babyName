'use client';

import { memo, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import NavButton from './NavButton';
import NavExportButtons from './NavExportButtons';
import ThemeToggle from './ThemeToggle';

interface NavMobileMenuProps {
  isOpen: boolean;
  onClose: () => void;
  buttons: Array<{ icon: string; label: string; path: string }>;
  showExport?: boolean;
  onExport?: () => void;
  onExportPDF?: () => void;
}

function NavMobileMenuComponent({ 
  isOpen, 
  onClose, 
  buttons, 
  showExport = false, 
  onExport, 
  onExportPDF 
}: NavMobileMenuProps) {
  const router = useRouter();
  const overlayRef = useRef<HTMLDivElement>(null);
  const firstButtonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (isOpen && firstButtonRef.current) {
      firstButtonRef.current.focus();
    }
  }, [isOpen]);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === overlayRef.current) {
      onClose();
    }
  };

  const handleButtonClick = (path: string) => {
    router.push(path);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div
      ref={overlayRef}
      className="fixed inset-0 z-50 bg-black/50 transition-opacity duration-300"
      onClick={handleOverlayClick}
      role="dialog"
      aria-modal="true"
      aria-label="导航菜单"
    >
      <div className="fixed right-0 top-0 h-full w-64 max-w-full bg-white shadow-xl transform transition-transform duration-300 ease-out">
        <div className="p-6">
          <div className="flex justify-between items-center mb-6">
            <h2 className="text-lg font-bold text-ink">导航菜单</h2>
            <button
              onClick={onClose}
              className="p-2 text-jade hover:text-crimson transition-colors"
              aria-label="关闭菜单"
            >
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div className="space-y-2">
            {buttons.map((btn, index) => (
              <NavButton
                key={btn.path}
                ref={index === 0 ? firstButtonRef : undefined}
                icon={btn.icon}
                label={btn.label}
                onClick={() => handleButtonClick(btn.path)}
                className="w-full justify-start"
              />
            ))}

            {showExport && onExport && onExportPDF && (
              <div className="pt-4 border-t border-paper">
                <NavExportButtons onExport={onExport} onExportPDF={onExportPDF} />
              </div>
            )}

            <div className="pt-4 border-t border-paper">
              <ThemeToggle />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

const NavMobileMenu = memo(NavMobileMenuComponent);
NavMobileMenu.displayName = 'NavMobileMenu';

export default NavMobileMenu;
