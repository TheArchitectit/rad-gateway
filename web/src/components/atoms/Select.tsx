import React from 'react';

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

interface SelectProps extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, 'onChange'> {
  label?: string;
  error?: string;
  helperText?: string;
  options: SelectOption[];
  placeholder?: string;
  onChange?: (value: string) => void;
}

export function Select({
  label,
  error,
  helperText,
  options,
  placeholder,
  onChange,
  className = '',
  ...props
}: SelectProps) {
  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onChange?.(e.target.value);
  };

  return (
    <div className="w-full">
      {label && (
        <label className="block text-sm font-medium text-[var(--ink-700)] mb-1">
          {label}
          {props.required && <span className="text-[#b45c3c] ml-1">*</span>}
        </label>
      )}
      <div className="relative">
        <select
          className={`block w-full rounded-lg border px-3 py-2 pr-10 bg-[var(--surface-panel)]
            text-[var(--ink-900)]
            focus:outline-none focus:ring-2 focus:ring-[#b18532] focus:border-transparent
            appearance-none cursor-pointer
            transition-colors
            ${error ? 'border-[#b45c3c] focus:ring-[#b45c3c]' : 'border-[var(--line-strong)]'}
            ${className}`}
          onChange={handleChange}
          {...props}
        >
          {placeholder && (
            <option value="" disabled>
              {placeholder}
            </option>
          )}
          {options.map((option) => (
            <option key={option.value} value={option.value} disabled={option.disabled}>
              {option.label}
            </option>
          ))}
        </select>
        <div className="absolute inset-y-0 right-0 flex items-center px-2 pointer-events-none">
          <svg
            className="h-5 w-5 text-[var(--ink-400)]"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>
      {error && <p className="mt-1 text-sm text-[#b45c3c]">{error}</p>}
      {helperText && !error && <p className="mt-1 text-sm text-[var(--ink-500)]">{helperText}</p>}
    </div>
  );
}

interface MultiSelectProps extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, 'onChange' | 'value'> {
  label?: string;
  error?: string;
  helperText?: string;
  options: SelectOption[];
  value: string[];
  onChange?: (value: string[]) => void;
}

export function MultiSelect({
  label,
  error,
  helperText,
  options,
  value,
  onChange,
  className = '',
  ...props
}: MultiSelectProps) {
  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const selectedOptions = Array.from(e.target.selectedOptions).map((opt) => opt.value);
    onChange?.(selectedOptions);
  };

  return (
    <div className="w-full">
      {label && (
        <label className="block text-sm font-medium text-[var(--ink-700)] mb-1">
          {label}
          {props.required && <span className="text-[#b45c3c] ml-1">*</span>}
        </label>
      )}
      <select
        multiple
        className={`block w-full rounded-lg border px-3 py-2 bg-[var(--surface-panel)]
          text-[var(--ink-900)]
          focus:outline-none focus:ring-2 focus:ring-[#b18532] focus:border-transparent
          transition-colors
          ${error ? 'border-[#b45c3c] focus:ring-[#b45c3c]' : 'border-[var(--line-strong)]'}
          ${className}`}
        onChange={handleChange}
        value={value}
        {...props}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value} disabled={option.disabled}>
            {option.label}
          </option>
        ))}
      </select>
      {error && <p className="mt-1 text-sm text-[#b45c3c]">{error}</p>}
      {helperText && !error && <p className="mt-1 text-sm text-[var(--ink-500)]">{helperText}</p>}
    </div>
  );
}
