import Link from 'next/link';

export default function NotFound() {
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
            <Link
              href="/"
              className="px-5 py-2.5 bg-crimson text-white rounded-lg text-sm font-medium
                         hover:bg-crimson-light transition-colors"
            >
              返回首页
            </Link>
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
