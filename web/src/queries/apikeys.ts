/**
 * RAD Gateway Admin UI - API Keys Query Hooks
 * Data Fetching Developer - Phase 3 Implementation
 *
 * TanStack Query hooks for API key management.
 * Features: pagination, filtering, key rotation, optimistic updates.
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
import { apiKeysKeys } from './keys';

// ============================================================================
// Types
// ============================================================================

export interface APIKeyListResponse {
  data: APIKeyResponse[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export interface APIKeyResponse {
  id: string;
  workspaceId: string;
  name: string;
  keyPreview: string;
  status: 'active' | 'revoked' | 'expired';
  createdBy?: string;
  expiresAt?: string;
  lastUsedAt?: string;
  revokedAt?: string;
  rateLimit?: number;
  allowedModels?: string[];
  allowedAPIs?: string[];
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface APIKeyWithSecret extends APIKeyResponse {
  keySecret: string;
}

export interface CreateAPIKeyRequest {
  name: string;
  workspaceId: string;
  expiresAt?: string | undefined;
  rateLimit?: number | undefined;
  allowedModels?: string[] | undefined;
  allowedAPIs?: string[] | undefined;
  metadata?: Record<string, unknown> | undefined;
}

export interface UpdateAPIKeyRequest {
  name?: string;
  status?: string;
  expiresAt?: string;
  rateLimit?: number;
  allowedModels?: string[];
  allowedAPIs?: string[];
  metadata?: Record<string, unknown>;
}

export interface RotateAPIKeyRequest {
  expiresAt?: string;
}

export interface APIKeyFilters {
  status?: string;
  workspaceId?: string;
  search?: string;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface BulkAPIKeyRequest {
  ids: string[];
  action: 'activate' | 'revoke' | 'delete';
}

export interface BulkAPIKeyResponse {
  processed: number;
  action: string;
  success: boolean;
}

export interface APIKeyStats {
  totalCount: number;
  activeCount: number;
  revokedCount: number;
  expiredCount: number;
  totalUsage: number;
}

// ============================================================================
// API Functions
// ============================================================================

const fetchAPIKeys = async (
  filters: APIKeyFilters & { page?: number; pageSize?: number }
): Promise<APIKeyListResponse> => {
  const params: Record<string, string | number> = {
    page: filters.page || 1,
    pageSize: filters.pageSize || 50,
  };

  if (filters.status) params['status'] = filters.status;
  if (filters.workspaceId) params['workspaceId'] = filters.workspaceId;
  if (filters.search) params['search'] = filters.search;
  if (filters.sortBy) params['sortBy'] = filters.sortBy;
  if (filters.sortOrder) params['sortOrder'] = filters.sortOrder;

  return apiClient.get<APIKeyListResponse>('/v0/admin/apikeys', { params });
};

const fetchAPIKey = async (id: string): Promise<APIKeyResponse> => {
  return apiClient.get<APIKeyResponse>(`/v0/admin/apikeys/${id}`);
};

const createAPIKey = async (
  data: CreateAPIKeyRequest
): Promise<APIKeyWithSecret> => {
  return apiClient.post<APIKeyWithSecret>('/v0/admin/apikeys', data);
};

const updateAPIKey = async (
  id: string,
  data: UpdateAPIKeyRequest
): Promise<APIKeyResponse> => {
  return apiClient.put<APIKeyResponse>(`/v0/admin/apikeys/${id}`, data);
};

const patchAPIKey = async (
  id: string,
  updates: Partial<UpdateAPIKeyRequest>
): Promise<APIKeyResponse> => {
  return apiClient.patch<APIKeyResponse>(`/v0/admin/apikeys/${id}`, updates);
};

const revokeAPIKey = async (id: string): Promise<APIKeyResponse> => {
  return apiClient.post<APIKeyResponse>(`/v0/admin/apikeys/${id}/revoke`);
};

const rotateAPIKey = async (
  id: string,
  data?: RotateAPIKeyRequest
): Promise<APIKeyWithSecret> => {
  return apiClient.post<APIKeyWithSecret>(`/v0/admin/apikeys/${id}/rotate`, data);
};

const deleteAPIKey = async (id: string): Promise<void> => {
  return apiClient.delete<void>(`/v0/admin/apikeys/${id}`);
};

const bulkOperation = async (
  data: BulkAPIKeyRequest
): Promise<BulkAPIKeyResponse> => {
  return apiClient.post<BulkAPIKeyResponse>('/v0/admin/apikeys/bulk', data);
};

// ============================================================================
// Query Hooks
// ============================================================================

/**
 * Hook to fetch paginated API keys with filtering.
 */
export function useAPIKeys(
  filters: APIKeyFilters & { page?: number; pageSize?: number } = {},
  options?: Omit<UseQueryOptions<APIKeyListResponse, APIError>, 'queryKey' | 'queryFn'>
) {
  return useQuery<APIKeyListResponse, APIError>({
    queryKey: apiKeysKeys.list(filters),
    queryFn: () => fetchAPIKeys(filters),
    ...options,
  });
}

/**
 * Hook for infinite scrolling of API keys.
 */
export function useAPIKeysInfinite(
  filters: Omit<APIKeyFilters, 'page'> & { pageSize?: number } = {},
  options?: Parameters<typeof useInfiniteQuery<APIKeyListResponse, APIError>>[0]
) {
  return useInfiniteQuery<APIKeyListResponse, APIError>({
    queryKey: apiKeysKeys.lists(),
    queryFn: ({ pageParam = 1 }) =>
      fetchAPIKeys({ ...filters, page: pageParam as number }),
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.page + 1 : undefined,
    initialPageParam: 1,
    ...options,
  });
}

/**
 * Hook to fetch a single API key by ID.
 */
export function useAPIKey(
  id: string | undefined,
  options?: Omit<UseQueryOptions<APIKeyResponse, APIError>, 'queryKey' | 'queryFn' | 'enabled'>
) {
  return useQuery<APIKeyResponse, APIError>({
    queryKey: apiKeysKeys.detail(id || ''),
    queryFn: () => fetchAPIKey(id!),
    enabled: !!id,
    ...options,
  });
}

// ============================================================================
// Mutation Hooks
// ============================================================================

/**
 * Hook to create a new API key.
 * Important: The secret key is only returned once - capture it immediately!
 */
export function useCreateAPIKey(
  options?: UseMutationOptions<APIKeyWithSecret, APIError, CreateAPIKeyRequest>
) {
  const queryClient = useQueryClient();

  return useMutation<APIKeyWithSecret, APIError, CreateAPIKeyRequest>({
    mutationFn: createAPIKey,
    onSuccess: (newKey) => {
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.lists() });

      // Cache the new key (without secret for security)
      const { keySecret: _, ...keyWithoutSecret } = newKey;
      queryClient.setQueryData(
        apiKeysKeys.detail(newKey.id),
        keyWithoutSecret
      );
    },
    ...options,
  });
}

/**
 * Hook to update an existing API key.
 * Includes optimistic update.
 */
export function useUpdateAPIKey(
  options?: UseMutationOptions<
    APIKeyResponse,
    APIError,
    { id: string; data: UpdateAPIKeyRequest }
  >
) {
  const queryClient = useQueryClient();

  return useMutation<APIKeyResponse, APIError, { id: string; data: UpdateAPIKeyRequest }>({
    // @ts-ignore - TanStack Query type inference issue
    mutationFn: ({ id, data }) => updateAPIKey(id, data),
    // @ts-ignore - Context type inference
    onMutate: async ({ id, data }): Promise<{ previousKey: APIKeyResponse | undefined }> => {
      await queryClient.cancelQueries({ queryKey: apiKeysKeys.detail(id) });

      const previousKey = queryClient.getQueryData<APIKeyResponse>(
        apiKeysKeys.detail(id)
      );

      if (previousKey) {
        queryClient.setQueryData<APIKeyResponse>(apiKeysKeys.detail(id), {
          ...previousKey,
          ...data,
          updatedAt: new Date().toISOString(),
        } as APIKeyResponse);
      }

      return { previousKey } as { previousKey: APIKeyResponse | undefined };
    },
    // @ts-ignore
    onError: (_err, { id }, context: any) => {
      if (context?.previousKey) {
        queryClient.setQueryData(apiKeysKeys.detail(id), context.previousKey);
      }
    },
    onSettled: (_data, _error, { id }) => {
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.lists() });
    },
    ...options,
  });
}

/**
 * Hook to patch (partially update) an API key.
 */
export function usePatchAPIKey(
  options?: UseMutationOptions<
    APIKeyResponse,
    APIError,
    { id: string; updates: Partial<UpdateAPIKeyRequest> }
  >
) {
  const queryClient = useQueryClient();

  return useMutation<APIKeyResponse, APIError, { id: string; updates: Partial<UpdateAPIKeyRequest> }>({
    mutationFn: ({ id, updates }) => patchAPIKey(id, updates),
    onMutate: async ({ id, updates }) => {
      await queryClient.cancelQueries({ queryKey: apiKeysKeys.detail(id) });

      const previousKey = queryClient.getQueryData<APIKeyResponse>(
        apiKeysKeys.detail(id)
      );

      if (previousKey) {
        queryClient.setQueryData<APIKeyResponse>(apiKeysKeys.detail(id), {
          ...previousKey,
          ...updates,
          updatedAt: new Date().toISOString(),
        } as APIKeyResponse);
      }

      return { previousKey } as { previousKey: APIKeyResponse | undefined };
    },
    // @ts-ignore
    onError: (_err, { id }, context: any) => {
      if (context?.previousKey) {
        queryClient.setQueryData(apiKeysKeys.detail(id), context.previousKey);
      }
    },
    onSettled: (_data, _error, { id }) => {
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.lists() });
    },
    ...options,
  });
}

/**
 * Hook to revoke an API key.
 * Revoked keys cannot be used but remain in the system for audit purposes.
 */
export function useRevokeAPIKey(
  options?: UseMutationOptions<APIKeyResponse, APIError, string>
) {
  const queryClient = useQueryClient();

  return useMutation<APIKeyResponse, APIError, string>({
    mutationFn: revokeAPIKey,
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: apiKeysKeys.detail(id) });

      const previousKey = queryClient.getQueryData<APIKeyResponse>(
        apiKeysKeys.detail(id)
      );

      if (previousKey) {
        queryClient.setQueryData<APIKeyResponse>(apiKeysKeys.detail(id), {
          ...previousKey,
          status: 'revoked',
          revokedAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        });
      }

      return { previousKey } as { previousKey: APIKeyResponse | undefined };
    },
    onError: (_err, id, context: any) => {
      if (context?.previousKey) {
        queryClient.setQueryData(apiKeysKeys.detail(id), context.previousKey);
      }
    },
    onSettled: (_data, _error, id) => {
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.lists() });
    },
    ...options,
  });
}

/**
 * Hook to rotate an API key.
 * Creates a new key, invalidates the old one, returns new secret.
 * Important: The new secret is only returned once!
 */
export function useRotateAPIKey(
  options?: UseMutationOptions<
    APIKeyWithSecret,
    APIError,
    { id: string; data?: RotateAPIKeyRequest }
  >
) {
  const queryClient = useQueryClient();

  return useMutation<APIKeyWithSecret, APIError, { id: string; data?: RotateAPIKeyRequest }>({
    mutationFn: ({ id, data }) => rotateAPIKey(id, data),
    onSuccess: (newKey, { id }) => {
      // Update the cache with the rotated key
      const { keySecret: _, ...keyWithoutSecret } = newKey;
      queryClient.setQueryData(apiKeysKeys.detail(id), keyWithoutSecret);

      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.lists() });
    },
    ...options,
  });
}

/**
 * Hook to delete an API key.
 * Permanent deletion - use revoke for soft-disable.
 */
export function useDeleteAPIKey(
  options?: UseMutationOptions<void, APIError, string>
) {
  const queryClient = useQueryClient();

  return useMutation<void, APIError, string>({
    mutationFn: deleteAPIKey,
    onSuccess: (_, id) => {
      queryClient.removeQueries({ queryKey: apiKeysKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.lists() });
    },
    ...options,
  });
}

/**
 * Hook for bulk operations on API keys.
 */
export function useBulkAPIKeyOperation(
  options?: UseMutationOptions<BulkAPIKeyResponse, APIError, BulkAPIKeyRequest>
) {
  const queryClient = useQueryClient();

  return useMutation<BulkAPIKeyResponse, APIError, BulkAPIKeyRequest>({
    mutationFn: bulkOperation,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: apiKeysKeys.all });
    },
    ...options,
  });
}

// ============================================================================
// Utility Hooks
// ============================================================================

/**
 * Hook to get API key statistics.
 * Computed from the cached list data.
 */
export function useAPIKeyStats(workspaceId?: string): APIKeyStats {
  const queryClient = useQueryClient();

  // Try to get stats from cached list data
  const cachedData = queryClient.getQueriesData<APIKeyListResponse>({
    queryKey: apiKeysKeys.lists(),
  });

  const allKeys = cachedData
    .flatMap(([, data]) => data?.data || [])
    .filter((key) => !workspaceId || key.workspaceId === workspaceId);

  return allKeys.reduce(
    (acc, key) => {
      acc.totalCount++;
      if (key.status === 'active') acc.activeCount++;
      if (key.status === 'revoked') acc.revokedCount++;
      if (key.status === 'expired') acc.expiredCount++;
      return acc;
    },
    {
      totalCount: 0,
      activeCount: 0,
      revokedCount: 0,
      expiredCount: 0,
      totalUsage: 0,
    }
  );
}

/**
 * Hook to prefetch an API key for instant navigation.
 */
export function usePrefetchAPIKey() {
  const queryClient = useQueryClient();

  return (id: string) => {
    queryClient.prefetchQuery({
      queryKey: apiKeysKeys.detail(id),
      queryFn: () => fetchAPIKey(id),
      staleTime: 60 * 1000,
    });
  };
}

/**
 * Hook to check if an API key name is available.
 * Useful for real-time validation during creation.
 */
export function useCheckAPIKeyName(workspaceId: string, name: string) {
  const { data: keys } = useAPIKeys({ workspaceId });

  return {
    isAvailable: !keys?.data.some(
      (key) => key.name.toLowerCase() === name.toLowerCase()
    ),
    isChecking: !keys,
  };
}
