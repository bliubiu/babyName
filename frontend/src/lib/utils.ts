import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

/**
 * 合并 Tailwind CSS 类名，自动处理冲突
 * @example
 * cn('px-2', 'py-4') // => 'px-2 py-4'
 * cn('px-2', 'px-4') // => 'px-4' (后者覆盖前者)
 * cn('text-red', true && 'text-bold') // => 'text-red text-bold'
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * 卡片样式变体
 */
export const cardVariants = {
  base: 'card cursor-pointer transition-all duration-300 hover:shadow-lg hover-lift animate-fadeInUp',
  selected: 'ring-2 ring-crimson',
  comparing: 'ring-2 ring-gold',
};

/**
 * 按钮样式变体
 */
export const buttonVariants = {
  primary: 'bg-crimson text-white hover:bg-crimson/90 transition-colors',
  secondary: 'bg-stone-200 text-stone-800 hover:bg-stone-300 transition-colors',
  outline: 'border-2 border-crimson text-crimson hover:bg-crimson hover:text-white transition-colors',
  disabled: 'bg-stone-300 text-stone-500 cursor-not-allowed',
};

/**
 * 输入框样式变体
 */
export const inputVariants = {
  base: 'proto-input flex-1 max-w-xs px-4 py-2 border border-stone-300 bg-white focus:outline-none focus:border-red-500',
  error: 'border-red-500',
  disabled: 'bg-stone-100 cursor-not-allowed',
};

/**
 * 标签样式变体
 */
export const badgeVariants = {
  default: 'text-xs px-2 py-0.5 bg-warm-white text-teal rounded',
  gender: 'text-xs px-2 py-0.5 bg-warm-white text-teal rounded hover:bg-warm-white/80 transition-colors duration-200',
  score: 'text-xs px-2 py-0.5 bg-gold/20 text-gold rounded hover:bg-gold/30 transition-colors duration-200',
};
