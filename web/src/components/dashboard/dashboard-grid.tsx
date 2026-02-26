'use client';

import * as React from 'react';
import { cn } from '@/lib/utils';

interface DashboardGridProps {
  children: React.ReactNode;
  className?: string;
}

/**
 * DashboardGrid - Responsive grid layout for dashboard KPI cards
 * 
 * Art Deco-inspired responsive grid with geometric precision.
 * Mobile: 1 column
 * Tablet: 2 columns  
 * Desktop: 4 columns
 */
export function DashboardGrid({ children, className }: DashboardGridProps) {
  return (
    <div 
      className={cn(
        "grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4",
        className
      )}
    >
      {children}
    </div>
  );
}

/**
 * DashboardSection - Full-width section wrapper with optional collapsible behavior
 */
interface DashboardSectionProps {
  children: React.ReactNode;
  title?: string;
  collapsible?: boolean;
  defaultOpen?: boolean;
  className?: string;
}

export function DashboardSection({
  children,
  title,
  collapsible = false,
  defaultOpen = true,
  className,
}: DashboardSectionProps) {
  const [isOpen, setIsOpen] = React.useState(defaultOpen);

  return (
    <div className={cn("space-y-4", className)}>
      {title && (
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 flex-1">
            {/* Art Deco decorative line */}
            <div className="h-px flex-1 bg-gradient-to-r from-transparent via-[var(--brass-500)]/50 to-transparent" />
            <h2 className="text-lg font-semibold text-[var(--ink-900)] tracking-wide">
              {title}
            </h2>
            <div className="h-px flex-1 bg-gradient-to-r from-transparent via-[var(--brass-500)]/50 to-transparent" />
          </div>
          {collapsible && (
            <button
              onClick={() => setIsOpen(!isOpen)}
              className="text-xs font-medium text-[var(--ink-500)] hover:text-[var(--brass-600)] transition-colors px-2 py-1 rounded hover:bg-[var(--brass-500)]/10"
            >
              {isOpen ? 'Collapse' : 'Expand'}
            </button>
          )}
        </div>
      )}
      
      {(!collapsible || isOpen) && (
        <div className="animate-in fade-in slide-in-from-top-2 duration-300">
          {children}
        </div>
      )}
    </div>
  );
}

/**
 * DashboardTwoColumn - Two column layout for charts and other content
 */
interface DashboardTwoColumnProps {
  left: React.ReactNode;
  right: React.ReactNode;
  className?: string;
}

export function DashboardTwoColumn({ left, right, className }: DashboardTwoColumnProps) {
  return (
    <div className={cn("grid grid-cols-1 lg:grid-cols-2 gap-6", className)}>
      <div className="min-w-0">
        {left}
      </div>
      <div className="min-w-0">
        {right}
      </div>
    </div>
  );
}

/**
 * DashboardThreeColumn - Three column layout for wider content
 */
interface DashboardThreeColumnProps {
  children: React.ReactNode;
  className?: string;
}

export function DashboardThreeColumn({ children, className }: DashboardThreeColumnProps) {
  return (
    <div className={cn("grid grid-cols-1 md:grid-cols-3 gap-6", className)}>
      {children}
    </div>
  );
}
