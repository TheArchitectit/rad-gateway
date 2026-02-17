/**
 * RAD Gateway Admin UI - Workspace Store
 * State Management Engineer - Phase 2 Implementation
 *
 * Zustand store for workspace state.
 * Manages current workspace context and workspace list.
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { WorkspaceState, Workspace } from '../types';
import { APIError } from '../api/client';

interface WorkspaceStore extends WorkspaceState {
  // Actions
  setCurrent: (workspace: Workspace | null) => void;
  setWorkspaces: (workspaces: Workspace[]) => void;
  addWorkspace: (workspace: Workspace) => void;
  updateWorkspace: (id: string, updates: Partial<Workspace>) => void;
  removeWorkspace: (id: string) => void;
  addToFavorites: (id: string) => void;
  removeFromFavorites: (id: string) => void;
  addToRecent: (workspace: Workspace) => void;
  clearError: () => void;
  setLoading: (loading: boolean) => void;

  // Async actions
  fetchWorkspaces: () => Promise<void>;
  createWorkspace: (data: Partial<Workspace>) => Promise<Workspace>;
  updateWorkspaceRemote: (id: string, updates: Partial<Workspace>) => Promise<void>;
  deleteWorkspace: (id: string) => Promise<void>;
}

const MAX_RECENT = 5;

export const useWorkspaceStore = create<WorkspaceStore>()(
  persist(
    (set, get) => ({
      // Initial state
      current: null,
      list: [],
      recent: [],
      favorites: [],
      isLoading: false,
      error: null,

      // Set current workspace
      setCurrent: (workspace: Workspace | null) => {
        if (workspace) {
          get().addToRecent(workspace);
        }
        set({ current: workspace });
      },

      // Set workspace list
      setWorkspaces: (workspaces: Workspace[]) => {
        set({ list: workspaces });
      },

      // Add workspace
      addWorkspace: (workspace: Workspace) => {
        const { list } = get();
        const exists = list.find(w => w.id === workspace.id);

        if (!exists) {
          set({ list: [...list, workspace] });
        }
      },

      // Update workspace locally
      updateWorkspace: (id: string, updates: Partial<Workspace>) => {
        const { list, current } = get();

        const updatedList = list.map(w =>
          w.id === id ? { ...w, ...updates } : w
        );

        const updatedCurrent = current?.id === id
          ? { ...current, ...updates }
          : current;

        set({
          list: updatedList,
          current: updatedCurrent,
        });
      },

      // Remove workspace
      removeWorkspace: (id: string) => {
        const { list, current, favorites } = get();

        set({
          list: list.filter(w => w.id !== id),
          current: current?.id === id ? null : current,
          favorites: favorites.filter(f => f !== id),
        });
      },

      // Add to favorites
      addToFavorites: (id: string) => {
        const { favorites } = get();
        if (!favorites.includes(id)) {
          set({ favorites: [...favorites, id] });
        }
      },

      // Remove from favorites
      removeFromFavorites: (id: string) => {
        const { favorites } = get();
        set({ favorites: favorites.filter(f => f !== id) });
      },

      // Add to recent (keep only MAX_RECENT)
      addToRecent: (workspace: Workspace) => {
        const { recent } = get();
        const filtered = recent.filter(w => w.id !== workspace.id);
        const updatedRecent = [workspace, ...filtered].slice(0, MAX_RECENT);

        set({ recent: updatedRecent });
      },

      // Clear error
      clearError: () => {
        set({ error: null });
      },

      // Set loading
      setLoading: (loading: boolean) => {
        set({ isLoading: loading });
      },

      // Fetch workspaces from API
      fetchWorkspaces: async () => {
        set({ isLoading: true, error: null });

        try {
          // TODO: Replace with actual API call when endpoint is available
          // For now, simulate with mock data
          await new Promise(resolve => setTimeout(resolve, 300));

          const mockWorkspaces: Workspace[] = [
            {
              id: 'ws-1',
              name: 'Production',
              slug: 'production',
              description: 'Production environment workspace',
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
              ownerId: 'user-1',
              memberCount: 12,
              settings: {
                theme: 'dark',
                timezone: 'UTC',
                currency: 'USD',
                dateFormat: 'YYYY-MM-DD',
              },
            },
            {
              id: 'ws-2',
              name: 'Staging',
              slug: 'staging',
              description: 'Staging environment for testing',
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
              ownerId: 'user-1',
              memberCount: 5,
              settings: {
                theme: 'light',
                timezone: 'UTC',
                currency: 'USD',
                dateFormat: 'YYYY-MM-DD',
              },
            },
            {
              id: 'ws-3',
              name: 'Development',
              slug: 'development',
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
              ownerId: 'user-1',
              memberCount: 3,
              settings: {
                theme: 'system',
                timezone: 'UTC',
                currency: 'USD',
                dateFormat: 'YYYY-MM-DD',
              },
            },
          ];

          set({
            list: mockWorkspaces,
            isLoading: false,
          });

          // Set first workspace as current if none selected
          const { current } = get();
          if (!current && mockWorkspaces.length > 0) {
            set({ current: mockWorkspaces[0] });
          }
        } catch (err) {
          const message = err instanceof APIError
            ? err.message
            : err instanceof Error
              ? err.message
              : 'Failed to fetch workspaces';

          set({
            isLoading: false,
            error: message,
          });

          throw err;
        }
      },

      // Create workspace
      createWorkspace: async (data: Partial<Workspace>) => {
        set({ isLoading: true, error: null });

        try {
          // TODO: Replace with actual API call
          await new Promise(resolve => setTimeout(resolve, 500));

          const newWorkspace: Workspace = {
            id: `ws-${Date.now()}`,
            name: data.name || 'New Workspace',
            slug: data.slug || `workspace-${Date.now()}`,
            description: data.description,
            logo: data.logo,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            ownerId: 'user-1',
            memberCount: 1,
            settings: data.settings || {
              theme: 'system',
              timezone: 'UTC',
              currency: 'USD',
              dateFormat: 'YYYY-MM-DD',
            },
          };

          get().addWorkspace(newWorkspace);
          set({ isLoading: false });

          return newWorkspace;
        } catch (err) {
          const message = err instanceof APIError
            ? err.message
            : err instanceof Error
              ? err.message
              : 'Failed to create workspace';

          set({
            isLoading: false,
            error: message,
          });

          throw err;
        }
      },

      // Update workspace (remote)
      updateWorkspaceRemote: async (id: string, updates: Partial<Workspace>) => {
        set({ isLoading: true, error: null });

        try {
          // TODO: Replace with actual API call
          await new Promise(resolve => setTimeout(resolve, 300));

          get().updateWorkspace(id, { ...updates, updatedAt: new Date().toISOString() });
          set({ isLoading: false });
        } catch (err) {
          const message = err instanceof APIError
            ? err.message
            : err instanceof Error
              ? err.message
              : 'Failed to update workspace';

          set({
            isLoading: false,
            error: message,
          });

          throw err;
        }
      },

      // Delete workspace
      deleteWorkspace: async (id: string) => {
        set({ isLoading: true, error: null });

        try {
          // TODO: Replace with actual API call
          await new Promise(resolve => setTimeout(resolve, 300));

          get().removeWorkspace(id);
          set({ isLoading: false });
        } catch (err) {
          const message = err instanceof APIError
            ? err.message
            : err instanceof Error
              ? err.message
              : 'Failed to delete workspace';

          set({
            isLoading: false,
            error: message,
          });

          throw err;
        }
      },
    }),
    {
      name: 'rad-workspace-storage',
      partialize: (state) => ({
        current: state.current,
        recent: state.recent,
        favorites: state.favorites,
      }),
    }
  )
);

// Selector hooks
export const useCurrentWorkspace = () => useWorkspaceStore((state) => state.current);
export const useWorkspaces = () => useWorkspaceStore((state) => state.list);
export const useRecentWorkspaces = () => useWorkspaceStore((state) => state.recent);
export const useFavoriteWorkspaces = () => useWorkspaceStore((state) => state.favorites);
export const useWorkspaceLoading = () => useWorkspaceStore((state) => state.isLoading);
export const useWorkspaceError = () => useWorkspaceStore((state) => state.error);
