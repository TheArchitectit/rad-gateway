/**
 * Error Fallback Component
 * Sprint 6.3: Error Boundary Fallbacks
 */

'use client';

import React from 'react';
import { AlertTriangle, RefreshCw, Home, ArrowLeft } from 'lucide-react';
import { Button } from '@/components/atoms/Button';
import { Card } from '@/components/atoms/Card';

interface ErrorFallbackProps {
  error: Error;
  resetErrorBoundary: () => void;
  title?: string;
  description?: string;
  showHome?: boolean;
  showBack?: boolean;
}

export const ErrorFallback: React.FC<ErrorFallbackProps> = ({
  error,
  resetErrorBoundary,
  title = 'Something went wrong',
  description = 'An unexpected error occurred. Please try again or contact support if the problem persists.',
  showHome = true,
  showBack = true,
}) => {
  const handleGoHome = () => {
    window.location.href = '/';
  };

  const handleGoBack = () => {
    window.history.back();
  };

  return (
    <div className="min-h-[400px] flex items-center justify-center p-4">
      <Card className="max-w-lg w-full">
        <div className="text-center space-y-6">
          {/* Error Icon */}
          <div className="mx-auto w-16 h-16 rounded-full bg-red-100 flex items-center justify-center">
            <AlertTriangle className="w-8 h-8 text-red-600" />
          </div>

          {/* Title & Description */}
          <div className="space-y-2">
            <h2 className="text-xl font-semibold text-[var(--ink-900)]">
              {title}
            </h2>
            <p className="text-[var(--ink-500)] text-sm">
              {description}
            </p>
          </div>

          {/* Error Details (collapsible) */}
          <details className="text-left">
            <summary className="cursor-pointer text-sm text-[var(--ink-500)] hover:text-[var(--ink-700)]">
              Show error details
            </summary>
            <pre className="mt-2 p-3 bg-[var(--panel)] rounded-lg text-xs text-[var(--ink-700)] overflow-auto max-h-32">
              {error.message}
              {error.stack && `\n\n${error.stack}`}
            </pre>
          </details>

          {/* Actions */}
          <div className="flex flex-wrap gap-3 justify-center">
            <Button
              onClick={resetErrorBoundary}
              variant="primary"
              className="gap-2"
            >
              <RefreshCw className="w-4 h-4" />
              Try Again
            </Button>

            {showBack && (
              <Button
                onClick={handleGoBack}
                variant="secondary"
                className="gap-2"
              >
                <ArrowLeft className="w-4 h-4" />
                Go Back
              </Button>
            )}

            {showHome && (
              <Button
                onClick={handleGoHome}
                variant="ghost"
                className="gap-2"
              >
                <Home className="w-4 h-4" />
                Dashboard
              </Button>
            )}
          </div>
        </div>
      </Card>
    </div>
  );
};

// Specialized fallbacks for specific sections
export const DashboardErrorFallback: React.FC<{
  error: Error;
  resetErrorBoundary: () => void;
}> = (props) => (
  <ErrorFallback
    {...props}
    title="Dashboard Error"
    description="Failed to load dashboard metrics. This may be due to a network issue or server problem."
  />
);

export const ProviderErrorFallback: React.FC<{
  error: Error;
  resetErrorBoundary: () => void;
}> = (props) => (
  <ErrorFallback
    {...props}
    title="Provider Load Error"
    description="Failed to load provider information. Please check your connection and try again."
  />
);

export const ApiKeysErrorFallback: React.FC<{
  error: Error;
  resetErrorBoundary: () => void;
}> = (props) => (
  <ErrorFallback
    {...props}
    title="API Keys Error"
    description="Failed to load API keys. This may be a permission issue or temporary server problem."
  />
);

export const ControlRoomErrorFallback: React.FC<{
  error: Error;
  resetErrorBoundary: () => void;
}> = (props) => (
  <ErrorFallback
    {...props}
    title="Control Room Error"
    description="Failed to initialize control room monitoring. Real-time updates may be unavailable."
  />
);

export default ErrorFallback;
