'use client';

import { useState, useEffect } from 'react';
import { IconSun, IconMoon } from './Icons';

export default function ThemeToggle() {
  // 初值固定 false 仅用于首帧渲染，避免 SSR/静态导出时的 hydration mismatch。
  // 真实主题由布局内联防闪烁脚本在 <html> 上挂好 .dark；本组件挂载后只「读取」
  // <html> 的 .dark 状态来同步图标，绝不在挂载时改写 <html> class（否则会把脚本
  // 已加好的 dark 误删，导致深色用户刷新时闪回浅色）。
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    setIsDark(document.documentElement.classList.contains('dark'));
  }, []);

  const toggleTheme = () => {
    const willDark = !document.documentElement.classList.contains('dark');
    document.documentElement.classList.toggle('dark', willDark);
    try {
      localStorage.setItem('theme', willDark ? 'dark' : 'light');
    } catch {
      /* 隐私模式 localStorage 不可用时忽略 */
    }
    setIsDark(willDark);
  };

  return (
    <button
      onClick={toggleTheme}
      className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2 py-1.5 rounded-lg hover:bg-crimson/5 text-sm"
      title={isDark ? '切换到浅色模式' : '切换到深色模式'}
      aria-label={isDark ? '切换到浅色模式' : '切换到深色模式'}
    >
      {isDark ? <IconSun size={16} /> : <IconMoon size={16} />}
      <span className="hidden sm:inline">{isDark ? '浅色' : '深色'}</span>
    </button>
  );
}
