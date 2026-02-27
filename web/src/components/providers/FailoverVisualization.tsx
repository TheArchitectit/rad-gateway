/**
 * Failover Visualization Component
 * Sprint 6.3: Provider Fail-over Visualization
 *
 * Visualizes the failover chain between providers in a topology view
 */

'use client';

import React from 'react';
import {
  ArrowRight,
  Server,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Activity,
  Zap,
  Shield,
  RefreshCw,
  PauseCircle,
} from 'lucide-react';
import { Card } from '@/components/atoms/Card';
import { Badge } from '@/components/atoms/Badge';
import { cn } from '@/lib/utils';
import type { Provider, ProviderStatus, CircuitBreakerState } from '@/types';

interface FailoverChain {
  primary: Provider;
  fallbacks: Provider[];
  activeProvider: string;
  lastFailoverAt?: string;
  failoverCount24h: number;
}

interface FailoverVisualizationProps {
  chains: FailoverChain[];
  className?: string;
}

// Status configuration
const statusConfig: Record<ProviderStatus, {
  color: 'success' | 'warning' | 'error' | 'default';
  icon: React.ElementType;
  label: string;
}> = {
  healthy: {
    color: 'success',
    icon: CheckCircle,
    label: 'Healthy',
  },
  degraded: {
    color: 'warning',
    icon: AlertTriangle,
    label: 'Degraded',
  },
  unhealthy: {
    color: 'error',
    icon: XCircle,
    label: 'Unhealthy',
  },
  disabled: {
    color: 'default',
    icon: PauseCircle,
    label: 'Disabled',
  },
};

// Circuit breaker configuration
const circuitConfig: Record<CircuitBreakerState, {
  color: 'success' | 'warning' | 'error';
  label: string;
  description: string;
}> = {
  closed: {
    color: 'success',
    label: 'Closed',
    description: 'Normal operation',
  },
  open: {
    color: 'error',
    label: 'Open',
    description: 'Failing fast',
  },
  'half-open': {
    color: 'warning',
    label: 'Half-Open',
    description: 'Testing recovery',
  },
};

// Provider node component
const ProviderNode: React.FC<{
  provider: Provider;
  isActive: boolean;
  isPrimary: boolean;
  position: 'start' | 'middle' | 'end';
}> = ({ provider, isActive, isPrimary, position }) => {
  const status = statusConfig[provider.status];
  const circuit = circuitConfig[provider.circuitBreaker];
  const StatusIcon = status.icon;

  return (
    <div
      className={cn(
        'relative flex flex-col items-center p-4 rounded-lg border-2 transition-all duration-300',
        isActive
          ? 'border-[var(--brass-500)] bg-[var(--brass-500)]/5 shadow-lg'
          : 'border-[var(--line-soft)] bg-[var(--panel)]',
        provider.status === 'unhealthy' && 'border-red-300 bg-red-50/30',
        provider.status === 'disabled' && 'opacity-60'
      )}
    >
      {/* Position indicator */}
      <div className="absolute -top-3 left-1/2 -translate-x-1/2">
        <span
          className={cn(
            'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
            isPrimary
              ? 'bg-[var(--brass-500)] text-white'
              : 'bg-[var(--panel)] text-[var(--ink-500)] border border-[var(--line-soft)]'
          )}
        >
          {isPrimary ? 'Primary' : `Fallback ${position === 'middle' ? '1' : '2'}`}
        </span>
      </div>

      {/* Provider Icon */}
      <div
        className={cn(
          'w-12 h-12 rounded-full flex items-center justify-center mb-3',
          status.color === 'success' && 'bg-emerald-50',
          status.color === 'warning' && 'bg-amber-50',
          status.color === 'error' && 'bg-red-50',
          status.color === 'default' && 'bg-slate-50'
        )}
      >
        <StatusIcon
          className={cn(
            'w-6 h-6',
            status.color === 'success' && 'text-emerald-600',
            status.color === 'warning' && 'text-amber-600',
            status.color === 'error' && 'text-red-600',
            status.color === 'default' && 'text-slate-400'
          )}
        />
      </div>

      {/* Provider Name */}
      <h4 className="font-medium text-[var(--ink-900)] text-center text-sm">
        {provider.displayName || provider.name}
      </h4>

      {/* Status Badge */}
      <div className="mt-2">
        <Badge color={status.color}>{status.label}</Badge>
      </div>

      {/* Circuit Breaker Badge */}
      <div className="mt-1">
        <Badge color={circuit.color} size="sm">
          <Shield className="w-3 h-3 mr-1" />
          {circuit.label}
        </Badge>
      </div>

      {/* Metrics */}
      <div className="mt-3 text-xs text-[var(--ink-500)] space-y-1">
        {provider.latencyMs && provider.latencyMs > 0 && (
          <div className="flex items-center gap-1">
            <Zap className="w-3 h-3" />
            <span>{provider.latencyMs}ms</span>
          </div>
        )}
        <div className="flex items-center gap-1">
          <Activity className="w-3 h-3" />
          <span>{(provider.requestCount24h || 0).toLocaleString()} req/24h</span>
        </div>
      </div>

      {/* Active indicator */}
      {isActive && (
        <div className="absolute -bottom-2 left-1/2 -translate-x-1/2">
          <span className="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium bg-[var(--brass-500)] text-white">
            <RefreshCw className="w-3 h-3 mr-1 animate-spin" />
            Active
          </span>
        </div>
      )}
    </div>
  );
};

// Connection arrow component
const ConnectionArrow: React.FC<{
  isActive: boolean;
  isFailing: boolean;
  failoverReason?: string | undefined;
}> = ({ isActive, isFailing, failoverReason }) => {
  return (
    <div className="relative flex flex-col items-center justify-center px-2">
      {/* Animated connection line */}
      <div
        className={cn(
          'w-8 h-0.5 rounded-full transition-all duration-500',
          isActive
            ? 'bg-[var(--brass-500)] animate-pulse'
            : isFailing
            ? 'bg-red-400'
            : 'bg-[var(--line-soft)]'
        )}
      />

      {/* Arrow */}
      <ArrowRight
        className={cn(
          'w-5 h-5 -mt-2.5 transition-colors duration-300',
          isActive
            ? 'text-[var(--brass-500)]'
            : isFailing
            ? 'text-red-400'
            : 'text-[var(--ink-300)]'
        )}
      />

      {/* Failover reason tooltip */}
      {isFailing && failoverReason && (
        <div className="absolute top-6 bg-red-50 text-red-700 text-xs px-2 py-1 rounded border border-red-200 whitespace-nowrap">
          {failoverReason}
        </div>
      )}
    </div>
  );
};

// Failover chain component
const FailoverChainComponent: React.FC<{
  chain: FailoverChain;
  index: number;
}> = ({ chain, index }) => {
  const { primary, fallbacks, activeProvider, lastFailoverAt, failoverCount24h } = chain;

  // Format last failover time
  const formatLastFailover = (timestamp?: string): string => {
    if (!timestamp) return 'No recent failovers';
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

  return (
    <Card className="overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-[var(--line-soft)] bg-[var(--panel)]/50">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Server className="w-5 h-5 text-[var(--brass-500)]" />
            <h3 className="font-medium text-[var(--ink-900)]">
              Failover Chain {index + 1}
            </h3>
          </div>
          <div className="flex items-center gap-3 text-sm text-[var(--ink-500)]">
            <span>Last failover: {formatLastFailover(lastFailoverAt)}</span>
            {failoverCount24h > 0 && (
              <Badge
                color={failoverCount24h > 5 ? 'error' : 'warning'}
                size="sm"
              >
                <AlertTriangle className="w-3 h-3 mr-1" />
                {failoverCount24h} failovers (24h)
              </Badge>
            )}
          </div>
        </div>
      </div>

      {/* Chain Visualization */}
      <div className="p-6">
        <div className="flex items-center justify-center gap-4 flex-wrap">
          {/* Primary */}
          <ProviderNode
            provider={primary}
            isActive={activeProvider === primary.id}
            isPrimary={true}
            position="start"
          />

          {/* Connection to first fallback */}
          {fallbacks.length > 0 && (
            <ConnectionArrow
              isActive={activeProvider === fallbacks[0]?.id}
              isFailing={primary.status === 'unhealthy'}
              {...(primary.status === 'unhealthy' ? { failoverReason: 'Primary failing' } : {})}
            />
          )}

          {/* Fallbacks */}
          {fallbacks.map((fallback, idx) => (
            <React.Fragment key={fallback.id}>
              <ProviderNode
                provider={fallback}
                isActive={activeProvider === fallback.id}
                isPrimary={false}
                position={idx === fallbacks.length - 1 ? 'end' : 'middle'}
              />
              {idx < fallbacks.length - 1 && (
                <ConnectionArrow
                  isActive={activeProvider === fallbacks[idx + 1]?.id}
                  isFailing={
                    fallback.status === 'unhealthy' &&
                    activeProvider === fallbacks[idx + 1]?.id
                  }
                  {...(fallback.status === 'unhealthy' ? { failoverReason: 'Fallback failing' } : {})}
                />
              )}
            </React.Fragment>
          ))}
        </div>

        {/* Legend */}
        <div className="mt-6 pt-4 border-t border-[var(--line-soft)]">
          <div className="flex flex-wrap items-center justify-center gap-4 text-xs text-[var(--ink-500)]">
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded-full bg-emerald-500" />
              <span>Healthy</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded-full bg-amber-500" />
              <span>Degraded</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded-full bg-red-500" />
              <span>Unhealthy</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-8 h-0.5 bg-[var(--brass-500)]" />
              <span>Active Traffic</span>
            </div>
          </div>
        </div>
      </div>
    </Card>
  );
};

// Main component
export const FailoverVisualization: React.FC<FailoverVisualizationProps> = ({
  chains,
  className,
}) => {
  if (chains.length === 0) {
    return (
      <Card className={cn('p-8 text-center', className)}>
        <Server className="w-12 h-12 text-[var(--ink-300)] mx-auto mb-4" />
        <h3 className="text-lg font-medium text-[var(--ink-700)] mb-2">
          No Failover Chains Configured
        </h3>
        <p className="text-sm text-[var(--ink-500)]">
          Configure provider fallbacks to see the failover topology.
        </p>
      </Card>
    );
  }

  return (
    <div className={cn('space-y-6', className)}>
      {chains.map((chain, index) => (
        <FailoverChainComponent key={chain.primary.id} chain={chain} index={index} />
      ))}
    </div>
  );
};

export default FailoverVisualization;
