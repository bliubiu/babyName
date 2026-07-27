'use client';

import { useEffect } from 'react';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('页面错误:', error);
  }, [error]);

  return (
    <main className="min-h-screen flex items-center justify-center p-4 cloud-bg">
      <div className="max-w-md mx-auto text-center">
        <div className="consultation-sheet p-8">
          {/* 错误图标 */}
          <div className="w-16 h-16 mx-auto mb-6 rounded-full bg-crimson/10 flex items-center justify-center">
            <span className="text-3xl text-crimson font-serif">惋</span>
          </div>

          <h1 className="font-serif text-2xl text-crimson mb-3">
            页面出了点问题
          </h1>
          <p className="text-ink-light text-sm mb-6 leading-relaxed">
            很抱歉，页面加载过程中出现了错误。
            <br />
            请稍后重试，或返回首页继续使用。
          </p>

          <div className="flex items-center justify-center gap-3">
            <button
              onClick={reset}
              className="px-5 py-2.5 bg-crimson text-white rounded-lg text-sm font-medium
                         hover:bg-crimson-light transition-colors"
            >
              重试
            </button>
            <a
              href="/"
              className="px-5 py-2.5 border border-paper-edge rounded-lg text-sm text-ink-light
                         hover:border-crimson/30 hover:text-crimson transition-colors"
            >
              返回首页
            </a>
          </div>

          {error.digest && (
            <p className="text-xs text-jade/40 mt-6">
              错误编号: {error.digest}
            </p>
          )}
        </div>
      </div>
    </main>
  );
}
