'use client';

import * as React from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import { TrendingUp, TrendingDown, LucideIcon } from 'lucide-react';

export interface StatCardProps {
  title: string;
  value: string | number;
  change?: {
    value: number;
    positive: boolean;
  };
  icon: LucideIcon;
  trend?: 'up' | 'down' | 'neutral';
  description?: string;
  isLoading?: boolean;
  className?: string;
}

export function StatCard({
  title,
  value,
  change,
  icon: Icon,
  trend = 'neutral',
  description,
  isLoading = false,
  className,
}: StatCardProps) {
  // Determine trend from change if not explicitly provided
  const computedTrend = trend === 'neutral' && change 
    ? (change.positive ? 'up' : 'down') 
    : trend;

  if (isLoading) {
    return (
      <Card className={cn(
        "relative overflow-hidden border-[var(--line-soft)] bg-gradient-to-br from-[var(--panel-bg)] to-[var(--panel-bg)]/50",
        className
      )}>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-8 w-8 rounded-lg" />
          </div>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-8 w-32 mb-2" />
          <Skeleton className="h-3 w-20" />
        </CardContent>
        {/* Art Deco decorative corner */}
        <div className="absolute top-0 right-0 w-16 h-16 opacity-10">
          <div className="absolute top-2 right-2 w-8 h-8 border-t-2 border-r-2 border-[var(--brass-500)]" />
        </div>
      </Card>
    );
  }

  return (
    <Card className={cn(
      "relative overflow-hidden border-[var(--line-soft)] bg-gradient-to-br from-[var(--panel-bg)] to-[var(--panel-bg)]/50 transition-all duration-300 hover:shadow-lg hover:border-[var(--brass-500)]/30 group",
      className
    )}>
      {/* Art Deco decorative elements */}
      <div className="absolute top-0 right-0 w-16 h-16 opacity-20 transition-opacity duration-300 group-hover:opacity-40">
        <div className="absolute top-2 right-2 w-8 h-8 border-t-2 border-r-2 border-[var(--brass-500)]" />
        <div className="absolute top-4 right-4 w-4 h-4 border-t border-r border-[var(--brass-500)]" />
      </div>
      
      {/* Bottom left decorative element */}
      <div className="absolute bottom-0 left-0 w-12 h-12 opacity-10 transition-opacity duration-300 group-hover:opacity-25">
        <div className="absolute bottom-2 left-2 w-6 h-6 border-b-2 border-l-2 border-[var(--copper-500)]" />
      </div>

      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium text-[var(--ink-500)] tracking-wide uppercase">
            {title}
          </CardTitle>
          <div className={cn(
            "flex h-10 w-10 items-center justify-center rounded-lg transition-all duration-300",
            computedTrend === 'up' && "bg-[var(--status-normal)]/10 text-[var(--status-normal)]",
            computedTrend === 'down' && "bg-[var(--status-critical)]/10 text-[var(--status-critical)]",
            computedTrend === 'neutral' && "bg-[var(--brass-500)]/10 text-[var(--brass-500)]"
          )}>
            <Icon className="h-5 w-5" />
          </div>
        </div>
      </CardHeader>
      
      <CardContent className="pt-0">
        <div className="flex items-baseline gap-2">
          <span className="text-3xl font-bold text-[var(--ink-900)] tracking-tight">
            {value}
          </span>
          
          {change && (
            <span className={cn(
              "inline-flex items-center gap-0.5 text-sm font-semibold",
              change.positive ? "text-[var(--status-normal)]" : "text-[var(--status-critical)]"
            )}>
              {change.positive ? (
                <TrendingUp className="h-3.5 w-3.5" />
              ) : (
                <TrendingDown className="h-3.5 w-3.5" />
              )}
              {change.positive ? '+' : ''}{change.value}%
            </span>
          )}
        </div>
        
        {description && (
          <p className="mt-1 text-xs text-[var(--ink-500)]">
            {description}
          </p>
        )}
      </CardContent>

      {/* Progress bar indicator */}
      <div className="absolute bottom-0 left-0 right-0 h-1 bg-gradient-to-r from-[var(--brass-500)] via-[var(--copper-500)] to-[var(--steel-500)] opacity-50" />
    </Card>
  );
}
