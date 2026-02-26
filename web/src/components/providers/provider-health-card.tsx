/**
 * Provider Health Card Component
 * Task 1.6: Provider Health Dashboard
 * Art Deco styling with brass/copper accents
 */

'use client';

import React from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Server,
  Activity,
  AlertTriangle,
  CheckCircle,
  XCircle,
  RefreshCw,
  Power,
  Zap,
  Clock,
  Shield,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { Provider, ProviderStatus, CircuitBreakerState } from '@/types';
import { ProviderStatusTimeline } from './provider-status-timeline';

interface ProviderHealthCardProps {
  provider: Provider;
  onRetry?: (providerId: string) => void;
  onToggle?: (providerId: string) => void;
  className?: string;
}

// Art Deco corner decoration component
const DecoCorner: React.FC<{ position: 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right' }> = ({ position }) => {
  const positionClasses = {
    'top-left': 'top-0 left-0 border-t-2 border-l-2',
    'top-right': 'top-0 right-0 border-t-2 border-r-2',
    'bottom-left': 'bottom-0 left-0 border-b-2 border-l-2',
    'bottom-right': 'bottom-0 right-0 border-b-2 border-r-2',
  };

  return (
    <div
      className={cn(
        'absolute w-4 h-4 border-[#B57D41] opacity-60',
        positionClasses[position]
      )}
    />
  );
};

// Status indicator with pulse animation
const StatusIndicator: React.FC<{ status: ProviderStatus }> = ({ status }) => {
  const statusConfig = {
    healthy: { color: 'bg-emerald-500', pulse: true, icon: CheckCircle },
    degraded: { color: 'bg-amber-500', pulse: true, icon: AlertTriangle },
    unhealthy: { color: 'bg-red-500', pulse: false, icon: XCircle },
    disabled: { color: 'bg-slate-400', pulse: false, icon: Power },
  };

  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <div className="flex items-center gap-2">
      <div className="relative">
        <div
          className={cn(
            'w-3 h-3 rounded-full',
            config.color,
            config.pulse && 'animate-pulse'
          )}
        />
        {config.pulse && (
          <div
            className={cn(
              'absolute inset-0 rounded-full animate-ping opacity-75',
              config.color
            )}
          />
        )}
      </div>
      <Icon className={cn('w-4 h-4', config.color.replace('bg-', 'text-'))} />
    </div>
  );
};

// Circuit breaker indicator
const CircuitBreakerIndicator: React.FC<{ state: CircuitBreakerState }> = ({ state }) => {
  const stateConfig = {
    closed: { color: 'bg-emerald-500', label: 'Closed', icon: Shield },
    open: { color: 'bg-red-500', label: 'Open', icon: AlertTriangle },
    'half-open': { color: 'bg-amber-500', label: 'Recovering', icon: Activity },
  };

  const config = stateConfig[state];
  const Icon = config.icon;

  return (
    <Badge
      variant="outline"
      className={cn(
        'gap-1.5 border-[#B87333]/30 text-[#B87333]',
        state === 'open' && 'border-red-500/30 text-red-600',
        state === 'half-open' && 'border-amber-500/30 text-amber-600'
      )}
    >
      <Icon className="w-3 h-3" />
      {config.label}
    </Badge>
  );
};

// Success rate progress bar
const SuccessRateBar: React.FC<{ rate: number }> = ({ rate }) => {
  const getColor = (r: number) => {
    if (r >= 95) return 'bg-emerald-500';
    if (r >= 80) return 'bg-amber-500';
    return 'bg-red-500';
  };

  return (
    <div className="space-y-1.5">
      <div className="flex justify-between text-xs">
        <span className="text-[#7A7F99]">Success Rate</span>
        <span className={cn(
          'font-medium',
          rate >= 95 ? 'text-emerald-600' : rate >= 80 ? 'text-amber-600' : 'text-red-600'
        )}>
          {rate.toFixed(1)}%
        </span>
      </div>
      <div className="h-1.5 bg-[#E2E8F0] rounded-full overflow-hidden">
        <div
          className={cn('h-full rounded-full transition-all duration-500', getColor(rate))}
          style={{ width: `${Math.min(rate, 100)}%` }}
        />
      </div>
    </div>
  );
};

// Provider icon based on name
const ProviderIcon: React.FC<{ name: string; className?: string }> = ({ name, className }) => {
  const lowerName = name.toLowerCase();
  
  if (lowerName.includes('openai')) {
    return (
      <div className={cn('w-10 h-10 rounded-lg bg-emerald-100 flex items-center justify-center', className)}>
        <span className="text-emerald-700 font-bold text-xs">OAI</span>
      </div>
    );
  }
  if (lowerName.includes('anthropic')) {
    return (
      <div className={cn('w-10 h-10 rounded-lg bg-orange-100 flex items-center justify-center', className)}>
        <span className="text-orange-700 font-bold text-xs">ANT</span>
      </div>
    );
  }
  if (lowerName.includes('gemini') || lowerName.includes('google')) {
    return (
      <div className={cn('w-10 h-10 rounded-lg bg-blue-100 flex items-center justify-center', className)}>
        <span className="text-blue-700 font-bold text-xs">GEM</span>
      </div>
    );
  }
  
  return (
    <div className={cn('w-10 h-10 rounded-lg bg-[#FDF8E8] flex items-center justify-center', className)}>
      <Server className="w-5 h-5 text-[#B57D41]" />
    </div>
  );
};

// Format timestamp
const formatTimestamp = (timestamp?: string): string => {
  if (!timestamp) return 'Never';
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  
  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return date.toLocaleDateString();
};

export const ProviderHealthCard: React.FC<ProviderHealthCardProps> = ({
  provider,
  onRetry,
  onToggle,
  className,
}) => {
  const isFailing = provider.status === 'unhealthy' || provider.status === 'degraded';
  const isDisabled = provider.status === 'disabled';

  // Generate sample 24h data for sparkline (in real app, this would come from API)
  const sparklineData = React.useMemo(() => {
    const baseRate = provider.errorRate24h;
    return Array.from({ length: 24 }, (_, i) => {
      const hour = new Date();
      hour.setHours(hour.getHours() - (23 - i));
      return {
        timestamp: hour.toISOString(),
        value: Math.max(0, Math.min(100, 100 - baseRate * 100 + (Math.random() - 0.5) * 20)),
      };
    });
  }, [provider.errorRate24h]);

  return (
    <Card
      className={cn(
        'relative overflow-hidden transition-all duration-300',
        'hover:shadow-lg hover:shadow-[#B57D41]/5',
        isFailing && 'border-red-200 bg-red-50/30',
        isDisabled && 'opacity-75',
        className
      )}
    >
      {/* Art Deco corners */}
      <DecoCorner position="top-left" />
      <DecoCorner position="top-right" />
      <DecoCorner position="bottom-left" />
      <DecoCorner position="bottom-right" />

      {/* Top accent line */}
      <div
        className={cn(
          'absolute top-0 left-0 right-0 h-0.5',
          provider.status === 'healthy' && 'bg-emerald-500',
          provider.status === 'degraded' && 'bg-amber-500',
          provider.status === 'unhealthy' && 'bg-red-500',
          provider.status === 'disabled' && 'bg-slate-400'
        )}
      />

      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <ProviderIcon name={provider.name} />
            <div>
              <CardTitle className="text-base font-semibold text-[#1E293B]">
                {provider.displayName || provider.name}
              </CardTitle>
              <CardDescription className="text-xs text-[#7A7F99] mt-0.5">
                {provider.models.length > 0 ? provider.models.join(', ') : 'No models configured'}
              </CardDescription>
            </div>
          </div>
          <StatusIndicator status={provider.status} />
        </div>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* Sparkline Timeline */}
        <div className="space-y-1.5">
          <div className="flex items-center justify-between text-xs text-[#7A7F99]">
            <span className="flex items-center gap-1">
              <Activity className="w-3 h-3" />
              24h Uptime
            </span>
            <span>{provider.requestCount24h.toLocaleString()} requests</span>
          </div>
          <ProviderStatusTimeline data={sparklineData} status={provider.status} />
        </div>

        {/* Metrics Grid */}
        <div className="grid grid-cols-2 gap-3">
          {/* Success Rate */}
          <div className="col-span-2">
            <SuccessRateBar rate={(1 - provider.errorRate24h) * 100} />
          </div>

          {/* Latency */}
          <div className="space-y-1">
            <div className="flex items-center gap-1.5 text-xs text-[#7A7F99]">
              <Zap className="w-3 h-3" />
              Latency
            </div>
            <div className="text-sm font-medium text-[#1E293B]">
              {provider.latencyMs ? `${provider.latencyMs}ms` : 'N/A'}
            </div>
          </div>

          {/* Circuit Breaker */}
          <div className="space-y-1">
            <div className="flex items-center gap-1.5 text-xs text-[#7A7F99]">
              <Shield className="w-3 h-3" />
              Circuit
            </div>
            <CircuitBreakerIndicator state={provider.circuitBreaker} />
          </div>
        </div>

        {/* Last Check */}
        <div className="flex items-center gap-1.5 text-xs text-[#7A7F99]">
          <Clock className="w-3 h-3" />
          <span>Last check: {formatTimestamp(provider.lastCheck)}</span>
        </div>

        {/* Error Count Badge */}
        {provider.errorRate24h > 0 && (
          <div className="flex items-center gap-2">
            <Badge
              variant="secondary"
              className={cn(
                'text-xs',
                provider.errorRate24h > 0.1
                  ? 'bg-red-100 text-red-700 hover:bg-red-100'
                  : 'bg-amber-100 text-amber-700 hover:bg-amber-100'
              )}
            >
              <AlertTriangle className="w-3 h-3 mr-1" />
              {Math.floor(provider.requestCount24h * provider.errorRate24h)} errors (24h)
            </Badge>
          </div>
        )}

        {/* Actions */}
        <div className="flex gap-2 pt-2 border-t border-[#E2E8F0]">
          <Button
            variant="outline"
            size="sm"
            className="flex-1 text-xs border-[#B57D41]/30 text-[#B57D41] hover:bg-[#B57D41]/5 hover:text-[#B57D41]"
            onClick={() => onRetry?.(provider.id)}
            disabled={isDisabled}
          >
            <RefreshCw className="w-3 h-3 mr-1.5" />
            Retry
          </Button>
          <Button
            variant="outline"
            size="sm"
            className={cn(
              'flex-1 text-xs',
              isDisabled
                ? 'border-emerald-500/30 text-emerald-600 hover:bg-emerald-50'
                : 'border-red-500/30 text-red-600 hover:bg-red-50'
            )}
            onClick={() => onToggle?.(provider.id)}
          >
            <Power className="w-3 h-3 mr-1.5" />
            {isDisabled ? 'Enable' : 'Disable'}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};

export default ProviderHealthCard;
