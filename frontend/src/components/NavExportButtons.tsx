'use client';

import { memo } from 'react';
import NavButton from './NavButton';

interface NavExportButtonsProps {
  onExport: () => void;
  onExportPDF: () => void;
}

function NavExportButtonsComponent({ onExport, onExportPDF }: NavExportButtonsProps) {
  return (
    <div className="flex gap-2">
      <NavButton
        icon="📷"
        label="导出图片"
        onClick={onExport}
        aria-label="导出为图片"
      />
      <NavButton
        icon="📄"
        label="导出PDF"
        onClick={onExportPDF}
        aria-label="导出为PDF"
      />
    </div>
  );
}

const NavExportButtons = memo(NavExportButtonsComponent);
NavExportButtons.displayName = 'NavExportButtons';

export default NavExportButtons;
