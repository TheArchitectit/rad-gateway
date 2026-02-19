'use client';

import { ReactNode } from 'react';

interface FormFieldProps {
  label: string;
  children: ReactNode;
  error?: string | undefined;
  hint?: string | undefined;
  required?: boolean;
}

export function FormField({ label, children, error, hint, required }: FormFieldProps) {
  return (
    <div className="space-y-1.5">
      <label className="block text-sm font-medium text-gray-700">
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </label>
      {children}
      {error ? (
        <p className="text-sm text-red-600">{error}</p>
      ) : hint ? (
        <p className="text-sm text-gray-500">{hint}</p>
      ) : null}
    </div>
  );
}
