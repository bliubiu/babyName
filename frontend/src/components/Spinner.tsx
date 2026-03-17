'use client';

interface SpinnerProps {
  size?: 'small' | 'medium' | 'large';
  color?: string;
}

export function Spinner({ size = 'medium', color }: SpinnerProps) {
  const sizeClass = {
    small: 'spinner-small',
    medium: 'spinner-medium',
    large: 'spinner-large',
  }[size];

  const style = color ? { borderTopColor: color } : {};

  return (
    <div className={`spinner ${sizeClass}`} style={style} />
  );
}

export function PageLoader() {
  return (
    <div className="page-loader">
      <Spinner size="large" />
      <p className="page-loader-text">加载中...</p>
    </div>
  );
}

export function InlineLoader({ text = '加载中...' }: { text?: string }) {
  return (
    <div className="inline-loader">
      <Spinner size="small" />
      <span>{text}</span>
    </div>
  );
}
