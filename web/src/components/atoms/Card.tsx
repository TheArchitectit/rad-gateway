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
    md: 'shadow-md',
    lg: 'shadow-lg',
  };

  return (
    <div className={`bg-white rounded-lg border border-gray-200 ${shadowStyles[shadow]} ${className}`}>
      {(title || header) && (
        <div className="px-6 py-4 border-b border-gray-200">
          {header || (title && <h3 className="text-lg font-semibold text-gray-900">{title}</h3>)}
        </div>
      )}
      <div className="p-6">{children}</div>
      {footer && (
        <div className="px-6 py-4 border-t border-gray-200 bg-gray-50 rounded-b-lg">
          {footer}
        </div>
      )}
    </div>
  );
}
