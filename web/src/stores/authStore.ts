/**
 * RAD Gateway Admin UI - Auth Store
 * Auth Integration Developer - Security Hardened
 *
 * Zustand store for authentication state.
 * Uses httpOnly cookies as primary storage with memory fallback.
 */

import { create } from 'zustand';
import { AuthState, User, Permission } from '../types';
import { apiClient, APIError } from '../api/client';

interface AuthStore extends AuthState {
  // Actions
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshToken: () => Promise<boolean>;
  setToken: (token: string | null) => void;
  updateUser: (user: Partial<User>) => void;
  clearError: () => void;
  hasPermission: (resource: string, action: string) => boolean;
  initialize: () => Promise<void>;
}

// Token is stored in memory only (not localStorage for XSS protection)
// httpOnly cookies handle persistence
let memoryToken: string | null = null;

export const useAuthStore = create<AuthStore>()((set, get) => ({
  // Initial state
  user: null,
  token: null,
  permissions: [],
  isAuthenticated: false,
  isLoading: true,
  error: null,

  // Initialize - check if user has valid session
  initialize: async () => {
    set({ isLoading: true, error: null });

    try {
      // Try to refresh token on load - this validates the httpOnly cookie
      const refreshed = await get().refreshToken();

      if (refreshed) {
        // Fetch user info
        const userData = await apiClient.get<User>('/v1/auth/me');
        set({
          user: userData,
          isAuthenticated: true,
          isLoading: false,
        });
      } else {
        set({ isLoading: false });
      }
    } catch {
      // No valid session
      set({ isLoading: false });
    }
  },

  // Login action
  login: async (email: string, password: string) => {
    set({ isLoading: true, error: null });

    try {
      const response = await apiClient.post<{
        user: User;
        access_token: string;
        refresh_token: string;
        expires_at: string;
      }>('/v1/auth/login', { email, password });

      // Store token in memory (httpOnly cookie already set by server)
      memoryToken = response.access_token;
      apiClient.setAuthToken(response.access_token);

      // Map backend permissions to frontend format
      const permissions: Permission[] = response.user.permissions?.map((p: string) => ({
        resource: '*',
        actions: [p as any],
      })) || [{ resource: '*', actions: ['read'] }];

      set({
        user: response.user,
        token: response.access_token,
        permissions,
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
  logout: async () => {
    try {
      // Call logout endpoint to clear httpOnly cookies server-side
      await apiClient.post('/v1/auth/logout');
    } catch {
      // Ignore errors on logout
    }

    // Clear memory token
    memoryToken = null;
    apiClient.setAuthToken(null);

    set({
      user: null,
      token: null,
      permissions: [],
      isAuthenticated: false,
      error: null,
    });
  },

  // Refresh token action
  refreshToken: async () => {
    try {
      const response = await apiClient.post<{
        access_token: string;
        refresh_token: string;
        expires_at: string;
      }>('/v1/auth/refresh');

      // Update memory token (httpOnly cookie already updated by server)
      memoryToken = response.access_token;
      apiClient.setAuthToken(response.access_token);

      set({
        token: response.access_token,
        isAuthenticated: true,
      });

      return true;
    } catch {
      // Clear invalid session
      memoryToken = null;
      apiClient.setAuthToken(null);
      set({
        token: null,
        isAuthenticated: false,
      });
      return false;
    }
  },

  // Set token directly (for external auth)
  setToken: (token: string | null) => {
    memoryToken = token;
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
    const { permissions, user } = get();

    // Admin has all permissions
    if (user?.role === 'admin') {
      return true;
    }

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
}));

// Selector hooks for better performance
export const useAuthToken = () => useAuthStore((state) => state.token);
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated);
export const useAuthUser = () => useAuthStore((state) => state.user);
export const useAuthLoading = () => useAuthStore((state) => state.isLoading);
export const useAuthError = () => useAuthStore((state) => state.error);
export const useHasPermission = (resource: string, action: string) =>
  useAuthStore((state) => state.hasPermission(resource, action));

// Helper to get token from memory (for API calls)
export const getAuthToken = () => memoryToken;
