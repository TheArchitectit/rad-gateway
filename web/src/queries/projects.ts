/**
 * RAD Gateway Admin UI - Projects Query Hooks
 * Data Fetching Developer - Phase 3 Implementation
 *
 * TanStack Query hooks for project/workspace management.
 * Features: pagination, filtering, optimistic updates, infinite scroll support.
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
import { projectsKeys } from './keys';
import type { Workspace } from '../types';

// ============================================================================
// Types
// ============================================================================

export interface ProjectListResponse {
  data: Workspace[];
  total: number;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export interface CreateProjectRequest {
  name: string;
  slug?: string;
  description?: string;
  settings?: Record<string, unknown>;
}

export interface UpdateProjectRequest {
  name?: string;
  description?: string;
  status?: string;
  settings?: Record<string, unknown>;
}

export interface ProjectFilters {
  status?: string;
  search?: string;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface BulkProjectRequest {
  ids: string[];
  action: 'activate' | 'deactivate' | 'delete' | 'archive';
}

export interface BulkProjectResponse {
  processed: number;
  action: string;
  success: boolean;
}

// ============================================================================
// API Functions
// ============================================================================

const fetchProjects = async (
  filters: ProjectFilters & { page?: number; pageSize?: number }
): Promise<ProjectListResponse> => {
  const params: Record<string, string | number> = {
    page: filters.page || 1,
    pageSize: filters.pageSize || 50,
  };

  if (filters.status) params['status'] = filters.status;
  if (filters.search) params['search'] = filters.search;
  if (filters.sortBy) params['sortBy'] = filters.sortBy;
  if (filters.sortOrder) params['sortOrder'] = filters.sortOrder;

  return apiClient.get<ProjectListResponse>('/v0/admin/projects', { params });
};

const fetchProject = async (id: string): Promise<Workspace> => {
  return apiClient.get<Workspace>(`/v0/admin/projects/${id}`);
};

const createProject = async (data: CreateProjectRequest): Promise<Workspace> => {
  return apiClient.post<Workspace>('/v0/admin/projects', data);
};

const updateProject = async (
  id: string,
  data: UpdateProjectRequest
): Promise<Workspace> => {
  return apiClient.put<Workspace>(`/v0/admin/projects/${id}`, data);
};

const patchProject = async (
  id: string,
  updates: Partial<UpdateProjectRequest>
): Promise<Workspace> => {
  return apiClient.patch<Workspace>(`/v0/admin/projects/${id}`, updates);
};

const deleteProject = async (id: string, force?: boolean): Promise<void> => {
  const params = force ? { force: 'true' } : undefined;
  return apiClient.delete<void>(`/v0/admin/projects/${id}`, { params });
};

const bulkOperation = async (
  data: BulkProjectRequest
): Promise<BulkProjectResponse> => {
  return apiClient.post<BulkProjectResponse>('/v0/admin/projects/bulk', data);
};

// ============================================================================
// Query Hooks
// ============================================================================

/**
 * Hook to fetch paginated projects with filtering.
 * Supports server-side pagination, sorting, and search.
 */
export function useProjects(
  filters: ProjectFilters & { page?: number; pageSize?: number } = {},
  options?: Omit<UseQueryOptions<ProjectListResponse, APIError>, 'queryKey' | 'queryFn'>
) {
  return useQuery<ProjectListResponse, APIError>({
    queryKey: projectsKeys.list(filters),
    queryFn: () => fetchProjects(filters),
    ...options,
  });
}

/**
 * Hook for infinite scrolling of projects.
 * Automatically handles pagination state.
 */
export function useProjectsInfinite(
  filters: Omit<ProjectFilters, 'page'> & { pageSize?: number } = {},
  options?: Parameters<typeof useInfiniteQuery<ProjectListResponse, APIError>>[0]
) {
  return useInfiniteQuery<ProjectListResponse, APIError>({
    queryKey: projectsKeys.lists(),
    queryFn: ({ pageParam = 1 }) =>
      fetchProjects({ ...filters, page: pageParam as number }),
    getNextPageParam: (lastPage) =>
      lastPage.hasMore ? lastPage.page + 1 : undefined,
    initialPageParam: 1,
    ...options,
  });
}

/**
 * Hook to fetch a single project by ID.
 * Automatically refetches when ID changes.
 */
export function useProject(
  id: string | undefined,
  options?: Omit<UseQueryOptions<Workspace, APIError>, 'queryKey' | 'queryFn' | 'enabled'>
) {
  return useQuery<Workspace, APIError>({
    queryKey: projectsKeys.detail(id || ''),
    queryFn: () => fetchProject(id!),
    enabled: !!id,
    ...options,
  });
}

// ============================================================================
// Mutation Hooks
// ============================================================================

/**
 * Hook to create a new project.
 * Includes optimistic updates and cache invalidation.
 */
export function useCreateProject(
  options?: UseMutationOptions<Workspace, APIError, CreateProjectRequest>
) {
  const queryClient = useQueryClient();

  return useMutation<Workspace, APIError, CreateProjectRequest>({
    mutationFn: createProject,
    onSuccess: (newProject) => {
      // Invalidate project lists to refetch
      queryClient.invalidateQueries({ queryKey: projectsKeys.lists() });

      // Immediately add to cache for instant UI feedback
      queryClient.setQueryData(
        projectsKeys.detail(newProject.id),
        newProject
      );
    },
    ...options,
  });
}

/**
 * Hook to update an existing project.
 * Includes optimistic update for immediate UI feedback.
 */
export function useUpdateProject(
  options?: UseMutationOptions<
    Workspace,
    APIError,
    { id: string; data: UpdateProjectRequest }
  >
) {
  const queryClient = useQueryClient();

  return useMutation<Workspace, APIError, { id: string; data: UpdateProjectRequest }>({
    mutationFn: ({ id, data }) => updateProject(id, data),
    onMutate: async ({ id, data }) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: projectsKeys.detail(id) });

      // Snapshot previous value
      const previousProject = queryClient.getQueryData<Workspace>(
        projectsKeys.detail(id)
      );

      // Optimistically update
      if (previousProject) {
        queryClient.setQueryData<Workspace>(projectsKeys.detail(id), {
          ...previousProject,
          ...data,
          updatedAt: new Date().toISOString(),
        } as Workspace);
      }

      return { previousProject };
    },
    onError: (_err, { id }, context: any) => {
      // Rollback on error
      if (context?.previousProject) {
        queryClient.setQueryData(projectsKeys.detail(id), context.previousProject);
      }
    },
    onSettled: (_data, _error, { id }) => {
      // Always refetch after error or success
      queryClient.invalidateQueries({ queryKey: projectsKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: projectsKeys.lists() });
    },
    ...options,
  });
}

/**
 * Hook to patch (partially update) a project.
 * Useful for quick toggles like status changes.
 */
export function usePatchProject(
  options?: UseMutationOptions<
    Workspace,
    APIError,
    { id: string; updates: Partial<UpdateProjectRequest> }
  >
) {
  const queryClient = useQueryClient();

  return useMutation<Workspace, APIError, { id: string; updates: Partial<UpdateProjectRequest> }>({
    mutationFn: ({ id, updates }) => patchProject(id, updates),
    onMutate: async ({ id, updates }) => {
      await queryClient.cancelQueries({ queryKey: projectsKeys.detail(id) });

      const previousProject = queryClient.getQueryData<Workspace>(
        projectsKeys.detail(id)
      );

      if (previousProject) {
        queryClient.setQueryData<Workspace>(projectsKeys.detail(id), {
          ...previousProject,
          ...updates,
          updatedAt: new Date().toISOString(),
        } as Workspace);
      }

      return { previousProject };
    },
    onError: (_err, { id }, context: any) => {
      if (context?.previousProject) {
        queryClient.setQueryData(projectsKeys.detail(id), context.previousProject);
      }
    },
    onSettled: (_data, _error, { id }) => {
      queryClient.invalidateQueries({ queryKey: projectsKeys.detail(id) });
      queryClient.invalidateQueries({ queryKey: projectsKeys.lists() });
    },
    ...options,
  });
}

/**
 * Hook to delete a project.
 * Supports soft delete (default) or force delete.
 */
export function useDeleteProject(
  options?: UseMutationOptions<void, APIError, { id: string; force?: boolean }>
) {
  const queryClient = useQueryClient();

  return useMutation<void, APIError, { id: string; force?: boolean }>({
    mutationFn: ({ id, force }) => deleteProject(id, force),
    onSuccess: (_, { id }) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: projectsKeys.detail(id) });
      // Invalidate lists
      queryClient.invalidateQueries({ queryKey: projectsKeys.lists() });
    },
    ...options,
  });
}

/**
 * Hook for bulk operations on projects.
 * Efficiently updates multiple projects at once.
 */
export function useBulkProjectOperation(
  options?: UseMutationOptions<BulkProjectResponse, APIError, BulkProjectRequest>
) {
  const queryClient = useQueryClient();

  return useMutation<BulkProjectResponse, APIError, BulkProjectRequest>({
    mutationFn: bulkOperation,
    onSuccess: () => {
      // Invalidate all project queries
      queryClient.invalidateQueries({ queryKey: projectsKeys.all });
    },
    ...options,
  });
}

// ============================================================================
// Utility Hooks
// ============================================================================

/**
 * Hook to toggle a project's active status.
 * Convenience wrapper around usePatchProject.
 */
export function useToggleProjectStatus() {
  const patchMutation = usePatchProject();

  return {
    ...patchMutation,
    mutate: (id: string, currentStatus: string) => {
      const newStatus = currentStatus === 'active' ? 'inactive' : 'active';
      return patchMutation.mutate({ id, updates: { status: newStatus } });
    },
    mutateAsync: async (id: string, currentStatus: string) => {
      const newStatus = currentStatus === 'active' ? 'inactive' : 'active';
      return patchMutation.mutateAsync({ id, updates: { status: newStatus } });
    },
  };
}

/**
 * Hook to prefetch a project for instant navigation.
 * Use this when hovering over project links.
 */
export function usePrefetchProject() {
  const queryClient = useQueryClient();

  return (id: string) => {
    queryClient.prefetchQuery({
      queryKey: projectsKeys.detail(id),
      queryFn: () => fetchProject(id),
      staleTime: 60 * 1000, // 1 minute
    });
  };
}
