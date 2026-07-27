import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // 主色调 - 温润中国风（CSS变量实现深色模式自适应）
        'warm-white': 'rgb(var(--color-warm-white) / <alpha-value>)',
        'warm-white-light': 'rgb(var(--color-warm-white-light) / <alpha-value>)',
        'paper': 'rgb(var(--color-paper) / <alpha-value>)',
        'ink': 'rgb(var(--color-ink) / <alpha-value>)',
        'ink-light': 'rgb(var(--color-ink-light) / <alpha-value>)',
        'crimson': 'rgb(var(--color-crimson) / <alpha-value>)',
        'crimson-light': 'rgb(var(--color-crimson-light) / <alpha-value>)',
        'crimson-dark': 'rgb(var(--color-crimson-dark) / <alpha-value>)',
        'gold': 'rgb(var(--color-gold) / <alpha-value>)',
        'gold-light': 'rgb(var(--color-gold-light) / <alpha-value>)',
        'gold-dark': 'rgb(var(--color-gold-dark) / <alpha-value>)',
        'paper-edge': 'rgb(var(--color-paper-edge) / <alpha-value>)',
        'jade': 'rgb(var(--color-jade) / <alpha-value>)',
        'jade-light': 'rgb(var(--color-jade-light) / <alpha-value>)',
        'jade-dark': 'rgb(var(--color-jade-dark) / <alpha-value>)',
        'seal-red': 'rgb(var(--color-seal-red) / <alpha-value>)',
        'seal-red-light': 'rgb(var(--color-seal-red-light) / <alpha-value>)',
        // 评分色
        'score-green': '#5B8C5A',
        'score-orange': '#D4A574',
        // 国画色
        'rouge': '#C04040',
        'rouge-light': '#D46060',
        'ochre': '#9A7B4F',
        'ochre-light': '#B8986A',
        'stone-blue': '#5B7B9A',
        'stone-blue-light': '#7895B0',
        'cinnabar': '#E85D3A',
        // 辅助色
        'stone-bg': '#E7E5E4',
        'stone-border': '#78716C',
        // 五行色
        'wuxing-jin': '#8C8C8C',
        'wuxing-mu': '#4CAF50',
        'wuxing-shui': '#2196F3',
        'wuxing-huo': '#F44336',
        'wuxing-tu': '#FF9800',
      },
      fontFamily: {
        'sans': ['var(--font-noto-sans-sc)', 'sans-serif'],
        'serif': ['var(--font-noto-serif-sc)', 'serif'],
      },
      animation: {
        'fade-in-up': 'fadeInUp 0.5s ease-out',
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-in-left': 'slideInLeft 0.5s ease-out',
        'slide-in-right': 'slideInRight 0.5s ease-out',
        'slide-down': 'slideDown 0.4s ease-out',
        'scale-in': 'scaleIn 0.3s ease-out',
        'shimmer': 'shimmer 2s infinite',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'seal-press': 'sealPress 0.6s ease-out forwards',
        'scroll-unfurl': 'scrollUnfurl 0.8s ease-out forwards',
        'ink-spread': 'inkSpread 1.2s ease-out forwards',
        'ink-bleed': 'inkBleed 1.5s ease-out forwards',
        'pattern-fade': 'patternFade 2s ease-out forwards',
      },
      keyframes: {
        fadeInUp: {
          '0%': { opacity: '0', transform: 'translateY(20px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideInLeft: {
          '0%': { opacity: '0', transform: 'translateX(-20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-20px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.9)' },
          '100%': { opacity: '1', transform: 'scale(1)' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        sealPress: {
          '0%': { opacity: '0', transform: 'scale(2) rotate(-15deg)' },
          '60%': { opacity: '0.8', transform: 'scale(0.95) rotate(3deg)' },
          '100%': { opacity: '1', transform: 'scale(1) rotate(0deg)' },
        },
        scrollUnfurl: {
          '0%': { clipPath: 'inset(0 100% 0 0)' },
          '100%': { clipPath: 'inset(0 0 0 0)' },
        },
        inkSpread: {
          '0%': { opacity: '0', filter: 'blur(8px)' },
          '50%': { opacity: '0.6', filter: 'blur(2px)' },
          '100%': { opacity: '1', filter: 'blur(0)' },
        },
        inkBleed: {
          '0%': { textShadow: '0 0 0 transparent', opacity: '0.6' },
          '40%': { textShadow: '2px 2px 4px rgba(0,0,0,0.08), -1px -1px 2px rgba(0,0,0,0.04)' },
          '100%': { textShadow: '1px 1px 2px rgba(0,0,0,0.06), -0.5px -0.5px 1px rgba(0,0,0,0.03)', opacity: '1' },
        },
        patternFade: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
      },
    },
  },
  plugins: [],
};

export default config;