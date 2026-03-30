'use client';

import { useRouter } from 'next/navigation';
import { memo } from 'react';

interface NavBackButtonProps {
  className?: string;
}

function NavBackButtonComponent({ className = '' }: NavBackButtonProps) {
  const router = useRouter();

  return (
    <button
      onClick={() => router.push('/')}
      className={`flex items-center gap-2 text-teal hover:text-crimson transition-all duration-300 text-sm md:text-base hover-lift ${className}`}
      aria-label="返回首页"
    >
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
      </svg>
      <span>返回首页</span>
    </button>
  );
}

const NavBackButton = memo(NavBackButtonComponent);
NavBackButton.displayName = 'NavBackButton';

export default NavBackButton;
