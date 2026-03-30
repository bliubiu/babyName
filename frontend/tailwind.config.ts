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
        'warm-white': '#F5F0E8',
        'paper': '#E8DFD0',
        'ink': '#2C2C2C',
        'crimson': '#8B2323',
        'gold': '#C9A962',
        'teal': '#4A6670',
        'score-green': '#5B8C5A',
        'score-orange': '#D4A574',
        'stone-bg': '#E7E5E4',
        'stone-border': '#78716C',
        'proto-red': '#B91C1C',
        'proto-red-dark': '#991B1B',
        'proto-amber': '#D97706',
        'proto-amber-light': '#FEF3C7',
        'proto-stone': '#57534E',
        dark: {
          'warm-white': '#1A1A1A',
          'paper': '#2D2D2D',
          'ink': '#F5F0E8',
          'crimson': '#A63939',
          'gold': '#D4B57A',
          'teal': '#5A7A85',
          'score-green': '#6B9D6A',
          'score-orange': '#E0B584',
          'stone-bg': '#292524',
          'stone-border': '#A8A29E',
        },
      },
      fontFamily: {
        'serif': ['Noto Serif SC', 'serif'],
        'sans': ['Noto Sans SC', 'sans-serif'],
      },
    },
  },
  plugins: [],
};

export default config;
