/**
 * RAD Gateway Admin UI - Providers Query Hooks
 * Data Fetching Developer - Phase 3 Implementation
 *
 * TanStack Query hooks for AI provider management.
 * Features: health checks, status monitoring, optimistic updates.
 */

import {
  useQuery,
  useMutation,
  useQueryClient,
  UseQueryOptions,
  UseMutationOptions,
} from '@tanstack/react-query';
import { apiClient, APIError } from '../api/client';
import { providersKeys } from './keys';
import type { Provider, ProviderHealth } from '../types';

// ============================================================================
// Types
// ============================================================================

export interface ProviderListResponse {
  providers: Provider[];
}

export interface ProviderStatusUpdate {
  status: 'healthy' | 'degraded' | 'unhealthy' | 'disabled';
  circuitBreaker?: 'closed' | 'open' | 'half-open';
}

// ============================================================================
// API Functions
// ============================================================================

const fetchProviders = async (): Promise<ProviderListResponse> => {
  return apiClient.get<ProviderListResponse>('/v0/admin/providers');
};

const fetchProvider = async (name: string): Promise<Provider> => {
  // Individual provider endpoint - uses the list and filters
  const response = await fetchProviders();
  const provider = response.providers.find((p) => p.name === name);
  if (!provider) {
    throw new APIError('Provider not found', 'not_found', 404);
  }
  return provider;
};

const checkProviderHealth = async (name: string): Promise<ProviderHealth> => {
  return apiClient.post<ProviderHealth>(`/v0/admin/providers/${name}/health`);
};

// ============================================================================
// Query Hooks
// ============================================================================

/**
 * Hook to fetch all providers.
 * Automatically refreshes to keep status current.
 */
export function useProviders(
  options?: Omit<UseQueryOptions<ProviderListResponse, APIError>, 'queryKey' | 'queryFn'>
) {
  return useQuery<ProviderListResponse, APIError>({
    queryKey: providersKeys.list(),
    queryFn: fetchProviders,
    // Refresh frequently for status updates
    refetchInterval: 30 * 1000, // 30 seconds
    staleTime: 15 * 1000,
    ...options,
  });
}

/**
 * Hook to fetch a single provider by name.
 */
export function useProvider(
  name: string | undefined,
  options?: Omit<UseQueryOptions<Provider, APIError>, 'queryKey' | 'queryFn' | 'enabled'>
) {
  return useQuery<Provider, APIError>({
    queryKey: providersKeys.detail(name || ''),
    queryFn: () => fetchProvider(name!),
    enabled: !!name,
    ...options,
  });
}

/**
 * Hook to check provider health.
 * Returns detailed health information.
 */
export function useProviderHealth(
  name: string | undefined,
  options?: Omit<UseQueryOptions<ProviderHealth, APIError>, 'queryKey' | 'queryFn' | 'enabled'>
) {
  return useQuery<ProviderHealth, APIError>({
    queryKey: providersKeys.health(name || ''),
    queryFn: () => checkProviderHealth(name!),
    enabled: !!name,
    // Health checks are fresh for a short time
    staleTime: 10 * 1000,
    ...options,
  });
}

// ============================================================================
// Mutation Hooks
// ============================================================================

/**
 * Hook to trigger a health check for a provider.
 * Updates the cached health data immediately.
 */
export function useCheckProviderHealth(
  options?: UseMutationOptions<ProviderHealth, APIError, string>
) {
  const queryClient = useQueryClient();

  return useMutation<ProviderHealth, APIError, string>({
    mutationFn: checkProviderHealth,
    onSuccess: (data, name) => {
      // Update the health cache
      queryClient.setQueryData(providersKeys.health(name), data);

      // Invalidate the provider list to reflect new status
      queryClient.invalidateQueries({ queryKey: providersKeys.lists() });
    },
    ...options,
  });
}

// ============================================================================
// Utility Hooks
// ============================================================================

/**
 * Hook to get providers grouped by status.
 * Useful for dashboard views.
 */
export function useProvidersByStatus() {
  const { data, ...rest } = useProviders();

  const grouped = data?.providers.reduce(
    (acc, provider) => {
      const status = provider.status;
      if (!acc[status]) {
        acc[status] = [];
      }
      acc[status].push(provider);
      return acc;
    },
    {} as Record<string, Provider[]>
  );

  return {
    ...rest,
    data: data?.providers || [],
    byStatus: grouped || {},
    healthy: grouped?.healthy || [],
    degraded: grouped?.degraded || [],
    unhealthy: grouped?.unhealthy || [],
    disabled: grouped?.disabled || [],
  };
}

/**
 * Hook to get provider statistics.
 */
export function useProviderStats() {
  const { data, isLoading, error } = useProviders();

  const stats = data?.providers.reduce(
    (acc, provider) => {
      acc.total++;
      acc[provider.status]++;
      acc.totalRequests += provider.requestCount24h;
      acc.totalErrors += Math.floor(
        provider.requestCount24h * provider.errorRate24h
      );
      return acc;
    },
    {
      total: 0,
      healthy: 0,
      degraded: 0,
      unhealthy: 0,
      disabled: 0,
      totalRequests: 0,
      totalErrors: 0,
    }
  );

  return {
    stats: stats || {
      total: 0,
      healthy: 0,
      degraded: 0,
      unhealthy: 0,
      disabled: 0,
      totalRequests: 0,
      totalErrors: 0,
    },
    isLoading,
    error,
  };
}

/**
 * Hook to prefetch provider data for instant navigation.
 */
export function usePrefetchProvider() {
  const queryClient = useQueryClient();

  return (name: string) => {
    queryClient.prefetchQuery({
      queryKey: providersKeys.detail(name),
      queryFn: () => fetchProvider(name),
      staleTime: 30 * 1000,
    });
  };
}

/**
 * Hook to refresh all provider data.
 * Useful for manual refresh buttons.
 */
export function useRefreshProviders() {
  const queryClient = useQueryClient();

  return () => {
    queryClient.invalidateQueries({ queryKey: providersKeys.all });
  };
}
