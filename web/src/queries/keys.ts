/**
 * RAD Gateway Admin UI - Query Keys
 * Data Fetching Developer - Phase 3 Implementation
 *
 * Centralized query key management for cache invalidation.
 * Follows the query key factory pattern for type-safe cache management.
 */

import { UsageFilters } from '../types';

// ============================================================================
// Projects / Workspaces Query Keys
// ============================================================================

export const projectsKeys = {
  all: ['projects'] as const,
  lists: () => [...projectsKeys.all, 'list'] as const,
  list: (filters: { status?: string; search?: string; page?: number; pageSize?: number }) =>
    [...projectsKeys.lists(), filters] as const,
  details: () => [...projectsKeys.all, 'detail'] as const,
  detail: (id: string) => [...projectsKeys.details(), id] as const,
  stream: () => [...projectsKeys.all, 'stream'] as const,
};

// ============================================================================
// API Keys Query Keys
// ============================================================================

export const apiKeysKeys = {
  all: ['apiKeys'] as const,
  lists: () => [...apiKeysKeys.all, 'list'] as const,
  list: (filters: { status?: string; workspaceId?: string; search?: string; page?: number; pageSize?: number }) =>
    [...apiKeysKeys.lists(), filters] as const,
  details: () => [...apiKeysKeys.all, 'detail'] as const,
  detail: (id: string) => [...apiKeysKeys.details(), id] as const,
  stats: () => [...apiKeysKeys.all, 'stats'] as const,
};

// ============================================================================
// Usage Query Keys
// ============================================================================

export const usageKeys = {
  all: ['usage'] as const,
  lists: () => [...usageKeys.all, 'list'] as const,
  list: (filters: UsageFilters & { page?: number; pageSize?: number }) =>
    [...usageKeys.lists(), filters] as const,
  records: () => [...usageKeys.all, 'records'] as const,
  record: (filters: { workspaceId?: string; apiKeyId?: string; startTime?: string; endTime?: string }) =>
    [...usageKeys.records(), filters] as const,
  trends: () => [...usageKeys.all, 'trends'] as const,
  trend: (params: { startTime?: string; endTime?: string; interval?: 'minute' | 'hour' | 'day' }) =>
    [...usageKeys.trends(), params] as const,
  summary: () => [...usageKeys.all, 'summary'] as const,
  summaryDetail: (filters: { workspaceId?: string; startTime?: string; endTime?: string }) =>
    [...usageKeys.summary(), filters] as const,
  aggregations: () => [...usageKeys.all, 'aggregations'] as const,
  aggregation: (groupBy: string[], filters: UsageFilters) =>
    [...usageKeys.aggregations(), { groupBy, filters }] as const,
  exports: () => [...usageKeys.all, 'exports'] as const,
  export: (exportId: string) => [...usageKeys.exports(), exportId] as const,
};

// ============================================================================
// Providers Query Keys
// ============================================================================

export const providersKeys = {
  all: ['providers'] as const,
  lists: () => [...providersKeys.all, 'list'] as const,
  list: () => [...providersKeys.lists()] as const,
  details: () => [...providersKeys.all, 'detail'] as const,
  detail: (name: string) => [...providersKeys.details(), name] as const,
  health: (name: string) => [...providersKeys.detail(name), 'health'] as const,
};

// ============================================================================
// Model Routes Query Keys
// ============================================================================

export const modelRoutesKeys = {
  all: ['modelRoutes'] as const,
  lists: () => [...modelRoutesKeys.all, 'list'] as const,
  list: () => [...modelRoutesKeys.lists()] as const,
  details: () => [...modelRoutesKeys.all, 'detail'] as const,
  detail: (modelId: string) => [...modelRoutesKeys.details(), modelId] as const,
};

// ============================================================================
// Admin Configuration Query Keys
// ============================================================================

export const adminKeys = {
  all: ['admin'] as const,
  config: () => [...adminKeys.all, 'config'] as const,
  health: () => [...adminKeys.all, 'health'] as const,
  detailedHealth: () => [...adminKeys.all, 'health', 'detailed'] as const,
  status: () => [...adminKeys.all, 'status'] as const,
  maintenance: () => [...adminKeys.all, 'maintenance'] as const,
};

// ============================================================================
// Control Rooms Query Keys
// ============================================================================

export const controlRoomsKeys = {
  all: ['controlRooms'] as const,
  lists: () => [...controlRoomsKeys.all, 'list'] as const,
  list: () => [...controlRoomsKeys.lists()] as const,
  details: () => [...controlRoomsKeys.all, 'detail'] as const,
  detail: (id: string) => [...controlRoomsKeys.details(), id] as const,
};
