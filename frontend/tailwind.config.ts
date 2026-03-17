import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
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
