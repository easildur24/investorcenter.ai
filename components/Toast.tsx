'use client';

import { useEffect } from 'react';

export type ToastType = 'success' | 'error' | 'warning' | 'info';

interface ToastProps {
  message: string;
  type: ToastType;
  onClose: () => void;
  duration?: number;
}

export default function Toast({ message, type, onClose, duration = 5000 }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onClose();
    }, duration);

    return () => clearTimeout(timer);
  }, [duration, onClose]);

  const bgColors = {
    success: 'bg-ic-positive',
    error: 'bg-ic-negative',
    warning: 'bg-ic-warning',
    info: 'bg-ic-blue',
  };

  const icons = {
    success: '✓',
    error: '✕',
    warning: '⚠',
    info: 'ℹ',
  };

  return (
    <div
      className={`fixed bottom-4 right-4 ${bgColors[type]} text-ic-text-primary px-6 py-4 rounded-lg shadow-lg flex items-center gap-3 animate-slide-up z-50`}
      role="alert"
    >
      <span className="text-xl font-bold">{icons[type]}</span>
      <span className="font-medium">{message}</span>
      <button
        onClick={onClose}
        className="ml-4 text-ic-text-primary hover:opacity-80 font-bold text-lg transition-opacity"
        aria-label="Close"
      >
        ×
      </button>
    </div>
  );
}
