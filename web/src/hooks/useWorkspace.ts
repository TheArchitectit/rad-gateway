/**
 * RAD Gateway Admin UI - Workspace Hooks
 * State Management Engineer - Phase 2 Implementation
 *
 * Custom hooks for workspace state and operations.
 */

import { useCallback, useEffect, useMemo } from 'react';
import { useWorkspaceStore } from '../stores/workspaceStore';
import { Workspace } from '../types';

interface UseWorkspaceReturn {
  current: Workspace | null;
  workspaces: Workspace[];
  recent: Workspace[];
  favorites: string[];
  favoriteWorkspaces: Workspace[];
  isLoading: boolean;
  error: string | null;
  setCurrent: (workspace: Workspace | null) => void;
  addToFavorites: (id: string) => void;
  removeFromFavorites: (id: string) => void;
  toggleFavorite: (id: string) => void;
  clearError: () => void;
  refresh: () => Promise<void>;
}

/**
 * Hook for workspace state and operations.
 * Provides access to workspaces, current selection, and favorites.
 */
export function useWorkspace(): UseWorkspaceReturn {
  const store = useWorkspaceStore();

  const favoriteWorkspaces = useMemo(() => {
    return store.list.filter((w) => store.favorites.includes(w.id));
  }, [store.list, store.favorites]);

  const toggleFavorite = useCallback(
    (id: string) => {
      if (store.favorites.includes(id)) {
        store.removeFromFavorites(id);
      } else {
        store.addToFavorites(id);
      }
    },
    [store.favorites, store.addToFavorites, store.removeFromFavorites]
  );

  const refresh = useCallback(async () => {
    await store.fetchWorkspaces();
  }, [store.fetchWorkspaces]);

  return {
    current: store.current,
    workspaces: store.list,
    recent: store.recent,
    favorites: store.favorites,
    favoriteWorkspaces,
    isLoading: store.isLoading,
    error: store.error,
    setCurrent: store.setCurrent,
    addToFavorites: store.addToFavorites,
    removeFromFavorites: store.removeFromFavorites,
    toggleFavorite,
    clearError: store.clearError,
    refresh,
  };
}

interface UseWorkspaceActionsReturn {
  create: (data: Partial<Workspace>) => Promise<Workspace>;
  update: (id: string, updates: Partial<Workspace>) => Promise<void>;
  delete: (id: string) => Promise<void>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for workspace CRUD operations.
 * Provides create, update, and delete functionality.
 */
export function useWorkspaceActions(): UseWorkspaceActionsReturn {
  const store = useWorkspaceStore();

  return {
    create: store.createWorkspace,
    update: store.updateWorkspaceRemote,
    delete: store.deleteWorkspace,
    isLoading: store.isLoading,
    error: store.error,
  };
}

interface UseCurrentWorkspaceReturn {
  workspace: Workspace | null;
  isLoading: boolean;
  error: string | null;
  update: (updates: Partial<Workspace>) => Promise<void>;
  leave: () => void;
}

/**
 * Hook for current workspace operations.
 * Provides easy access to the active workspace.
 */
export function useCurrentWorkspace(): UseCurrentWorkspaceReturn {
  const store = useWorkspaceStore();

  const update = useCallback(
    async (updates: Partial<Workspace>) => {
      if (!store.current) return;
      await store.updateWorkspaceRemote(store.current.id, updates);
    },
    [store.current, store.updateWorkspaceRemote]
  );

  const leave = useCallback(() => {
    store.setCurrent(null);
  }, [store.setCurrent]);

  return {
    workspace: store.current,
    isLoading: store.isLoading,
    error: store.error,
    update,
    leave,
  };
}

interface UseWorkspaceByIdReturn {
  workspace: Workspace | undefined;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook to get a specific workspace by ID.
 */
export function useWorkspaceById(id: string): UseWorkspaceByIdReturn {
  const store = useWorkspaceStore();

  const workspace = useMemo(() => {
    return store.list.find((w) => w.id === id);
  }, [store.list, id]);

  return {
    workspace,
    isLoading: store.isLoading,
    error: store.error,
  };
}

/**
 * Hook to load workspaces on mount.
 * Useful for workspace selector components.
 */
export function useWorkspacesLoader(): {
  workspaces: Workspace[];
  isLoading: boolean;
  error: string | null;
} {
  const store = useWorkspaceStore();

  useEffect(() => {
    if (store.list.length === 0 && !store.isLoading) {
      store.fetchWorkspaces();
    }
  }, [store.list.length, store.isLoading, store.fetchWorkspaces]);

  return {
    workspaces: store.list,
    isLoading: store.isLoading,
    error: store.error,
  };
}

/**
 * Hook to get workspace settings.
 */
export function useWorkspaceSettings(): {
  settings: Workspace['settings'] | null;
  updateTheme: (theme: Workspace['settings']['theme']) => void;
} {
  const { current, updateWorkspace } = useWorkspaceStore();

  const updateTheme = useCallback(
    (theme: Workspace['settings']['theme']) => {
      if (!current) return;
      updateWorkspace(current.id, {
        settings: { ...current.settings, theme },
      });
    },
    [current, updateWorkspace]
  );

  return {
    settings: current?.settings || null,
    updateTheme,
  };
}
