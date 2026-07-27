'use client';

import React, { Component, ErrorInfo, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
    };
  }

  static getDerivedStateFromError(error: Error): State {
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error('Error caught by ErrorBoundary:', error, errorInfo);
  }

  render(): ReactNode {
    const { hasError, error } = this.state;
    const { children, fallback } = this.props;

    if (hasError) {
      if (fallback) {
        return fallback;
      }

      return (
        <div className="min-h-screen flex items-center justify-center p-4">
          <div className="max-w-md mx-auto text-center">
            <h2 className="text-2xl font-bold text-crimson mb-4">
              发生错误
            </h2>
            <p className="text-ink mb-6">
              页面加载过程中出现了错误，请稍后重试。
            </p>
            <button
              onClick={() => window.location.reload()}
              className="px-4 py-2 bg-crimson text-white rounded-lg hover:bg-crimson/90 transition-colors"
            >
              刷新页面
            </button>
            <p className="text-xs text-jade mt-4">
              错误信息: {error?.message}
            </p>
          </div>
        </div>
      );
    }

    return children;
  }
}
