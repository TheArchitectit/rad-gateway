/**
 * RAD Gateway Admin UI - Real-time Metrics Hook
 * Real-time Integration Developer - Phase 5 Implementation
 *
 * Provides real-time metrics updates via Server-Sent Events.
 * Integrates with existing stores and handles reconnection gracefully.
 *
 * Pessimist's Note: Real-time data can go stale. We handle:
 * - Connection drops (data freshness)
 * - Reconnection storms (rate limiting)
 * - Memory bloat (event cleanup)
 * - Store synchronization (race conditions)
 */

import { useCallback, useEffect, useRef, useState, useMemo } from 'react';
import { useAuthStore } from '../stores/authStore';
import { useSSE, useSSEEvent } from './useSSE';
import type {
  RealtimeUsageMetrics,
  ProviderHealthUpdate,
  CircuitBreakerUpdate,
  SystemAlert,
  ProviderStatus,
  CircuitBreakerState,
} from '../types';

// ============================================================================
// Types
// ============================================================================

export interface RealtimeMetricsState {
  /** Whether connected to SSE stream */
  connected: boolean;

  /** Whether currently reconnecting */
  isReconnecting: boolean;

  /** Number of reconnection attempts */
  reconnectAttempts: number;

  /** Timestamp of last successful connection */
  lastConnectedAt: Date | null;

  /** Timestamp of last received event */
  lastEventAt: Date | null;

  /** Error if connection failed */
  error: Error | null;

  /** Data freshness - true if data is recent (< 30s) */
  isDataFresh: boolean;
}

export interface RealtimeMetricsData {
  /** Current usage metrics (last received) */
  usage: RealtimeUsageMetrics | null;

  /** Provider health updates by provider name */
  providerHealth: Map<string, ProviderHealthUpdate>;

  /** Circuit breaker states by provider name */
  circuitBreakers: Map<string, CircuitBreakerUpdate>;

  /** Recent system alerts (last 50) */
  alerts: SystemAlert[];

  /** Request rate history for charting */
  requestRateHistory: { timestamp: Date; value: number }[];

  /** Latency history for charting */
  latencyHistory: { timestamp: Date; value: number }[];
}

export interface RealtimeMetricsActions {
  /** Manually reconnect to SSE */
  reconnect: () => void;

  /** Disconnect from SSE */
  disconnect: () => void;

  /** Clear all cached data */
  clearData: () => void;

  /** Mark an alert as read */
  markAlertRead: (alertId: string) => void;

  /** Clear all alerts */
  clearAlerts: () => void;
}

export interface RealtimeMetricsReturn extends RealtimeMetricsState, RealtimeMetricsData, RealtimeMetricsActions {}

// ============================================================================
// Configuration
// ============================================================================

const DEFAULT_OPTIONS = {
  /** Maximum number of data points to keep in history */
  maxHistoryPoints: 60,

  /** Maximum number of alerts to keep */
  maxAlerts: 50,

  /** Data freshness threshold in ms */
  dataFreshnessThreshold: 30000,

  /** SSE reconnection options */
  reconnect: {
    enabled: true,
    initialDelayMs: 1000,
    maxDelayMs: 30000,
    backoffMultiplier: 2,
    maxAttempts: Infinity,
  },

  /** Heartbeat timeout - consider connection dead if no event in this time */
  heartbeatTimeoutMs: 45000,
};

type RealtimeOptions = Partial<typeof DEFAULT_OPTIONS>;

// ============================================================================
// useRealtimeMetrics - Main Hook
// ============================================================================

/**
 * Hook for subscribing to real-time gateway metrics via SSE.
 *
 * Features:
 * - Automatic connection management with auth token
 * - Data history tracking for charts
 * - Provider health monitoring
 * - Circuit breaker state tracking
 * - System alerts collection
 * - Data freshness detection
 *
 * @param options - Configuration options
 *
 * @example
 * ```tsx
 * function Dashboard() {
 *   const { connected, usage, providerHealth, error, reconnect } = useRealtimeMetrics({
 *     maxHistoryPoints: 60,
 *   });
 *
 *   if (error) {
 *     return <Alert onRetry={reconnect}>Connection failed</Alert>;
 *   }
 *
 *   return (
 *     <div>
 *       <ConnectionStatus connected={connected} />
 *       <MetricsCard data={usage} />
 *       <ProviderHealthGrid health={providerHealth} />
 *     </div>
 *   );
 * }
 * ```
 */
export function useRealtimeMetrics(options: RealtimeOptions = {}): RealtimeMetricsReturn {
  const opts = useMemo(() => ({ ...DEFAULT_OPTIONS, ...options }), [options]);

  const token = useAuthStore((state) => state.token);
  const isMountedRef = useRef(true);

  // Data state
  const [usage, setUsage] = useState<RealtimeUsageMetrics | null>(null);
  const [providerHealth, setProviderHealth] = useState<Map<string, ProviderHealthUpdate>>(new Map());
  const [circuitBreakers, setCircuitBreakers] = useState<Map<string, CircuitBreakerUpdate>>(new Map());
  const [alerts, setAlerts] = useState<SystemAlert[]>([]);
  const [requestRateHistory, setRequestRateHistory] = useState<{ timestamp: Date; value: number }[]>([]);
  const [latencyHistory, setLatencyHistory] = useState<{ timestamp: Date; value: number }[]>([]);

  // Connection state
  const [lastConnectedAt, setLastConnectedAt] = useState<Date | null>(null);
  const [lastEventAt, setLastEventAt] = useState<Date | null>(null);
  const [isDataFresh, setIsDataFresh] = useState(false);

  // Data freshness check
  useEffect(() => {
    if (!lastEventAt) {
      setIsDataFresh(false);
      return;
    }

    const checkFreshness = () => {
      const now = new Date();
      const diff = now.getTime() - lastEventAt.getTime();
      setIsDataFresh(diff < opts.dataFreshnessThreshold);
    };

    checkFreshness();
    const interval = setInterval(checkFreshness, 5000);

    return () => clearInterval(interval);
  }, [lastEventAt, opts.dataFreshnessThreshold]);

  // Setup SSE with multi-event subscription
  const { state, error, reconnectAttempts, isReconnecting, connect, disconnect } = useSSE(
    '/v0/admin/events',
    {
      events: ['usage:realtime', 'provider:health', 'provider:circuit', 'system:alert'],
      token,
      autoConnect: true,
      reconnect: opts.reconnect,
      heartbeatTimeoutMs: opts.heartbeatTimeoutMs,
      onOpen: () => {
        if (isMountedRef.current) {
          setLastConnectedAt(new Date());
        }
      },
      onMessage: (event) => {
        if (!isMountedRef.current) return;

        try {
          const parsed = JSON.parse(event.data);
          handleEvent(parsed);
        } catch (err) {
          console.error('[useRealtimeMetrics] Failed to parse event:', err);
        }
      },
      onError: (err) => {
        console.error('[useRealtimeMetrics] SSE error:', err);
      },
    }
  );

  // Handle incoming events
  const handleEvent = useCallback(
    (event: { type: string; payload: unknown }) => {
      setLastEventAt(new Date());

      switch (event.type) {
        case 'usage:realtime': {
          const data = event.payload as RealtimeUsageMetrics;
          setUsage(data);

          // Update history for charts
          setRequestRateHistory((prev) => {
            const newPoint = { timestamp: new Date(), value: data.requestsPerSecond };
            const newHistory = [...prev, newPoint];
            if (newHistory.length > opts.maxHistoryPoints) {
              return newHistory.slice(-opts.maxHistoryPoints);
            }
            return newHistory;
          });

          setLatencyHistory((prev) => {
            const newPoint = { timestamp: new Date(), value: data.latencyMs };
            const newHistory = [...prev, newPoint];
            if (newHistory.length > opts.maxHistoryPoints) {
              return newHistory.slice(-opts.maxHistoryPoints);
            }
            return newHistory;
          });
          break;
        }

        case 'provider:health': {
          const data = event.payload as ProviderHealthUpdate;
          setProviderHealth((prev) => {
            const next = new Map(prev);
            next.set(data.provider, data);
            return next;
          });
          break;
        }

        case 'provider:circuit': {
          const data = event.payload as CircuitBreakerUpdate;
          setCircuitBreakers((prev) => {
            const next = new Map(prev);
            next.set(data.provider, data);
            return next;
          });
          break;
        }

        case 'system:alert': {
          const data = event.payload as SystemAlert;
          setAlerts((prev) => {
            const newAlerts = [data, ...prev];
            if (newAlerts.length > opts.maxAlerts) {
              return newAlerts.slice(0, opts.maxAlerts);
            }
            return newAlerts;
          });
          break;
        }

        case 'heartbeat': {
          // Heartbeat received - connection is healthy
          break;
        }

        default:
          // Unknown event type - log but ignore
          console.warn('[useRealtimeMetrics] Unknown event type:', event.type);
      }
    },
    [opts.maxHistoryPoints, opts.maxAlerts]
  );

  // Cleanup on unmount
  useEffect(() => {
    isMountedRef.current = true;
    return () => {
      isMountedRef.current = false;
    };
  }, []);

  // Actions
  const clearData = useCallback(() => {
    setUsage(null);
    setProviderHealth(new Map());
    setCircuitBreakers(new Map());
    setAlerts([]);
    setRequestRateHistory([]);
    setLatencyHistory([]);
    setLastEventAt(null);
    setIsDataFresh(false);
  }, []);

  const markAlertRead = useCallback((alertId: string) => {
    // Currently alerts don't have a read flag, but we could add it
    // For now, just remove the alert
    setAlerts((prev) => prev.filter((a) => a.id !== alertId));
  }, []);

  const clearAlerts = useCallback(() => {
    setAlerts([]);
  }, []);

  return {
    // State
    connected: state === 'open',
    isReconnecting,
    reconnectAttempts,
    lastConnectedAt,
    lastEventAt,
    error,
    isDataFresh,

    // Data
    usage,
    providerHealth,
    circuitBreakers,
    alerts,
    requestRateHistory,
    latencyHistory,

    // Actions
    reconnect: connect,
    disconnect,
    clearData,
    markAlertRead,
    clearAlerts,
  };
}

// ============================================================================
// Specialized Hooks
// ============================================================================

/**
 * Hook for tracking a single real-time metric value.
 *
 * @param metric - Metric name to track
 * @param options - Configuration options
 *
 * @example
 * ```tsx
 * const { value, history } = useRealtimeMetric('requestsPerSecond');
 * return <Chart data={history} current={value} />;
 * ```
 */
export function useRealtimeMetric(
  metric: keyof RealtimeUsageMetrics,
  options: RealtimeOptions = {}
): {
  value: number | null;
  history: { timestamp: Date; value: number }[];
  connected: boolean;
  error: Error | null;
  reconnect: () => void;
} {
  const token = useAuthStore((state) => state.token);
  const [history, setHistory] = useState<{ timestamp: Date; value: number }[]>([]);
  const opts = useMemo(() => ({ ...DEFAULT_OPTIONS, ...options }), [options]);

  const { data, state, error, connect } = useSSEEvent<RealtimeUsageMetrics>(
    '/v0/admin/events',
    'usage:realtime',
    {
      token,
      autoConnect: true,
      reconnect: opts.reconnect,
      heartbeatTimeoutMs: opts.heartbeatTimeoutMs,
      onData: (data) => {
        const value = data[metric];
        if (typeof value === 'number') {
          setHistory((prev) => {
            const newHistory = [...prev, { timestamp: new Date(), value }];
            if (newHistory.length > opts.maxHistoryPoints) {
              return newHistory.slice(-opts.maxHistoryPoints);
            }
            return newHistory;
          });
        }
      },
    }
  );

  const currentValue = data ? (data[metric] as number) : null;

  return {
    value: currentValue,
    history,
    connected: state === 'open',
    error,
    reconnect: connect,
  };
}

/**
 * Hook for monitoring provider health status.
 *
 * @param provider - Provider name to monitor (optional - monitors all if not specified)
 *
 * @example
 * ```tsx
 * const { status, latency, isHealthy } = useProviderHealth('openai');
 * return <HealthIndicator status={status} latency={latency} />;
 * ```
 */
export function useProviderHealth(
  provider?: string
): {
  status: ProviderStatus;
  latencyMs: number;
  checkedAt: Date | null;
  isHealthy: boolean;
  connected: boolean;
  error: Error | null;
} {
  const token = useAuthStore((state) => state.token);
  const [health, setHealth] = useState<ProviderHealthUpdate | null>(null);

  const { state, error } = useSSE('/v0/admin/events', {
    events: ['provider:health'],
    token,
    autoConnect: true,
    onMessage: (event) => {
      try {
        const parsed = JSON.parse(event.data);
        if (parsed.type === 'provider:health') {
          const data = parsed.payload as ProviderHealthUpdate;
          if (!provider || data.provider === provider) {
            setHealth(data);
          }
        }
      } catch (err) {
        console.error('[useProviderHealth] Failed to parse event:', err);
      }
    },
  });

  return {
    status: health?.status || 'disabled',
    latencyMs: health?.latencyMs || 0,
    checkedAt: health ? new Date(health.checkedAt) : null,
    isHealthy: health?.status === 'healthy',
    connected: state === 'open',
    error,
  };
}

/**
 * Hook for monitoring circuit breaker states.
 *
 * @param provider - Provider name to monitor (optional - monitors all if not specified)
 *
 * @example
 * ```tsx
 * const { state, isOpen } = useCircuitBreaker('anthropic');
 * return <CircuitIndicator open={isOpen} />;
 * ```
 */
export function useCircuitBreaker(
  provider?: string
): {
  state: CircuitBreakerState;
  reason: string | undefined;
  isOpen: boolean;
  isClosed: boolean;
  isHalfOpen: boolean;
  connected: boolean;
} {
  const token = useAuthStore((state) => state.token);
  const [cbState, setCbState] = useState<CircuitBreakerUpdate | null>(null);

  const { state: connectionState } = useSSE('/v0/admin/events', {
    events: ['provider:circuit'],
    token,
    autoConnect: true,
    onMessage: (event) => {
      try {
        const parsed = JSON.parse(event.data);
        if (parsed.type === 'provider:circuit') {
          const data = parsed.payload as CircuitBreakerUpdate;
          if (!provider || data.provider === provider) {
            setCbState(data);
          }
        }
      } catch (err) {
        console.error('[useCircuitBreaker] Failed to parse event:', err);
      }
    },
  });

  return {
    state: cbState?.state || 'closed',
    reason: cbState?.reason,
    isOpen: cbState?.state === 'open',
    isClosed: cbState?.state === 'closed',
    isHalfOpen: cbState?.state === 'half-open',
    connected: connectionState === 'open',
  };
}

/**
 * Hook for receiving system alerts.
 *
 * @param options - Configuration options including severity filter
 *
 * @example
 * ```tsx
 * const { alerts, unreadCount, markRead } = useSystemAlerts({
 *   severity: ['warning', 'critical'],
 * });
 * return <AlertList alerts={alerts} unread={unreadCount} onMarkRead={markRead} />;
 * ```
 */
export function useSystemAlerts(
  options: {
    severity?: ('info' | 'warning' | 'critical')[];
    maxAlerts?: number;
  } = {}
): {
  alerts: SystemAlert[];
  unreadCount: number;
  markRead: (id: string) => void;
  markAllRead: () => void;
  connected: boolean;
} {
  const token = useAuthStore((state) => state.token);
  const { severity, maxAlerts = 50 } = options;
  const [alerts, setAlerts] = useState<SystemAlert[]>([]);
  const [readIds, setReadIds] = useState<Set<string>>(new Set());

  const { state } = useSSE('/v0/admin/events', {
    events: ['system:alert'],
    token,
    autoConnect: true,
    onMessage: (event) => {
      try {
        const parsed = JSON.parse(event.data);
        if (parsed.type === 'system:alert') {
          const data = parsed.payload as SystemAlert;

          // Filter by severity if specified
          if (severity && !severity.includes(data.severity)) {
            return;
          }

          setAlerts((prev) => {
            const newAlerts = [data, ...prev];
            if (newAlerts.length > maxAlerts) {
              return newAlerts.slice(0, maxAlerts);
            }
            return newAlerts;
          });
        }
      } catch (err) {
        console.error('[useSystemAlerts] Failed to parse event:', err);
      }
    },
  });

  const markRead = useCallback((id: string) => {
    setReadIds((prev) => new Set([...prev, id]));
  }, []);

  const markAllRead = useCallback(() => {
    setReadIds(new Set(alerts.map((a) => a.id)));
  }, [alerts]);

  const unreadCount = alerts.filter((a) => !readIds.has(a.id)).length;

  return {
    alerts,
    unreadCount,
    markRead,
    markAllRead,
    connected: state === 'open',
  };
}

// ============================================================================
// Utility Exports
// ============================================================================

/**
 * Utility to check if a metric value is within acceptable range.
 */
export function isMetricHealthy(
  metric: keyof RealtimeUsageMetrics,
  value: number,
  thresholds?: { warning: number; critical: number }
): 'healthy' | 'warning' | 'critical' {
  const defaultThresholds: Record<string, { warning: number; critical: number }> = {
    requestsPerSecond: { warning: 1000, critical: 5000 },
    latencyMs: { warning: 500, critical: 1000 },
    activeConnections: { warning: 100, critical: 500 },
  };

  const t = thresholds || defaultThresholds[metric];
  if (!t) return 'healthy';

  if (value >= t.critical) return 'critical';
  if (value >= t.warning) return 'warning';
  return 'healthy';
}

/**
 * Format metric value for display.
 */
export function formatMetric(
  metric: keyof RealtimeUsageMetrics,
  value: number
): string {
  switch (metric) {
    case 'requestsPerSecond':
      return `${value.toFixed(1)} req/s`;
    case 'latencyMs':
      return `${value.toFixed(0)} ms`;
    case 'activeConnections':
      return value.toString();
    default:
      return String(value);
  }
}
