/**
 * RAD Gateway Admin UI - Stores
 * State Management Engineer - Phase 2 Implementation
 *
 * Export all Zustand stores from a single entry point.
 */

export { useAuthStore, useAuthToken, useIsAuthenticated, useAuthUser, useAuthLoading, useAuthError, useHasPermission } from './authStore';
export { useWorkspaceStore, useCurrentWorkspace, useWorkspaces, useRecentWorkspaces, useFavoriteWorkspaces, useWorkspaceLoading, useWorkspaceError } from './workspaceStore';
export { useUIStore, useSidebarCollapsed, useTheme, useNotifications, useUnreadNotificationCount, useActiveModal, useGlobalSearchOpen, useGlobalLoading, showNotification } from './uiStore';
