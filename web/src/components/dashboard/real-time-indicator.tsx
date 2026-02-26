/**
 * RAD Gateway Admin UI - Real-time Indicator Component
 * Real-time Integration Developer - Phase 5 Implementation
 *
 * Connection status indicator for real-time dashboard metrics.
 * Shows connection state, last update timestamp, and reconnect button.
 *
 * Features:
 * - Animated connection status dot
 * - Last update timestamp
 * - Manual reconnect button
 * - Smooth transitions without flicker
 * - Art Deco aesthetic (Brass/Copper/Steel palette)
 */

import * as React from 'react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import { Wifi, WifiOff, RefreshCw } from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

export interface RealTimeIndicatorProps {
  /** Current connection state */
  connectionState: 'connecting' | 'open' | 'closed' | 'error';
  /** Whether currently reconnecting */
  isReconnecting: boolean;
  /** Last update timestamp */
  lastUpdate: Date | null;
  /** Seconds since last update */
  secondsSinceUpdate: number;
  /** Whether data is stale (> 10 seconds) */
  isStale: boolean;
  /** Reconnect function */
  onReconnect: () => void;
  /** Additional CSS classes */
  className?: string;
  /** Number of reconnection attempts */
  reconnectAttempts?: number;
  /** Connection error if any */
  error?: Error | null;
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Format seconds since update into human-readable string
 */
function formatTimeAgo(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}s ago`;
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes}m ago`;
  } else {
    const hours = Math.floor(seconds / 3600);
    return `${hours}h ago`;
  }
}

/**
 * Format last update time for tooltip
 */
function formatLastUpdate(date: Date): string {
  return date.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

// ============================================================================
// Component
// ============================================================================

export function RealTimeIndicator({
  connectionState,
  isReconnecting,
  lastUpdate,
  secondsSinceUpdate,
  isStale,
  onReconnect,
  className,
  reconnectAttempts = 0,
  error,
}: RealTimeIndicatorProps) {
  // Determine status color and icon
  const getStatusConfig = () => {
    switch (connectionState) {
      case 'open':
        return {
          color: 'bg-[var(--status-normal)]',
          glowColor: 'shadow-[0_0_8px_rgba(47,122,79,0.6)]',
          borderColor: 'border-[var(--status-normal)]',
          icon: Wifi,
          label: 'Live',
          description: 'Real-time updates active',
        };
      case 'connecting':
        return {
          color: 'bg-[var(--brass-500)]',
          glowColor: 'shadow-[0_0_8px_rgba(198,164,110,0.6)]',
          borderColor: 'border-[var(--brass-500)]',
          icon: RefreshCw,
          label: 'Connecting',
          description: 'Establishing connection...',
        };
      case 'error':
        return {
          color: 'bg-[var(--status-critical)]',
          glowColor: 'shadow-[0_0_8px_rgba(152,43,33,0.6)]',
          borderColor: 'border-[var(--status-critical)]',
          icon: WifiOff,
          label: 'Disconnected',
          description: error?.message || 'Connection lost',
        };
      case 'closed':
      default:
        return {
          color: 'bg-[var(--steel-500)]',
          glowColor: 'shadow-[0_0_8px_rgba(113,121,132,0.6)]',
          borderColor: 'border-[var(--steel-500)]',
          icon: WifiOff,
          label: 'Offline',
          description: 'Not connected to stream',
        };
    }
  };

  const status = getStatusConfig();
  const StatusIcon = status.icon;

  return (
    <TooltipProvider>
      <div
        className={cn(
          'flex items-center gap-3 rounded-lg border border-[var(--line-soft)]',
          'bg-[var(--panel-bg)]/80 px-3 py-2 backdrop-blur-sm',
          'transition-all duration-300',
          className
        )}
      >
        {/* Connection Status Badge with Tooltip */}
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge
              variant="outline"
              className={cn(
                'cursor-help border-[var(--line-soft)] bg-transparent',
                'flex items-center gap-2 px-2.5 py-1',
                status.borderColor
              )}
            >
              {/* Animated Status Dot */}
              <span
                className={cn(
                  'relative flex h-2.5 w-2.5',
                  connectionState === 'connecting' && 'animate-pulse'
                )}
              >
                {/* Outer glow ring */}
                <span
                  className={cn(
                    'absolute inline-flex h-full w-full rounded-full opacity-75',
                    status.color,
                    connectionState === 'open' && 'animate-ping'
                  )}
                />
                {/* Inner dot */}
                <span
                  className={cn(
                    'relative inline-flex h-2.5 w-2.5 rounded-full',
                    status.color,
                    status.glowColor
                  )}
                />
              </span>

              {/* Status Label */}
              <span className="text-xs font-medium text-[var(--ink-700)]">
                {status.label}
              </span>

              {/* Status Icon */}
              <StatusIcon
                className={cn(
                  'h-3.5 w-3.5',
                  status.color.replace('bg-', 'text-')
                )}
              />
            </Badge>
          </TooltipTrigger>
          <TooltipContent side="bottom" className="max-w-xs">
            <div className="space-y-1">
              <p className="font-medium">{status.description}</p>
              {reconnectAttempts > 0 && (
                <p className="text-xs text-[var(--ink-500)]">
                  Reconnection attempt {reconnectAttempts}
                </p>
              )}
              {lastUpdate && (
                <p className="text-xs text-[var(--ink-500)]">
                  Last update: {formatLastUpdate(lastUpdate)}
                </p>
              )}
            </div>
          </TooltipContent>
        </Tooltip>

        {/* Last Update Timestamp */}
        <div className="flex items-center gap-1.5 text-xs text-[var(--ink-500)]">
          <span className="hidden sm:inline">Updated</span>
          <span
            className={cn(
              'font-medium transition-colors duration-300',
              isStale && 'text-[var(--status-warning)]'
            )}
          >
            {lastUpdate ? formatTimeAgo(secondsSinceUpdate) : 'never'}
          </span>
        </div>

        {/* Reconnect Button (only show when disconnected or error) */}
        {(connectionState === 'error' || connectionState === 'closed') && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                onClick={onReconnect}
                disabled={isReconnecting}
                className={cn(
                  'h-7 px-2',
                  'text-[var(--brass-500)] hover:text-[var(--brass-600)]',
                  'hover:bg-[var(--brass-500)]/10'
                )}
              >
                <RefreshCw
                  className={cn(
                    'h-3.5 w-3.5',
                    isReconnecting && 'animate-spin'
                  )}
                />
                <span className="ml-1 text-xs">
                  {isReconnecting ? 'Retrying...' : 'Reconnect'}
                </span>
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">
              <p>Manually reconnect to real-time stream</p>
            </TooltipContent>
          </Tooltip>
        )}

        {/* Art Deco decorative corner element */}
        <div className="absolute -top-px -right-px w-6 h-6 opacity-20 pointer-events-none">
          <div className="absolute top-0 right-0 w-4 h-4 border-t border-r border-[var(--brass-500)]" />
        </div>
      </div>
    </TooltipProvider>
  );
}

export default RealTimeIndicator;
