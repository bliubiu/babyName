import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

/** 路由级错误页（与 Next app/error.tsx 同口径保留，供后续使用） */
export function RouteErrorBoundary({
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
            <button
              onClick={() => window.location.assign('/')}
              className="px-5 py-2.5 border border-paper-edge rounded-lg text-sm text-ink-light
                         hover:border-crimson/30 hover:text-crimson transition-colors"
            >
              返回首页
            </button>
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

export function NotFound() {
  const navigate = useNavigate();
  return (
    <main className="min-h-screen flex items-center justify-center p-4 cloud-bg">
      <div className="max-w-md mx-auto text-center">
        <div className="consultation-sheet p-8">
          {/* 404 装饰 */}
          <div className="mb-6">
            <span className="font-serif text-7xl text-crimson/20 tracking-widest">
              404
            </span>
          </div>

          <h1 className="font-serif text-2xl text-crimson mb-3">
            此页未找到
          </h1>
          <p className="text-ink-light text-sm mb-6 leading-relaxed">
            您访问的页面不存在，或已被移除。
            <br />
            请检查网址是否正确，或返回首页。
          </p>

          <div className="flex items-center justify-center gap-3">
            <button
              onClick={() => navigate('/')}
              className="px-5 py-2.5 bg-crimson text-white rounded-lg text-sm font-medium
                         hover:bg-crimson-light transition-colors"
            >
              返回首页
            </button>
          </div>

          {/* 装饰分隔线 */}
          <div className="flex items-center justify-center gap-4 mt-8">
            <div className="brush-divider max-w-[48px] flex-1" />
            <span className="text-paper-edge/50 text-xs tracking-[0.4em] font-serif">起名</span>
            <div className="brush-divider max-w-[48px] flex-1" />
          </div>
        </div>
      </div>
    </main>
  );
}
