'use client';

import { memo, forwardRef } from 'react';
import { cn } from '@/lib/utils';

interface NavButtonProps {
  icon: string;
  label: string;
  onClick: () => void;
  isActive?: boolean;
  className?: string;
}

const NavButton = forwardRef<HTMLButtonElement, NavButtonProps>(function NavButtonComponent({ 
  icon, 
  label, 
  onClick, 
  isActive = false,
  className = ''
}, ref) {
  return (
    <button
      ref={ref}
      onClick={onClick}
      className={cn(
        'text-jade hover:text-crimson transition-all duration-300',
        'px-2 py-1 rounded-md hover:bg-crimson/10 hover-lift',
        'text-sm md:text-base flex items-center gap-1',
        isActive && 'text-crimson bg-crimson/10',
        className
      )}
      aria-label={label}
      aria-current={isActive ? 'page' : undefined}
    >
      <span className="text-base md:text-lg">{icon}</span>
      <span className="hidden sm:inline">{label}</span>
    </button>
  );
});

NavButton.displayName = 'NavButton';

export default NavButton;
