/**
 * RAD Gateway Admin UI - Query Hooks Index
 * Data Fetching Developer - Phase 3 Implementation
 *
 * Centralized exports for all TanStack Query hooks.
 */

// ============================================================================
// Query Provider
// ============================================================================

export { QueryProvider, QueryClient } from './QueryProvider';

// ============================================================================
// Query Keys
// ============================================================================

export {
  projectsKeys,
  apiKeysKeys,
  usageKeys,
  providersKeys,
  modelRoutesKeys,
  adminKeys,
  controlRoomsKeys,
} from './keys';

// ============================================================================
// Projects / Workspaces
// ============================================================================

export {
  // Queries
  useProjects,
  useProjectsInfinite,
  useProject,

  // Mutations
  useCreateProject,
  useUpdateProject,
  usePatchProject,
  useDeleteProject,
  useBulkProjectOperation,

  // Utilities
  useToggleProjectStatus,
  usePrefetchProject,

  // Types
  type ProjectListResponse,
  type CreateProjectRequest,
  type UpdateProjectRequest,
  type ProjectFilters,
  type BulkProjectRequest,
  type BulkProjectResponse,
} from './projects';

// ============================================================================
// API Keys
// ============================================================================

export {
  // Queries
  useAPIKeys,
  useAPIKeysInfinite,
  useAPIKey,

  // Mutations
  useCreateAPIKey,
  useUpdateAPIKey,
  usePatchAPIKey,
  useRevokeAPIKey,
  useRotateAPIKey,
  useDeleteAPIKey,
  useBulkAPIKeyOperation,

  // Utilities
  useAPIKeyStats,
  usePrefetchAPIKey,
  useCheckAPIKeyName,

  // Types
  type APIKeyListResponse,
  type APIKeyResponse,
  type APIKeyWithSecret,
  type CreateAPIKeyRequest,
  type UpdateAPIKeyRequest,
  type RotateAPIKeyRequest,
  type APIKeyFilters,
  type BulkAPIKeyRequest,
  type BulkAPIKeyResponse,
  type APIKeyStats,
} from './apikeys';

// ============================================================================
// Usage & Analytics
// ============================================================================

export {
  // Queries
  useUsage,
  useUsageInfinite,
  useUsageAdvanced,
  useUsageRecords,
  useUsageTrends,
  useUsageSummary,
  useExportStatus,

  // Mutations
  useCreateExport,

  // Utilities
  useUsageMetrics,
  useUsageByDimension,
  useUsageTrendsPreset,
  usePrefetchUsage,
  useInvalidateUsage,

  // Types
  type UsageRecordResponse,
  type UsageAggregation,
  type UsageSummary,
  type UsageListResponse,
  type UsageTrendPoint,
  type UsageTrendResponse,
  type UsageQueryRequest,
  type UsageExportRequest,
  type UsageExportResponse,
  type UsageTimeRange,
} from './usage';

// ============================================================================
// Providers
// ============================================================================

export {
  // Queries
  useProviders,
  useProvider,
  useProviderHealth,

  // Mutations
  useCheckProviderHealth,

  // Utilities
  useProvidersByStatus,
  useProviderStats,
  usePrefetchProvider,
  useRefreshProviders,

  // Types
  type ProviderListResponse,
  type ProviderStatusUpdate,
} from './providers';
