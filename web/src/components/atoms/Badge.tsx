import React from 'react';

type BadgeColor = 'success' | 'warning' | 'error' | 'info' | 'default';

interface BadgeProps {
  children: React.ReactNode;
  color?: BadgeColor;
  size?: 'sm' | 'md';
  className?: string;
}

export function Badge({
  children,
  color = 'default',
  size = 'md',
  className = '',
}: BadgeProps) {
  const colorStyles: Record<BadgeColor, string> = {
    success: 'border border-[rgba(47,122,79,0.3)] bg-[rgba(47,122,79,0.12)] text-[var(--status-normal)]',
    warning: 'border border-[rgba(182,109,29,0.3)] bg-[rgba(182,109,29,0.12)] text-[var(--status-warning)]',
    error: 'border border-[rgba(152,43,33,0.3)] bg-[rgba(152,43,33,0.12)] text-[var(--status-critical)]',
    info: 'border border-[rgba(47,95,132,0.3)] bg-[rgba(47,95,132,0.12)] text-[var(--status-info)]',
    default: 'border border-[var(--line-soft)] bg-[rgba(43,32,21,0.08)] text-[var(--ink-700)]',
  };

  const sizeStyles = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-0.5 text-sm',
  };

  return (
    <span className={`inline-flex items-center rounded-full font-medium ${colorStyles[color]} ${sizeStyles[size]} ${className}`}>
      {children}
    </span>
  );
}
