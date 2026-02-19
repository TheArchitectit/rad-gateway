import React from 'react';
import { Badge } from '../atoms/Badge';

type Status = 'healthy' | 'degraded' | 'unhealthy' | 'pending' | 'active' | 'inactive' | 'closed' | 'open' | 'half-open';

interface StatusBadgeProps {
  status: Status;
  showPulse?: boolean;
}

const statusConfig: Record<Status, { color: 'success' | 'warning' | 'error' | 'info' | 'default'; label: string }> = {
  healthy: { color: 'success', label: 'Healthy' },
  degraded: { color: 'warning', label: 'Degraded' },
  unhealthy: { color: 'error', label: 'Unhealthy' },
  pending: { color: 'info', label: 'Pending' },
  active: { color: 'success', label: 'Active' },
  inactive: { color: 'default', label: 'Inactive' },
  closed: { color: 'success', label: 'Closed' },
  open: { color: 'error', label: 'Open' },
  'half-open': { color: 'warning', label: 'Half-Open' },
};

export function StatusBadge({ status, showPulse = false }: StatusBadgeProps) {
  const config = statusConfig[status];

  return (
    <div className="flex items-center gap-2">
      {showPulse && status === 'healthy' && (
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
          <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
        </span>
      )}
      <Badge color={config.color}>{config.label}</Badge>
    </div>
  );
}
