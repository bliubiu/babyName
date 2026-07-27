'use client';

import { useState, useEffect } from 'react';
import { IconSun, IconMoon } from './Icons';

export default function ThemeToggle() {
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
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
      className="flex items-center gap-1.5 text-jade hover:text-crimson transition-all duration-300 px-2 py-1.5 rounded-lg hover:bg-crimson/5 text-sm"
      title={isDark ? '切换到浅色模式' : '切换到深色模式'}
    >
      {isDark ? <IconSun size={16} /> : <IconMoon size={16} />}
      <span className="hidden sm:inline">{isDark ? '浅色' : '深色'}</span>
    </button>
  );
}