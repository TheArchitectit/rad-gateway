import React from 'react';

interface CardProps {
  children: React.ReactNode;
  title?: string;
  header?: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
  shadow?: 'sm' | 'md' | 'lg' | 'none';
}

export function Card({
  children,
  title,
  header,
  footer,
  className = '',
  shadow = 'md',
}: CardProps) {
  const shadowStyles = {
    none: '',
    sm: 'shadow-sm',
    md: 'shadow-[0_8px_22px_rgba(7,9,13,0.28)]',
    lg: 'shadow-[0_14px_28px_rgba(7,9,13,0.35)]',
  };

  return (
    <div className={`ui-panel rounded-xl ${shadowStyles[shadow]} ${className}`}>
      {(title || header) && (
        <div className="border-b border-[var(--line-strong)] px-6 py-4">
          {header || (title && <h3 className="text-lg font-semibold text-[var(--ink-900)]">{title}</h3>)}
        </div>
      )}
      <div className="p-6">{children}</div>
      {footer && (
        <div className="ui-panel-soft rounded-b-xl border-t border-[var(--line-soft)] px-6 py-4">
          {footer}
        </div>
      )}
    </div>
  );
}
