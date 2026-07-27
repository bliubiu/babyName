'use client';

import { memo } from 'react';

interface NavExportButtonsProps {
  onExport: () => void;
  onExportPDF: () => void;
}

function NavExportButtonsComponent({ onExport, onExportPDF }: NavExportButtonsProps) {
  return (
    <div className="flex flex-col gap-1">
      <button
        onClick={onExport}
        className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2 py-1 rounded-md hover:bg-crimson/10 text-sm"
        aria-label="导出为图片"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M23 19a2 2 0 01-2 2H3a2 2 0 01-2-2V8a2 2 0 012-2h4l2-3h6l2 3h4a2 2 0 012 2v11z" />
          <circle cx="12" cy="13" r="4" />
        </svg>
        导出图片
      </button>
      <button
        onClick={onExportPDF}
        className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2 py-1 rounded-md hover:bg-crimson/10 text-sm"
        aria-label="导出为PDF"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
          <polyline points="14 2 14 8 20 8" />
          <line x1="16" y1="13" x2="8" y2="13" />
          <line x1="16" y1="17" x2="8" y2="17" />
        </svg>
        导出PDF
      </button>
    </div>
  );
}

const NavExportButtons = memo(NavExportButtonsComponent);
NavExportButtons.displayName = 'NavExportButtons';

export default NavExportButtons;
