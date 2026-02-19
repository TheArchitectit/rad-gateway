import React from 'react';
import { Input } from '../atoms/Input';

interface FormFieldProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  helperText?: string;
}

export function FormField({ label, error, helperText, ...props }: FormFieldProps) {
  return (
    <div className="space-y-1">
      <Input
        label={label}
        {...(error && { error })}
        {...(helperText && { helperText })}
        {...props}
      />
    </div>
  );
}
