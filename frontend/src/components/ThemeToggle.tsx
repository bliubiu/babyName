'use client';

import { useState, useEffect } from 'react';
import { IconSun, IconMoon } from './Icons';

function getInitialTheme(): boolean {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('theme');
    return saved === 'dark';
  }
  return false;
}

export default function ThemeToggle() {
  const [isDark, setIsDark] = useState(getInitialTheme);

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