import { useNavigate } from 'react-router-dom';

/**
 * Next.js useRouter 迁移垫片：仅覆盖本项目用到的 push 能力。
 * 各页面只需把 `import { useRouter } from 'next/navigation'`
 * 换成 `import { useRouter } from '@/lib/navigation'`，其余代码不动。
 */
export function useRouter(): { push: (path: string) => void } {
  const navigate = useNavigate();
  return { push: (path: string) => navigate(path) };
}
