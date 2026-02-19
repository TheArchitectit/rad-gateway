/**
 * RAD Gateway Admin UI - Usage Hooks
 * State Management Engineer - Phase 2 Implementation
 *
 * Custom hooks for usage data and analytics.
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import { adminAPI } from '../api/client';
import {
  UsageFilters,
  UsageRecord,
  UsageMetrics,
  TimeSeriesData,
  PaginatedResponse,
} from '../types';

interface UseUsageReturn {
  usage: UsageRecord[];
  pagination: PaginatedResponse<unknown>['pagination'] | null;
  isLoading: boolean;
  error: string | null;
  hasMore: boolean;
  fetch: (filters?: UsageFilters) => Promise<void>;
  fetchMore: () => Promise<void>;
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch and manage usage data.
 */
export function useUsage(initialFilters?: UsageFilters): UseUsageReturn {
  const [usage, setUsage] = useState<UsageRecord[]>([]);
  const [pagination, setPagination] = useState<PaginatedResponse<unknown>['pagination'] | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<UsageFilters | undefined>(initialFilters);

  const fetchUsage = useCallback(
    async (cursor?: string, merge: boolean = false) => {
      setIsLoading(true);
      setError(null);

      try {
        const params = {
          ...filters,
          ...(cursor && { cursor }),
          limit: 50,
        };

        const response = await adminAPI.getLogs(params);

        if (merge) {
          setUsage((prev) => [...prev, ...response.data]);
        } else {
          setUsage(response.data);
        }
        setPagination(response.pagination);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to fetch usage data';
        setError(message);
      } finally {
        setIsLoading(false);
      }
    },
    [filters]
  );

  useEffect(() => {
    fetchUsage();
  }, [fetchUsage]);

  const fetch = useCallback(
    async (newFilters?: UsageFilters) => {
      setFilters(newFilters);
      await fetchUsage(undefined, false);
    },
    [fetchUsage]
  );

  const fetchMore = useCallback(async () => {
    if (pagination?.hasMore && pagination?.cursor) {
      await fetchUsage(pagination.cursor, true);
    }
  }, [pagination, fetchUsage]);

  const refresh = useCallback(async () => {
    await fetchUsage(undefined, false);
  }, [fetchUsage]);

  return {
    usage,
    pagination,
    isLoading,
    error,
    hasMore: pagination?.hasMore ?? false,
    fetch,
    fetchMore,
    refresh,
  };
}

interface UseUsageMetricsReturn {
  metrics: UsageMetrics | null;
  isLoading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

/**
 * Hook to calculate usage metrics from usage data.
 */
export function useUsageMetrics(filters?: UsageFilters): UseUsageMetricsReturn {
  const { usage, isLoading, error, refresh } = useUsage(filters);

  const metrics = useMemo((): UsageMetrics | null => {
    if (usage.length === 0) return null;

    const totalRequests = usage.length;
    const totalTokens = usage.reduce((sum, record) => sum + record.usage.totalTokens, 0);
    const totalCost = usage.reduce((sum, record) => sum + record.usage.costTotal, 0);
    const totalLatency = usage.reduce((sum, record) => sum + record.durationMs, 0);
    const errorCount = usage.filter((record) => record.responseStatus === 'error').length;

    return {
      totalRequests,
      totalTokens,
      totalCost,
      averageLatency: totalLatency / totalRequests,
      errorRate: errorCount / totalRequests,
      requestsPerSecond: totalRequests / 3600, // Assuming hourly data
    };
  }, [usage]);

  return {
    metrics,
    isLoading,
    error,
    refresh,
  };
}

interface UseUsageTimeSeriesReturn {
  requests: TimeSeriesData[];
  tokens: TimeSeriesData[];
  cost: TimeSeriesData[];
  latency: TimeSeriesData[];
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook to get usage data as time series for charts.
 */
export function useUsageTimeSeries(
  timeRange: 'hour' | 'day' | 'week' | 'month' = 'day'
): UseUsageTimeSeriesReturn {
  const { usage, isLoading, error } = useUsage();

  const timeSeries = useMemo(() => {
    const grouped = new Map<string, { requests: number; tokens: number; cost: number; latency: number }>();

    usage.forEach((record) => {
      const date = new Date(record.timestamp);
      let key: string;

      switch (timeRange) {
        case 'hour':
          key = date.toISOString().slice(0, 13); // YYYY-MM-DDTHH
          break;
        case 'day':
          key = date.toISOString().slice(0, 10); // YYYY-MM-DD
          break;
        case 'week':
          const weekStart = new Date(date);
          weekStart.setDate(date.getDate() - date.getDay());
          key = weekStart.toISOString().slice(0, 10);
          break;
        case 'month':
          key = date.toISOString().slice(0, 7); // YYYY-MM
          break;
      }

      const existing = grouped.get(key) || { requests: 0, tokens: 0, cost: 0, latency: 0 };
      existing.requests++;
      existing.tokens += record.usage.totalTokens;
      existing.cost += record.usage.costTotal;
      existing.latency += record.durationMs;
      grouped.set(key, existing);
    });

    const sortedKeys = Array.from(grouped.keys()).sort();

    return {
      requests: sortedKeys.map((key) => ({
        timestamp: key,
        value: grouped.get(key)!.requests,
      })),
      tokens: sortedKeys.map((key) => ({
        timestamp: key,
        value: grouped.get(key)!.tokens,
      })),
      cost: sortedKeys.map((key) => ({
        timestamp: key,
        value: grouped.get(key)!.cost,
      })),
      latency: sortedKeys.map((key) => ({
        timestamp: key,
        value: Math.round(grouped.get(key)!.latency / grouped.get(key)!.requests),
      })),
    };
  }, [usage, timeRange]);

  return {
    requests: timeSeries.requests,
    tokens: timeSeries.tokens,
    cost: timeSeries.cost,
    latency: timeSeries.latency,
    isLoading,
    error,
  };
}

interface UseUsageByProviderReturn {
  byProvider: Record<string, { requests: number; tokens: number; cost: number }>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook to get usage grouped by provider.
 */
export function useUsageByProvider(): UseUsageByProviderReturn {
  const { usage, isLoading, error } = useUsage();

  const byProvider = useMemo(() => {
    return usage.reduce(
      (acc, record) => {
        const provider = record.provider;
        if (!acc[provider]) {
          acc[provider] = { requests: 0, tokens: 0, cost: 0 };
        }
        acc[provider].requests++;
        acc[provider].tokens += record.usage.totalTokens;
        acc[provider].cost += record.usage.costTotal;
        return acc;
      },
      {} as Record<string, { requests: number; tokens: number; cost: number }>
    );
  }, [usage]);

  return {
    byProvider,
    isLoading,
    error,
  };
}

interface UseUsageByModelReturn {
  byModel: Record<string, { requests: number; tokens: number; cost: number }>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook to get usage grouped by model.
 */
export function useUsageByModel(): UseUsageByModelReturn {
  const { usage, isLoading, error } = useUsage();

  const byModel = useMemo(() => {
    return usage.reduce(
      (acc, record) => {
        const model = record.selectedModel;
        if (!acc[model]) {
          acc[model] = { requests: 0, tokens: 0, cost: 0 };
        }
        acc[model].requests++;
        acc[model].tokens += record.usage.totalTokens;
        acc[model].cost += record.usage.costTotal;
        return acc;
      },
      {} as Record<string, { requests: number; tokens: number; cost: number }>
    );
  }, [usage]);

  return {
    byModel,
    isLoading,
    error,
  };
}
