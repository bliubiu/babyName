'use client';

import { useState, useEffect } from 'react';

export default function ThemeToggle() {
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    // 检查本地存储中的主题设置
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme === 'dark') {
      enableDarkMode();
    }
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
      className="flex items-center gap-2 text-teal hover:text-crimson transition-all duration-300 px-2 py-1 rounded-md hover:bg-crimson/10 hover-lift"
      title={isDark ? '切换到浅色模式' : '切换到深色模式'}
    >
      {isDark ? '☀️ 浅色' : '🌙 深色'}
    </button>
  );
}
