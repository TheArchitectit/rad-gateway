/**
 * RAD Gateway Admin UI - Auth Store Tests
 * Auth Integration Developer - Security Hardened
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useAuthStore, getAuthToken } from './authStore';

// Mock the API client
vi.mock('../api/client', () => ({
  apiClient: {
    post: vi.fn(),
    get: vi.fn(),
    setAuthToken: vi.fn(),
  },
  APIError: class APIError extends Error {
    constructor(message: string, public code: string, public status: number) {
      super(message);
    }
  },
}));

import { apiClient } from '../api/client';

describe('authStore', () => {
  beforeEach(() => {
    // Reset store state
    const store = useAuthStore.getState();
    store.logout();
    vi.clearAllMocks();
  });

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const state = useAuthStore.getState();

      expect(state.user).toBeNull();
      expect(state.token).toBeNull();
      expect(state.isAuthenticated).toBe(false);
      expect(state.isLoading).toBe(true);
      expect(state.error).toBeNull();
      expect(state.permissions).toEqual([]);
    });
  });

  describe('login', () => {
    it('should handle successful login', async () => {
      const mockResponse = {
        user: {
          id: 'user-123',
          email: 'test@example.com',
          name: 'Test User',
          role: 'admin',
          createdAt: new Date().toISOString(),
        },
        access_token: 'mock-access-token',
        refresh_token: 'mock-refresh-token',
        expires_at: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
      };

      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      const store = useAuthStore.getState();
      await store.login('test@example.com', 'password123');

      const newState = useAuthStore.getState();
      expect(newState.isAuthenticated).toBe(true);
      expect(newState.user).toEqual(mockResponse.user);
      expect(newState.token).toBe('mock-access-token');
      expect(newState.error).toBeNull();
      expect(apiClient.setAuthToken).toHaveBeenCalledWith('mock-access-token');
    });

    it('should handle login failure', async () => {
      const error = new (await import('../api/client')).APIError(
        'Invalid credentials',
        'auth_failed',
        401
      );
      vi.mocked(apiClient.post).mockRejectedValueOnce(error);

      const store = useAuthStore.getState();

      await expect(store.login('test@example.com', 'wrongpassword')).rejects.toThrow();

      const newState = useAuthStore.getState();
      expect(newState.isAuthenticated).toBe(false);
      expect(newState.user).toBeNull();
      expect(newState.error).toBe('Invalid credentials');
    });

    it('should set loading state during login', async () => {
      const mockResponse = {
        user: { id: 'user-123', email: 'test@example.com', name: 'Test', role: 'developer', createdAt: '' },
        access_token: 'token',
        refresh_token: 'refresh',
        expires_at: new Date().toISOString(),
      };

      vi.mocked(apiClient.post).mockImplementation(() =>
        new Promise((resolve) => setTimeout(() => resolve(mockResponse), 10))
      );

      const store = useAuthStore.getState();
      const loginPromise = store.login('test@example.com', 'password');

      expect(useAuthStore.getState().isLoading).toBe(true);

      await loginPromise;

      expect(useAuthStore.getState().isLoading).toBe(false);
    });
  });

  describe('logout', () => {
    it('should clear auth state on logout', async () => {
      // First login
      const mockResponse = {
        user: { id: 'user-123', email: 'test@example.com', name: 'Test', role: 'admin', createdAt: '' },
        access_token: 'token',
        refresh_token: 'refresh',
        expires_at: new Date().toISOString(),
      };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      const store = useAuthStore.getState();
      await store.login('test@example.com', 'password');

      // Then logout
      vi.mocked(apiClient.post).mockResolvedValueOnce({});
      await store.logout();

      const newState = useAuthStore.getState();
      expect(newState.isAuthenticated).toBe(false);
      expect(newState.user).toBeNull();
      expect(newState.token).toBeNull();
      expect(newState.permissions).toEqual([]);
      expect(apiClient.setAuthToken).toHaveBeenLastCalledWith(null);
    });

    it('should clear state even if logout request fails', async () => {
      // Login first
      const mockResponse = {
        user: { id: 'user-123', email: 'test@example.com', name: 'Test', role: 'admin', createdAt: '' },
        access_token: 'token',
        refresh_token: 'refresh',
        expires_at: new Date().toISOString(),
      };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      const store = useAuthStore.getState();
      await store.login('test@example.com', 'password');

      // Logout fails
      vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('Network error'));
      await store.logout();

      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });
  });

  describe('refreshToken', () => {
    it('should refresh token successfully', async () => {
      const mockResponse = {
        access_token: 'new-access-token',
        refresh_token: 'new-refresh-token',
        expires_at: new Date(Date.now() + 15 * 60 * 1000).toISOString(),
      };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      const store = useAuthStore.getState();
      const result = await store.refreshToken();

      expect(result).toBe(true);
      expect(useAuthStore.getState().token).toBe('new-access-token');
      expect(apiClient.setAuthToken).toHaveBeenCalledWith('new-access-token');
    });

    it('should clear auth state on refresh failure', async () => {
      vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('Refresh failed'));

      const store = useAuthStore.getState();
      const result = await store.refreshToken();

      expect(result).toBe(false);
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
      expect(useAuthStore.getState().token).toBeNull();
    });
  });

  describe('hasPermission', () => {
    beforeEach(async () => {
      const mockResponse = {
        user: { id: 'user-123', email: 'test@example.com', name: 'Test', role: 'developer', createdAt: '' },
        access_token: 'token',
        refresh_token: 'refresh',
        expires_at: new Date().toISOString(),
      };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      await useAuthStore.getState().login('test@example.com', 'password');
    });

    it('should grant admin all permissions', async () => {
      // Login as admin
      const adminResponse = {
        user: { id: 'user-123', email: 'admin@example.com', name: 'Admin', role: 'admin', createdAt: '' },
        access_token: 'token',
        refresh_token: 'refresh',
        expires_at: new Date().toISOString(),
      };
      vi.mocked(apiClient.post).mockResolvedValueOnce(adminResponse);

      const store = useAuthStore.getState();
      await store.login('admin@example.com', 'password');

      expect(store.hasPermission('any-resource', 'any-action')).toBe(true);
      expect(store.hasPermission('providers', 'delete')).toBe(true);
    });

    it('should check specific permissions', () => {
      const store = useAuthStore.getState();
      // Developer role has read/write permissions
      expect(store.hasPermission('*', 'read')).toBe(true);
      expect(store.hasPermission('*', 'write')).toBe(true);
    });

    it('should handle wildcard resource patterns', () => {
      const store = useAuthStore.getState();
      // Admin has all permissions
      store.updateUser({ role: 'admin' });
      expect(store.hasPermission('providers:openai', 'write')).toBe(true);
    });
  });

  describe('updateUser', () => {
    it('should update user data', async () => {
      const mockResponse = {
        user: { id: 'user-123', email: 'test@example.com', name: 'Test', role: 'developer', createdAt: '' },
        access_token: 'token',
        refresh_token: 'refresh',
        expires_at: new Date().toISOString(),
      };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      const store = useAuthStore.getState();
      await store.login('test@example.com', 'password');

      store.updateUser({ name: 'Updated Name' });

      expect(useAuthStore.getState().user?.name).toBe('Updated Name');
    });

    it('should not update if user is null', () => {
      const store = useAuthStore.getState();
      store.logout();

      // Should not throw
      store.updateUser({ name: 'New Name' });

      expect(useAuthStore.getState().user).toBeNull();
    });
  });

  describe('setToken', () => {
    it('should set token and auth state', () => {
      const store = useAuthStore.getState();
      store.setToken('new-token');

      expect(useAuthStore.getState().token).toBe('new-token');
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
      expect(apiClient.setAuthToken).toHaveBeenCalledWith('new-token');
    });

    it('should clear auth when token is null', () => {
      const store = useAuthStore.getState();
      store.setToken('token');
      store.setToken(null);

      expect(useAuthStore.getState().token).toBeNull();
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });
  });

  describe('clearError', () => {
    it('should clear error state', async () => {
      const error = new (await import('../api/client')).APIError(
        'Error message',
        'error_code',
        400
      );
      vi.mocked(apiClient.post).mockRejectedValueOnce(error);

      const store = useAuthStore.getState();
      try {
        await store.login('test@example.com', 'password');
      } catch {
        // Expected
      }

      expect(useAuthStore.getState().error).toBe('Error message');

      store.clearError();

      expect(useAuthStore.getState().error).toBeNull();
    });
  });

  describe('getAuthToken', () => {
    it('should return the current token from memory', async () => {
      const mockResponse = {
        user: { id: 'user-123', email: 'test@example.com', name: 'Test', role: 'admin', createdAt: '' },
        access_token: 'memory-token-123',
        refresh_token: 'refresh',
        expires_at: new Date().toISOString(),
      };
      vi.mocked(apiClient.post).mockResolvedValueOnce(mockResponse);

      await useAuthStore.getState().login('test@example.com', 'password');

      expect(getAuthToken()).toBe('memory-token-123');
    });

    it('should return null when not authenticated', () => {
      useAuthStore.getState().logout();
      expect(getAuthToken()).toBeNull();
    });
  });
});
