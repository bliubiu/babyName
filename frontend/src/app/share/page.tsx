'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Navigation from '@/components/Navigation';
import { getShareSnapshot, type ShareSnapshot } from '@/lib/api';

type LoadState =
  | { status: 'loading' }
  | { status: 'ready'; data: ShareSnapshot }
  | { status: 'error'; message: string };

/**
 * 分享落地页：/share?id=<随机 id>。
 * 用 window.location.search 读取 id（静态导出下不引入 useSearchParams 的
 * Suspense 约束），快照由后端 /api/v1/share/:id 提供，页面本身无登录态。
 */
export default function SharePage() {
  const router = useRouter();
  const [state, setState] = useState<LoadState>({ status: 'loading' });

  useEffect(() => {
    let cancelled = false;
    const id = new URLSearchParams(window.location.search).get('id') ?? '';
    // 缺 id 与读取失败都走同一条 Promise 链：状态只在异步回调里更新，
    // 避免在 effect 主体里同步 setState 触发级联渲染。
    const task = id
      ? getShareSnapshot(id)
      : Promise.reject(new Error('分享链接缺少标识，请向分享者索要完整链接'));
    task
      .then((data) => {
        if (!cancelled) setState({ status: 'ready', data });
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setState({
            status: 'error',
            message: err instanceof Error ? err.message : '分享内容读取失败',
          });
        }
      });
    return () => { cancelled = true; };
  }, []);

  const chips = state.status === 'ready'
    ? [
        state.data.rishou ? `日主 ${state.data.rishou}` : '',
        state.data.xiyongshen?.length ? `喜用神 ${state.data.xiyongshen.join('、')}` : '',
        state.data.zodiac || '',
        state.data.hexagram_name || '',
        state.data.five_elements || '',
      ].filter(Boolean)
    : [];

  return (
    <main className="min-h-screen py-4 md:py-8 px-4 md:px-6">
      <div className="max-w-5xl xl:max-w-6xl 2xl:max-w-7xl mx-auto">
        <Navigation showBackButton={true} showHistory={false} showFavorites={false} />

        {state.status === 'loading' && (
          <div className="card text-center py-14 text-jade/80 text-sm">正在读取分享内容…</div>
        )}

        {state.status === 'error' && (
          <div className="card text-center py-14">
            <p className="text-ink font-medium mb-2">无法打开这份分享</p>
            <p className="text-sm text-jade/80 mb-6">{state.message}</p>
            <button
              type="button"
              onClick={() => router.push('/')}
              className="min-h-[44px] px-5 py-2 bg-crimson text-white rounded-xl text-sm hover:bg-crimson-light transition-all duration-200"
            >
              回到首页
            </button>
          </div>
        )}

        {state.status === 'ready' && (
          <>
            <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
              <div>
                <h1 className="font-serif text-xl md:text-2xl text-ink">
                  {state.data.title || '宝宝起名结果分享'}
                </h1>
                <p className="text-xs text-ink-light/80 mt-1">
                  共 {state.data.names.length} 个名字{state.data.total ? ` / 原结果 ${state.data.total} 个` : ''} · 来自朋友分享
                </p>
              </div>
              <span className="text-xs text-jade/80">匿名分享 · 不含出生信息</span>
            </div>

            {chips.length > 0 && (
              <div className="mb-5 flex flex-wrap gap-2">
                {chips.map(chip => (
                  <span key={chip} className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade/80 rounded-full">
                    {chip}
                  </span>
                ))}
              </div>
            )}

            <section aria-label="分享的名字列表" className="mb-6">
              <h2 className="section-title mb-4">推荐名字</h2>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-4">
                {state.data.names.map((name, index) => (
                  <div key={`${name.full_name}-${index}`} className="card p-4">
                    <div className="flex items-baseline justify-between gap-2">
                      <span className="font-serif text-lg text-ink">{name.full_name}</span>
                      {typeof name.score === 'number' && (
                        <span className="text-sm text-gold font-medium">{name.score.toFixed(1)}分</span>
                      )}
                    </div>
                    <div className="mt-1 text-xs text-jade/80">
                      {[name.pinyin, name.wuxing].filter(Boolean).join(' · ')}
                    </div>
                    {name.meaning && (
                      <p className="mt-2 text-sm text-ink-light/80 leading-relaxed">{name.meaning}</p>
                    )}
                  </div>
                ))}
              </div>
            </section>

            <div className="card flex flex-wrap items-center justify-between gap-3 p-4">
              <p className="text-sm text-jade/80">想按自己的生辰重新算一份？</p>
              <button
                type="button"
                onClick={() => router.push('/')}
                className="min-h-[44px] px-5 py-2 bg-crimson text-white rounded-xl text-sm hover:bg-crimson-light transition-all duration-200"
              >
                我也想起名
              </button>
            </div>
          </>
        )}
      </div>
    </main>
  );
}
