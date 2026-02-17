/**
 * RAD Gateway Admin UI - Auth Store
 * State Management Engineer - Phase 2 Implementation
 *
 * Zustand store for authentication state.
 * Simple and focused - handles auth state and user session.
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { AuthState, User, Permission } from '../types';
import { apiClient, APIError } from '../api/client';

interface AuthStore extends AuthState {
  // Actions
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  setToken: (token: string | null) => void;
  updateUser: (user: Partial<User>) => void;
  clearError: () => void;
  hasPermission: (resource: string, action: string) => boolean;
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set, get) => ({
      // Initial state
      user: null,
      token: null,
      permissions: [],
      isAuthenticated: false,
      isLoading: false,
      error: null,

      // Login action
      login: async (email: string, password: string) => {
        set({ isLoading: true, error: null });

        try {
          // TODO: Replace with actual login endpoint when available
          // For now, simulate login with mock data
          await new Promise(resolve => setTimeout(resolve, 500));

          const mockUser: User = {
            id: 'user-1',
            email,
            name: email.split('@')[0],
            role: 'admin',
            createdAt: new Date().toISOString(),
            lastLoginAt: new Date().toISOString(),
          };

          const mockToken = 'mock-jwt-token-' + Date.now();
          const mockPermissions: Permission[] = [
            { resource: '*', actions: ['read', 'write', 'delete', 'admin'] },
          ];

          // Set token on API client
          apiClient.setAuthToken(mockToken);

          set({
            user: mockUser,
            token: mockToken,
            permissions: mockPermissions,
            isAuthenticated: true,
            isLoading: false,
            error: null,
          });
        } catch (err) {
          const message = err instanceof APIError
            ? err.message
            : err instanceof Error
              ? err.message
              : 'Login failed';

          set({
            isLoading: false,
            error: message,
            isAuthenticated: false,
          });

          throw err;
        }
      },

      // Logout action
      logout: () => {
        apiClient.setAuthToken(null);

        set({
          user: null,
          token: null,
          permissions: [],
          isAuthenticated: false,
          error: null,
        });
      },

      // Set token directly (for token refresh or external auth)
      setToken: (token: string | null) => {
        apiClient.setAuthToken(token);

        set({
          token,
          isAuthenticated: !!token,
        });
      },

      // Update user data
      updateUser: (userData: Partial<User>) => {
        const currentUser = get().user;
        if (!currentUser) return;

        set({
          user: { ...currentUser, ...userData },
        });
      },

      // Clear error
      clearError: () => {
        set({ error: null });
      },

      // Check permission
      hasPermission: (resource: string, action: string): boolean => {
        const { permissions } = get();

        return permissions.some(permission => {
          // Wildcard resource permission
          if (permission.resource === '*') {
            return permission.actions.includes(action as any) ||
                   permission.actions.includes('admin');
          }

          // Exact resource match
          if (permission.resource === resource) {
            return permission.actions.includes(action as any) ||
                   permission.actions.includes('admin');
          }

          // Resource pattern match (e.g., 'providers:*' matches 'providers:openai')
          if (permission.resource.endsWith(':*')) {
            const prefix = permission.resource.slice(0, -2);
            if (resource.startsWith(prefix)) {
              return permission.actions.includes(action as any) ||
                     permission.actions.includes('admin');
            }
          }

          return false;
        });
      },
    }),
    {
      name: 'rad-auth-storage',
      partialize: (state) => ({
        token: state.token,
        user: state.user,
        permissions: state.permissions,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

// Selector hooks for better performance
export const useAuthToken = () => useAuthStore((state) => state.token);
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);
export const useAuthUser = () => useAuthStore((state) => state.user);
export const useAuthLoading = () => useAuthStore((state) => state.isLoading);
export const useAuthError = () => useAuthStore((state) => state.error);
export const useHasPermission = (resource: string, action: string) =>
  useAuthStore((state) => state.hasPermission(resource, action));
