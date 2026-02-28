import React from 'react';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helperText?: string;
}

export function Input({
  label,
  error,
  helperText,
  className = '',
  ...props
}: InputProps) {
  return (
    <div className="w-full">
      {label && (
        <label className="block text-sm font-medium text-[var(--ink-700)] mb-1">
          {label}
          {props.required && <span className="text-[#b45c3c] ml-1">*</span>}
        </label>
      )}
      <input
        className={`block w-full rounded-lg border px-3 py-2 bg-[var(--surface-panel)]
          text-[var(--ink-900)] placeholder-[var(--ink-400)]
          focus:outline-none focus:ring-2 focus:ring-[#b18532] focus:border-transparent
          transition-colors
          ${error ? 'border-[#b45c3c] focus:ring-[#b45c3c]' : 'border-[var(--line-strong)]'}
          ${className}`}
        {...props}
      />
      {error && <p className="mt-1 text-sm text-[#b45c3c]">{error}</p>}
      {helperText && !error && <p className="mt-1 text-sm text-[var(--ink-500)]">{helperText}</p>}
    </div>
  );
}
