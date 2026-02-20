import React from 'react';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
}

export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  children,
  disabled,
  className = '',
  ...props
}: ButtonProps) {
  const baseStyles = 'inline-flex items-center justify-center rounded-lg font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:cursor-not-allowed';

  const variantStyles = {
    primary:
      'bg-gradient-to-r from-[#c79a45] via-[#9f712a] to-[#73531e] text-[#21160f] shadow-[inset_0_1px_0_rgba(255,255,255,0.4)] hover:brightness-110 focus:ring-[#b18532] disabled:opacity-45',
    secondary:
      'border border-[var(--line-strong)] bg-[var(--surface-panel-muted)] text-[var(--ink-900)] hover:bg-[var(--surface-panel-soft)] focus:ring-[#b18532] disabled:opacity-45',
    danger:
      'bg-gradient-to-r from-[#b45c3c] to-[#7a2c20] text-[#f8e8df] hover:brightness-110 focus:ring-[#982b21] disabled:opacity-45',
    ghost:
      'bg-transparent text-[var(--ink-700)] hover:bg-[rgba(43,32,21,0.08)] focus:ring-[#7b6647] disabled:opacity-45',
  };
  
  const sizeStyles = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg',
  };

  return (
    <button
      className={`${baseStyles} ${variantStyles[variant]} ${sizeStyles[size]} ${className}`}
      disabled={disabled || loading}
      {...props}
    >
      {loading && (
        <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-current" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      )}
      {children}
    </button>
  );
}
