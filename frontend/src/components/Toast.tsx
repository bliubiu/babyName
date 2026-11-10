'use client';

import { createContext, useContext, useState, useCallback, useEffect, useRef, ReactNode } from 'react';

type ToastType = 'success' | 'error' | 'warning' | 'info';

export interface ToastAction {
  label: string;
  onClick: () => void;
}

interface Toast {
  id: string;
  message: string;
  type: ToastType;
  action?: ToastAction;
}

interface ToastContextType {
  showToast: (message: string, type?: ToastType, action?: ToastAction) => void;
}

const ToastContext = createContext<ToastContextType | undefined>(undefined);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());
  // 自增 id：Date.now() 同毫秒触发两条 toast 会生成相同 id，
  // 定时器互相覆盖、filter 误删（docs/29 P3）
  const seqRef = useRef(0);

  // 组件卸载时清理所有未完成的定时器
  useEffect(() => {
    const timers = timersRef.current;
    return () => {
      timers.forEach((timer) => clearTimeout(timer));
      timers.clear();
    };
  }, []);

  const showToast = useCallback((message: string, type: ToastType = 'info', action?: ToastAction) => {
    seqRef.current += 1;
    const id = `toast-${seqRef.current}`;
    setToasts((prev) => [...prev, { id, message, type, action }]);

    // 带操作按钮的提示留更长时间，给用户看清并点击「撤销」
    const timer = setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
      timersRef.current.delete(id);
    }, action ? 5000 : 3000);
    timersRef.current.set(id, timer);
  }, []);

  const removeToast = (id: string) => {
    const timer = timersRef.current.get(id);
    if (timer) {
      clearTimeout(timer);
      timersRef.current.delete(id);
    }
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      {/* aria-live：屏幕阅读器需要能感知到操作反馈（docs/29 P3 a11y） */}
      <div className="toast-container" role="status" aria-live="polite">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`toast toast-${toast.type}`}
            onClick={() => removeToast(toast.id)}
          >
            <span className="toast-icon">
              {toast.type === 'success' && (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                </svg>
              )}
              {toast.type === 'error' && (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M18 6L6 18M6 6l12 12" />
                </svg>
              )}
              {toast.type === 'warning' && (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 9v2m0 4h.01M12 2l10 18H2L12 2z" />
                </svg>
              )}
              {toast.type === 'info' && (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <circle cx="12" cy="12" r="10" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M12 16v-4m0-4h.01" />
                </svg>
              )}
            </span>
            <span className="toast-message">{toast.message}</span>
            {toast.action && (
              <button
                type="button"
                className="shrink-0 ml-2 px-2 py-1 text-xs font-medium text-crimson hover:bg-crimson/10 rounded transition-colors"
                onClick={(e) => {
                  // 阻止冒泡：整条 toast 的 onClick 会直接关闭，操作按钮要先执行动作
                  e.stopPropagation();
                  toast.action?.onClick();
                  removeToast(toast.id);
                }}
              >
                {toast.action.label}
              </button>
            )}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
}
