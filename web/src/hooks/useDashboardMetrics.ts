/**
 * RAD Gateway Admin UI - Dashboard Metrics Hook
 * Real-time Integration Developer - Phase 5 Implementation
 *
 * Provides real-time dashboard metrics via SSE with React Query integration.
 * Wraps useSSE for dashboard-specific metrics streaming.
 *
 * Features:
 * - Real-time metrics streaming via SSE
 * - Automatic React Query cache updates
 * - Smooth data transitions without flicker
 * - Connection state management
 * - Reconnection handling
 *
 * Art Deco Design System: Brass/Copper/Steel palette
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useSSEEvent } from './useSSE';
import { usageKeys } from '@/queries/keys';

// ============================================================================
// Types
// ============================================================================

export interface DashboardMetrics {
  totalRequests: number;
  avgLatency: number;
  errorRate: number;
  activeProviders: number;
}

export interface DashboardMetricsEvent {
  type: 'dashboard_metrics';
  timestamp: string;
  data: DashboardMetrics;
}

export interface DashboardMetricsChange {
  totalRequests: { value: number; positive: boolean } | null;
  avgLatency: { value: number; positive: boolean } | null;
  errorRate: { value: number; positive: boolean } | null;
  activeProviders: { value: number; positive: boolean } | null;
}

export interface UseDashboardMetricsReturn {
  /** Current metrics data */
  metrics: DashboardMetrics | null;
  /** Previous metrics for comparison */
  previousMetrics: DashboardMetrics | null;
  /** Calculated changes between updates */
  changes: DashboardMetricsChange | null;
  /** Connection state */
  connectionState: 'connecting' | 'open' | 'closed' | 'error';
  /** Connection error if any */
  error: Error | null;
  /** Whether currently reconnecting */
  isReconnecting: boolean;
  /** Number of reconnection attempts */
  reconnectAttempts: number;
  /** Last update timestamp */
  lastUpdate: Date | null;
  /** Time since last update in seconds */
  secondsSinceUpdate: number;
  /** Whether data is stale (> 10 seconds) */
  isStale: boolean;
  /** Manually reconnect */
  reconnect: () => void;
  /** Disconnect from stream */
  disconnect: () => void;
  /** Whether initial data has been received */
  hasReceivedData: boolean;
}

// ============================================================================
// Configuration
// ============================================================================

const SSE_ENDPOINT = '/v0/admin/events';
const METRICS_EVENT_TYPE = 'dashboard:metrics';
const STALE_THRESHOLD_MS = 10000; // 10 seconds

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Calculate percentage change between two values
 * Returns null if previous is 0 to avoid division by zero
 */
function calculateChange(current: number, previous: number): { value: number; positive: boolean } | null {
  if (previous === 0 || previous === undefined) return null;
  const change = ((current - previous) / previous) * 100;
  return {
    value: Math.abs(parseFloat(change.toFixed(1))),
    positive: change >= 0,
  };
}

/**
 * Calculate all metric changes
 */
function calculateChanges(
  current: DashboardMetrics | null,
  previous: DashboardMetrics | null
): DashboardMetricsChange | null {
  if (!current || !previous) return null;

  return {
    totalRequests: calculateChange(current.totalRequests, previous.totalRequests),
    avgLatency: calculateChange(current.avgLatency, previous.avgLatency),
    errorRate: calculateChange(current.errorRate, previous.errorRate),
    activeProviders: calculateChange(current.activeProviders, previous.activeProviders),
  };
}

// ============================================================================
// Hook
// ============================================================================

export function useDashboardMetrics(): UseDashboardMetricsReturn {
  const queryClient = useQueryClient();
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [previousMetrics, setPreviousMetrics] = useState<DashboardMetrics | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const [secondsSinceUpdate, setSecondsSinceUpdate] = useState(0);
  const [hasReceivedData, setHasReceivedData] = useState(false);

  // Ref to track previous metrics for change calculation
  const metricsRef = useRef<DashboardMetrics | null>(null);

  // Handle incoming SSE data
  const handleData = useCallback((data: DashboardMetrics) => {
    // Store previous for change calculation
    if (metricsRef.current) {
      setPreviousMetrics(metricsRef.current);
    }

    // Update current metrics
    metricsRef.current = data;
    setMetrics(data);
    setLastUpdate(new Date());
    setHasReceivedData(true);

    // Update React Query cache with new data
    // This ensures other components using useUsageSummary get updated data
    queryClient.setQueryData(usageKeys.summary(), (oldData: unknown) => {
      if (!oldData) return oldData;
      return {
        ...oldData,
        totalRequests: data.totalRequests,
        avgDurationMs: data.avgLatency,
        errorRate: data.errorRate,
      };
    });

  }, [queryClient]);

  // Use SSE hook for dashboard metrics
  const {
    data: sseData,
    state,
    error,
    isReconnecting,
    reconnectAttempts,
    connect,
    disconnect,
  } = useSSEEvent<DashboardMetricsEvent>(SSE_ENDPOINT, METRICS_EVENT_TYPE, {
    autoConnect: true,
    reconnect: {
      enabled: true,
      initialDelayMs: 1000,
      maxDelayMs: 30000,
      backoffMultiplier: 2,
      maxAttempts: Infinity,
    },
    heartbeatTimeoutMs: 15000, // 15 seconds for dashboard metrics
    onData: (event) => {
      if (event?.data) {
        handleData(event.data);
      }
    },
  });

  // Process SSE data when received
  useEffect(() => {
    if (sseData?.data) {
      handleData(sseData.data);
    }
  }, [sseData, handleData]);

  // Update seconds since last update every second
  useEffect(() => {
    const interval = setInterval(() => {
      if (lastUpdate) {
        const seconds = Math.floor((Date.now() - lastUpdate.getTime()) / 1000);
        setSecondsSinceUpdate(seconds);
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [lastUpdate]);

  // Calculate changes
  const changes = calculateChanges(metrics, previousMetrics);

  // Determine if data is stale
  const isStale = lastUpdate ? Date.now() - lastUpdate.getTime() > STALE_THRESHOLD_MS : false;

  return {
    metrics,
    previousMetrics,
    changes,
    connectionState: state,
    error,
    isReconnecting,
    reconnectAttempts,
    lastUpdate,
    secondsSinceUpdate,
    isStale,
    reconnect: connect,
    disconnect,
    hasReceivedData,
  };
}

export default useDashboardMetrics;
