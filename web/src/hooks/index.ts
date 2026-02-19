/**
 * RAD Gateway Admin UI - Hooks
 * State Management Engineer - Phase 2 Implementation
 *
 * Export all custom hooks from a single entry point.
 */

// Auth hooks
export {
  useAuth,
  useLoginForm,
  useRequireAuth,
  usePermission,
  useIsAdmin,
  useUserDisplayName,
} from './useAuth';

// Workspace hooks
export {
  useWorkspace,
  useWorkspaceActions,
  useCurrentWorkspace,
  useWorkspaceById,
  useWorkspacesLoader,
  useWorkspaceSettings,
} from './useWorkspace';

// Provider hooks
export {
  useProviders,
  useProvider,
  useProviderStats,
  useProvidersByStatus,
} from './useProviders';

// API Key hooks
export { useApiKeys, useApiKey, useApiKeyStats } from './useApiKeys';

// Usage hooks
export {
  useUsage,
  useUsageMetrics,
  useUsageTimeSeries,
  useUsageByProvider,
  useUsageByModel,
} from './useUsage';

// UI hooks
export {
  useThemeManager,
  useSidebar,
  useModal,
  useNotificationsManager,
  useLoading,
  useDebounce,
  useLocalStorage,
  useMediaQuery,
} from './useUI';

// Data fetching hooks
export { useAsync, useFetch, usePagination } from './useAsync';

// Real-time hooks
export {
  useSSE,
  useSSEEvent,
  useSSEEvents,
} from './useSSE';

export {
  useRealtimeMetrics,
  useRealtimeMetric,
  useProviderHealth,
  useCircuitBreaker,
  useSystemAlerts,
  isMetricHealthy,
  formatMetric,
} from './useRealtimeMetrics';
