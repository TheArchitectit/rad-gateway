/**
 * RAD Gateway Admin UI - Metric Card Component
 * State Management Engineer - Phase 2 Implementation
 *
 * Card for displaying metrics with loading state.
 */

import { Skeleton } from '../common/Skeleton';

interface MetricCardProps {
  title: string;
  value: string | number;
  description?: string;
  change?: {
    value: number;
    positive: boolean;
  };
  isLoading?: boolean;
  className?: string;
}

export function MetricCard({
  title,
  value,
  description,
  change,
  isLoading = false,
  className = '',
}: MetricCardProps) {
  if (isLoading) {
    return (
      <div className={`p-4 rounded-lg border border-gray-200 dark:border-gray-700 ${className}`}>
        <Skeleton className="h-4 w-24 mb-2" />
        <Skeleton className="h-8 w-32 mb-2" />
        <Skeleton className="h-3 w-16" />
      </div>
    );
  }

  return (
    <div className={`p-4 rounded-lg border border-gray-200 dark:border-gray-700 ${className}`}>
      <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">
        {title}
      </h3>
      <div className="mt-2 flex items-baseline">
        <span className="text-2xl font-semibold text-gray-900 dark:text-white">
          {value}
        </span>
        {change && (
          <span
            className={`ml-2 text-sm font-medium ${
              change.positive ? 'text-green-600' : 'text-red-600'
            }`}
          >
            {change.positive ? '+' : ''}
            {change.value}%
          </span>
        )}
      </div>
      {description && (
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {description}
        </p>
      )}
    </div>
  );
}
