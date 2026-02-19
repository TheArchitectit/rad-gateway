/**
 * RAD Gateway Admin UI - Usage Query Hooks
 * Data Fetching Developer - Phase 3 Implementation
 *
 * TanStack Query hooks for usage analytics and reporting.
 * Features: aggregation, trends, exports, real-time updates support.
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  useInfiniteQuery,
  UseQueryOptions,
  UseMutationOptions,
} from '@tanstack/react-query';
import { apiClient, APIError } from '../api/client';
import { usageKeys } from './keys';
import type { UsageFilters } from '../types';

// ============================================================================
// Types
// ============================================================================

export interface UsageRecordResponse {
  id: string;
  workspaceId: string;
  requestId: string;
  traceId: string;
  apiKeyId?: string;
  controlRoomId?: string;
  incomingApi: string;
  incomingModel: string;
  selectedModel?: string;
  providerId?: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  costUsd?: number;
  durationMs: number;
  responseStatus: string;
  errorCode?: string;
  errorMessage?: string;
  attempts: number;
  startedAt: string;
  completedAt?: string;
}

export interface UsageAggregation {
  dimension: string;
  value: string;
  metrics: Record<string, number>;
}

export interface UsageSummary {
  totalRequests: number;
  totalTokens: number;
  totalPromptTokens: number;
  totalOutputTokens: number;
  totalCostUsd: number;
  avgDurationMs: number;
  successCount: number;
  errorCount: number;
  errorRate: number;
}

export interface UsageListResponse {
  data: UsageRecordResponse[];
  aggregations?: UsageAggregation[];
  summary: UsageSummary;
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export interface UsageTrendPoint {
  timestamp: string;
  requestCount: number;
  tokenCount: number;
  costUsd: number;
  avgLatencyMs: number;
  errorCount: number;
}

export interface UsageTrendResponse {
  timeRange: {
    start: string;
    end: string;
  };
  interval: 'minute' | 'hour' | 'day';
  points: UsageTrendPoint[];
}

export interface UsageQueryRequest {
  startTime?: string;
  endTime?: string;
  workspaceId?: string;
  apiKeyId?: string;
  providerId?: string;
  model?: string;
  incomingApi?: string;
  status?: string;
  groupBy?: string[];
  aggregations?: string[];
}

export interface UsageExportRequest {
  startTime: string;
  endTime: string;
  format: 'json' | 'csv';
  workspaceId?: string;
  includeCost: boolean;
}

export interface UsageExportResponse {
  exportId: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  downloadUrl?: string;
  expiresAt?: string;
  recordCount: number;
}

export type UsageTimeRange = '1h' | '24h' | '7d' | '30d' | 'custom';

// ============================================================================
// API Functions
// ============================================================================

const fetchUsage = async (
  filters: UsageFilters & { page?: number; pageSize?: number }
): Promise<UsageListResponse> => {
  const params: Record<string, string | number> = {
    page: filters.page || 1,
    pageSize: filters.pageSize || 50,
  };

  if (filters.startTime) params['startTime'] = filters.startTime;
  if (filters.endTime) params['endTime'] = filters.endTime;
  if (filters.apiKeyName) params['apiKeyId'] = filters.apiKeyName;
  if (filters.provider) params['providerId'] = filters.provider;
  if (filters.status) params['status'] = filters.status;

  return apiClient.get<UsageListResponse>('/v0/admin/usage', { params });
};

const fetchUsageRecords = async (
  filters: { workspaceId?: string; apiKeyId?: string; startTime?: string; endTime?: string; page?: number; pageSize?: number }
): Promise<UsageListResponse> => {
  const params: Record<string, string | number> = {
    page: filters.page || 1,
    pageSize: filters.pageSize || 100,
  };

  if (filters.workspaceId) params['workspaceId'] = filters.workspaceId;
  if (filters.apiKeyId) params['apiKeyId'] = filters.apiKeyId;
  if (filters.startTime) params['startTime'] = filters.startTime;
  if (filters.endTime) params['endTime'] = filters.endTime;

  return apiClient.get<UsageListResponse>('/v0/admin/usage/records', { params });
};

const queryUsageAdvanced = async (
  request: UsageQueryRequest
): Promise<UsageListResponse> => {
  return apiClient.post<UsageListResponse>('/v0/admin/usage', request);
};

const fetchUsageTrends = async (params: {
  startTime?: string;
  endTime?: string;
  interval?: 'minute' | 'hour' | 'day';
}): Promise<UsageTrendResponse> => {
  const queryParams: Record<string, string> = {};
  if (params.startTime) queryParams['startTime'] = params.startTime;
  if (params.endTime) queryParams['endTime'] = params.endTime;
  if (params.interval) queryParams['interval'] = params.interval;

  return apiClient.get<UsageTrendResponse>('/v0/admin/usage/trends', {
    params: queryParams,
  });
};

const fetchUsageSummary = async (filters?: {
  workspaceId?: string;
  startTime?: string;
  endTime?: string;
}): Promise<UsageSummary> => {
  const params: Record<string, string> = {};
  if (filters?.workspaceId) params['workspaceId'] = filters.workspaceId;
  if (filters?.startTime) params['startTime'] = filters.startTime;
  if (filters?.endTime) params['endTime'] = filters.endTime;

  return apiClient.get<UsageSummary>('/v0/admin/usage/summary', { params });
};

const createExport = async (
  request: UsageExportRequest
): Promise<UsageExportResponse> => {
  return apiClient.post<UsageExportResponse>('/v0/admin/usage/export', request);
};

const fetchExportStatus = async (exportId: string): Promise<UsageExportResponse> => {
  return apiClient.get<UsageExportResponse>(`/v0/admin/usage/export/${exportId}`);
};

// ============================================================================
// Query Hooks
// ============================================================================

/**
 * Hook to fetch usage data with filtering and pagination.
 * Includes summary and optional aggregations.
 */
export function useUsage(
  filters: UsageFilters & { page?: number; pageSize?: number } = {},
  options?: Omit<UseQueryOptions<UsageListResponse, APIError>, 'queryKey' | 'queryFn'>
) {
  return useQuery<UsageListResponse, APIError>({
    queryKey: usageKeys.list(filters),
    queryFn: () => fetchUsage(filters),
    ...options,
  });
}

/**
 * Hook for infinite scrolling of usage records.
 * Efficient for large log datasets.
 */
export function useUsageInfinite(
  filters: Omit<UsageFilters, 'page'> & { pageSize?: number } = {},
  options?: Parameters<typeof useInfiniteQuery<UsageListResponse, APIError>>[0]
) {
  return useInfiniteQuery<UsageListResponse, APIError>({
    queryKey: usageKeys.lists(),
    queryFn: ({ pageParam = 1 }) =>
      fetchUsage({ ...filters, page: pageParam as number }),
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.page + 1 : undefined,
    initialPageParam: 1,
    ...options,
  });
}

/**
 * Hook for advanced usage queries with aggregations.
 * Use this for grouped/aggregated analytics.
 */
export function useUsageAdvanced(
  request: UsageQueryRequest,
  options?: Omit<UseQueryOptions<UsageListResponse, APIError>, 'queryKey' | 'queryFn'>
) {
  return useQuery<UsageListResponse, APIError>({
    queryKey: usageKeys.aggregation(request.groupBy || [], {
      ...(request.startTime && { startTime: request.startTime }),
      ...(request.endTime && { endTime: request.endTime }),
      ...(request.providerId && { provider: request.providerId }),
      ...(request.status && { status: request.status as 'success' | 'error' | 'timeout' }),
    }),
    queryFn: () => queryUsageAdvanced(request),
    enabled: !!request.groupBy?.length,
    ...options,
  });
}

/**
 * Hook to fetch individual usage records.
 * Optimized for detailed log viewing.
 */
export function useUsageRecords(
  filters: {
    workspaceId?: string;
    apiKeyId?: string;
    startTime?: string;
    endTime?: string;
  },
  options?: Omit<UseQueryOptions<UsageListResponse, APIError>, 'queryKey' | 'queryFn'>
) {
  return useQuery<UsageListResponse, APIError>({
    queryKey: usageKeys.record(filters),
    queryFn: () => fetchUsageRecords(filters),
    ...options,
  });
}

/**
 * Hook to fetch usage trends over time.
 * Perfect for charting usage patterns.
 */
export function useUsageTrends(
  params: {
    startTime?: string;
    endTime?: string;
    interval?: 'minute' | 'hour' | 'day';
  },
  options?: Omit<UseQueryOptions<UsageTrendResponse, APIError>, 'queryKey' | 'queryFn'>
) {
  return useQuery<UsageTrendResponse, APIError>({
    queryKey: usageKeys.trend(params),
    queryFn: () => fetchUsageTrends(params),
    ...options,
  });
}

/**
 * Hook to fetch usage summary statistics.
 * Lightweight endpoint for dashboard metrics.
 */
export function useUsageSummary(
  filters?: { workspaceId?: string; startTime?: string; endTime?: string },
  options?: Omit<UseQueryOptions<UsageSummary, APIError>, 'queryKey' | 'queryFn'>
) {
  return useQuery<UsageSummary, APIError>({
    queryKey: usageKeys.summaryDetail(filters || {}),
    queryFn: () => fetchUsageSummary(filters),
    ...options,
  });
}

/**
 * Hook to poll export status.
 * Automatically polls while export is pending/processing.
 */
export function useExportStatus(
  exportId: string | undefined,
  options?: Omit<UseQueryOptions<UsageExportResponse, APIError>, 'queryKey' | 'queryFn' | 'enabled'>
) {
  return useQuery<UsageExportResponse, APIError>({
    queryKey: usageKeys.export(exportId || ''),
    queryFn: () => fetchExportStatus(exportId!),
    enabled: !!exportId,
    refetchInterval: (data) => {
      const response = (data as unknown) as UsageExportResponse | undefined;
      return response?.status === 'pending' || response?.status === 'processing' ? 2000 : false;
    },
    ...options,
  });
}

// ============================================================================
// Mutation Hooks
// ============================================================================

/**
 * Hook to create a usage data export.
 * Poll for completion using useExportStatus.
 */
export function useCreateExport(
  options?: UseMutationOptions<UsageExportResponse, APIError, UsageExportRequest>
) {
  const queryClient = useQueryClient();

  return useMutation<UsageExportResponse, APIError, UsageExportRequest>({
    mutationFn: createExport,
    onSuccess: (data) => {
      queryClient.setQueryData(usageKeys.export(data.exportId), data);
    },
    ...options,
  });
}

// ============================================================================
// Utility Hooks
// ============================================================================

/**
 * Hook to get computed usage metrics from cached data.
 */
export function useUsageMetrics(
  filters?: UsageFilters
): {
  metrics: UsageSummary | null;
  isLoading: boolean;
  error: APIError | null;
} {
  const { data, isLoading, error } = useUsage(filters);

  return {
    metrics: data?.summary || null,
    isLoading,
    error: error || null,
  };
}

/**
 * Hook to get usage grouped by dimension.
 */
export function useUsageByDimension(
  dimension: 'workspaceId' | 'providerId' | 'model' | 'api' | 'status',
  filters?: UsageFilters
) {
  const { data, isLoading, error } = useUsageAdvanced({
    ...filters,
    groupBy: [dimension],
    aggregations: ['requestCount', 'totalTokens', 'costUsd'],
  });

  const grouped = data?.aggregations?.reduce(
    (acc, agg) => {
      if (agg.dimension === dimension) {
        acc[agg.value] = agg.metrics;
      }
      return acc;
    },
    {} as Record<string, Record<string, number>>
  );

  return {
    data: grouped || {},
    isLoading,
    error,
  };
}

/**
 * Hook to get usage trends with automatic time range calculation.
 */
export function useUsageTrendsPreset(
  preset: UsageTimeRange,
  options?: Omit<UseQueryOptions<UsageTrendResponse, APIError>, 'queryKey' | 'queryFn'>
) {
  const now = new Date();
  let startTime: string;
  let interval: 'minute' | 'hour' | 'day';

  switch (preset) {
    case '1h':
      startTime = new Date(now.getTime() - 60 * 60 * 1000).toISOString();
      interval = 'minute';
      break;
    case '24h':
      startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
      interval = 'hour';
      break;
    case '7d':
      startTime = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000).toISOString();
      interval = 'day';
      break;
    case '30d':
      startTime = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000).toISOString();
      interval = 'day';
      break;
    default:
      startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
      interval = 'hour';
  }

  return useUsageTrends(
    {
      startTime,
      endTime: now.toISOString(),
      interval,
    },
    options
  );
}

/**
 * Hook to prefetch usage data for instant navigation.
 */
export function usePrefetchUsage() {
  const queryClient = useQueryClient();

  return (filters: UsageFilters) => {
    queryClient.prefetchQuery({
      queryKey: usageKeys.list(filters),
      queryFn: () => fetchUsage(filters),
      staleTime: 30 * 1000,
    });
  };
}

/**
 * Hook to invalidate all usage queries.
 */
export function useInvalidateUsage() {
  const queryClient = useQueryClient();

  return () => {
    queryClient.invalidateQueries({ queryKey: usageKeys.all });
  };
}
