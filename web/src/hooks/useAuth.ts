/**
 * RAD Gateway Admin UI - Auth Hooks
 * State Management Engineer - Phase 2 Implementation
 *
 * Custom hooks for authentication state and operations.
 */

import { useCallback, useEffect, useState } from 'react';
import { useAuthStore } from '../stores/authStore';
import { APIError } from '../api/client';

interface UseAuthReturn {
  user: ReturnType<typeof useAuthStore>['user'];
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  clearError: () => void;
  hasPermission: (resource: string, action: string) => boolean;
}

/**
 * Hook for authentication state and operations.
 * Provides login, logout, and permission checking.
 */
export function useAuth(): UseAuthReturn {
  const store = useAuthStore();

  return {
    user: store.user,
    isAuthenticated: store.isAuthenticated,
    isLoading: store.isLoading,
    error: store.error,
    login: store.login,
    logout: store.logout,
    clearError: store.clearError,
    hasPermission: store.hasPermission,
  };
}

interface UseLoginFormReturn {
  email: string;
  password: string;
  isSubmitting: boolean;
  error: string | null;
  setEmail: (email: string) => void;
  setPassword: (password: string) => void;
  submit: () => Promise<void>;
  reset: () => void;
}

/**
 * Hook for login form state and submission.
 * Manages form fields and submission handling.
 */
export function useLoginForm(): UseLoginFormReturn {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { login } = useAuth();

  const submit = useCallback(async () => {
    setIsSubmitting(true);
    setError(null);

    try {
      await login(email, password);
    } catch (err) {
      const message = err instanceof APIError
        ? err.message
        : 'Login failed. Please try again.';
      setError(message);
    } finally {
      setIsSubmitting(false);
    }
  }, [email, password, login]);

  const reset = useCallback(() => {
    setEmail('');
    setPassword('');
    setError(null);
  }, []);

  return {
    email,
    password,
    isSubmitting,
    error,
    setEmail,
    setPassword,
    submit,
    reset,
  };
}

interface UseRequireAuthReturn {
  isAuthenticated: boolean;
  isLoading: boolean;
  user: ReturnType<typeof useAuthStore>['user'];
}

/**
 * Hook to require authentication.
 * Triggers redirect to login if not authenticated.
 * Returns auth status for conditional rendering.
 */
export function useRequireAuth(redirectTo: string = '/login'): UseRequireAuthReturn {
  const { isAuthenticated, isLoading, user } = useAuth();

  useEffect(() => {
    if (!isLoading && !isAuthenticated && typeof window !== 'undefined') {
      // Store the current path for redirect after login
      const currentPath = window.location.pathname;
      if (currentPath !== redirectTo) {
        sessionStorage.setItem('redirectAfterLogin', currentPath);
        window.location.href = redirectTo;
      }
    }
  }, [isAuthenticated, isLoading, redirectTo]);

  return { isAuthenticated, isLoading, user };
}

interface UsePermissionReturn {
  hasPermission: boolean;
  isLoading: boolean;
}

/**
 * Hook to check if user has a specific permission.
 */
export function usePermission(resource: string, action: string): UsePermissionReturn {
  const { isAuthenticated, hasPermission: checkPermission } = useAuth();

  const hasPermission = isAuthenticated && checkPermission(resource, action);

  return {
    hasPermission,
    isLoading: false, // Permissions are synchronous from store
  };
}

/**
 * Hook to check if user has admin role.
 */
export function useIsAdmin(): boolean {
  const { user, isAuthenticated } = useAuth();
  return isAuthenticated && user?.role === 'admin';
}

/**
 * Hook to get the current user's display name.
 */
export function useUserDisplayName(): string {
  const { user } = useAuth();
  return user?.name || user?.email?.split('@')[0] || 'User';
}
