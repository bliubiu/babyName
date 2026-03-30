'use client';

import React, { ReactNode, Suspense } from 'react';

interface SuspenseWrapperProps {
  children: ReactNode;
  fallback?: ReactNode;
}

export function SuspenseWrapper({ children, fallback }: SuspenseWrapperProps) {
  return (
    <Suspense
      fallback={
        fallback || (
          <div className="min-h-screen flex items-center justify-center">
            <div className="text-center">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-crimson mx-auto mb-4"></div>
              <p className="text-ink">加载中...</p>
            </div>
          </div>
        )
      }
    >
      {children}
    </Suspense>
  );
}
