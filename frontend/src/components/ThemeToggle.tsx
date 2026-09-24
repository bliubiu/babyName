'use client';

import { useState, useEffect } from 'react';
import { IconSun, IconMoon } from './Icons';

export default function ThemeToggle() {
  // 初值固定 false：SSR 阶段读不到 localStorage，若用它做初值会导致
  // 服务端渲染 false、客户端渲染 true 的 hydration 不一致。真实主题在
  // 挂载后的 effect 里同步（代价是首帧短暂为浅色，但不会再报 mismatch）。
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    setIsDark(localStorage.getItem('theme') === 'dark');
  }, []);

  const enableDarkMode = () => {
    document.body.classList.add('dark');
    localStorage.setItem('theme', 'dark');
    setIsDark(true);
  };

  const enableLightMode = () => {
    document.body.classList.remove('dark');
    localStorage.setItem('theme', 'light');
    setIsDark(false);
  };

  useEffect(() => {
    if (isDark) {
      document.body.classList.add('dark');
    } else {
      document.body.classList.remove('dark');
    }
  }, [isDark]);

  const toggleTheme = () => {
    if (isDark) {
      enableLightMode();
    } else {
      enableDarkMode();
    }
  };

  return (
    <button
      onClick={toggleTheme}
      className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2 py-1.5 rounded-lg hover:bg-crimson/5 text-sm"
      title={isDark ? '切换到浅色模式' : '切换到深色模式'}
    >
      {isDark ? <IconSun size={16} /> : <IconMoon size={16} />}
      <span className="hidden sm:inline">{isDark ? '浅色' : '深色'}</span>
    </button>
  );
}