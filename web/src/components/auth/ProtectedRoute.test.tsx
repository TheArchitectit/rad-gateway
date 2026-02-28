/**
 * RAD Gateway Admin UI - ProtectedRoute Tests
 * Auth Integration Developer - Security Hardened
 */

import React from 'react';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { ProtectedRoute, PublicRoute } from './ProtectedRoute';

// Mock the auth store
const mockUseAuthStore = vi.fn();
vi.mock('../../stores/authStore', () => ({
  useAuthStore: () => mockUseAuthStore(),
  getAuthToken: vi.fn(),
}));

// Mock window.location and sessionStorage
Object.defineProperty(window, 'location', {
  value: { href: '', pathname: '/dashboard', search: '' },
  writable: true,
});

Object.defineProperty(window, 'sessionStorage', {
  value: {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  },
  writable: true,
});

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.location.href = '';
  });

  describe('loading state', () => {
    it('should show loading spinner when auth is loading', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: false,
        isLoading: true,
        user: null,
        hasPermission: () => false,
      });

      render(
        <ProtectedRoute>
          <div data-testid="protected-content">Protected</div>
        </ProtectedRoute>
      );

      expect(screen.getByText('Loading...')).toBeInTheDocument();
      expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument();
    });

    it('should show custom loading component when provided', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: false,
        isLoading: true,
        user: null,
        hasPermission: () => false,
      });

      render(
        <ProtectedRoute loadingComponent={<div data-testid="custom-loading">Custom Loading</div>}>
          <div>Protected</div>
        </ProtectedRoute>
      );

      expect(screen.getByTestId('custom-loading')).toBeInTheDocument();
    });
  });

  describe('authentication', () => {
    it('should render children when authenticated', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'test@example.com', role: 'developer' },
        hasPermission: () => true,
      });

      render(
        <ProtectedRoute>
          <div data-testid="protected-content">Protected Content</div>
        </ProtectedRoute>
      );

      expect(screen.getByTestId('protected-content')).toBeInTheDocument();
    });

    it('should redirect to login when not authenticated', async () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: false,
        isLoading: false,
        user: null,
        hasPermission: () => false,
      });

      render(
        <ProtectedRoute>
          <div>Protected</div>
        </ProtectedRoute>
      );

      await waitFor(() => {
        expect(window.location.href).toBe('/login');
      });
    });

    it('should store current path for redirect', () => {
      window.location.pathname = '/admin/settings';
      window.location.search = '?tab=general';

      mockUseAuthStore.mockReturnValue({
        isAuthenticated: false,
        isLoading: false,
        user: null,
        hasPermission: () => false,
      });

      render(
        <ProtectedRoute>
          <div>Protected</div>
        </ProtectedRoute>
      );

      expect(window.sessionStorage.setItem).toHaveBeenCalledWith(
        'redirectAfterLogin',
        '/admin/settings?tab=general'
      );
    });
  });

  describe('role-based access', () => {
    it('should grant access to admin for admin-only routes', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'admin@example.com', role: 'admin' },
        hasPermission: () => true,
      });

      render(
        <ProtectedRoute requiredRole="admin">
          <div data-testid="admin-content">Admin Panel</div>
        </ProtectedRoute>
      );

      expect(screen.getByTestId('admin-content')).toBeInTheDocument();
    });

    it('should deny access to developer for admin-only routes', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'dev@example.com', role: 'developer' },
        hasPermission: () => true,
      });

      render(
        <ProtectedRoute requiredRole="admin">
          <div>Admin Panel</div>
        </ProtectedRoute>
      );

      expect(screen.getByText('Access Denied')).toBeInTheDocument();
      expect(screen.getByText('This area requires admin access.')).toBeInTheDocument();
    });

    it('should grant access to higher roles', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'admin@example.com', role: 'admin' },
        hasPermission: () => true,
      });

      render(
        <ProtectedRoute requiredRole="developer">
          <div data-testid="content">Content</div>
        </ProtectedRoute>
      );

      expect(screen.getByTestId('content')).toBeInTheDocument();
    });

    it('should show custom fallback when access denied', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'dev@example.com', role: 'developer' },
        hasPermission: () => false,
      });

      render(
        <ProtectedRoute requiredRole="admin" fallback={<div data-testid="custom-denied">Custom Denied</div>}>
          <div>Admin Panel</div>
        </ProtectedRoute>
      );

      expect(screen.getByTestId('custom-denied')).toBeInTheDocument();
      expect(screen.queryByText('Access Denied')).not.toBeInTheDocument();
    });
  });

  describe('permission-based access', () => {
    it('should grant access when user has required permissions', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'user@example.com', role: 'developer' },
        hasPermission: (resource: string, action: string) => {
          return resource === 'providers' && action === 'write';
        },
      });

      render(
        <ProtectedRoute requiredPermissions={[{ resource: 'providers', action: 'write' }]}>
          <div data-testid="content">Provider Config</div>
        </ProtectedRoute>
      );

      expect(screen.getByTestId('content')).toBeInTheDocument();
    });

    it('should deny access when user lacks permissions', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'user@example.com', role: 'viewer' },
        hasPermission: () => false,
      });

      render(
        <ProtectedRoute requiredPermissions={[{ resource: 'providers', action: 'delete' }]}>
          <div>Provider Config</div>
        </ProtectedRoute>
      );

      expect(screen.getByText("Access Denied")).toBeInTheDocument();
      expect(screen.getByText("You don't have the required permissions to access this area.")).toBeInTheDocument();
    });

    it('should check all permissions when multiple required', () => {
      const hasPermission = vi.fn()
        .mockReturnValueOnce(true)  // First permission passes
        .mockReturnValueOnce(false); // Second permission fails

      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'user@example.com', role: 'developer' },
        hasPermission,
      });

      render(
        <ProtectedRoute requiredPermissions={[
          { resource: 'providers', action: 'read' },
          { resource: 'providers', action: 'delete' },
        ]}>
          <div>Content</div>
        </ProtectedRoute>
      );

      expect(screen.getByText('Access Denied')).toBeInTheDocument();
    });
  });

  describe('combined role and permission checks', () => {
    it('should require both role and permissions', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'admin@example.com', role: 'admin' },
        hasPermission: () => true,
      });

      render(
        <ProtectedRoute
          requiredRole="admin"
          requiredPermissions={[{ resource: '*', action: 'admin' }]}
        >
          <div data-testid="content">Super Admin</div>
        </ProtectedRoute>
      );

      expect(screen.getByTestId('content')).toBeInTheDocument();
    });
  });
});

describe('PublicRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    window.location.href = '';
  });

  describe('loading state', () => {
    it('should show loading spinner when auth is loading', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: false,
        isLoading: true,
        user: null,
      });

      render(
        <PublicRoute>
          <div data-testid="public-content">Login Page</div>
        </PublicRoute>
      );

      expect(document.querySelector('.animate-spin')).toBeInTheDocument();
      expect(screen.queryByTestId('public-content')).not.toBeInTheDocument();
    });
  });

  describe('authenticated users', () => {
    it('should redirect authenticated users to dashboard', async () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'user@example.com' },
      });

      render(
        <PublicRoute>
          <div>Login Page</div>
        </PublicRoute>
      );

      await waitFor(() => {
        expect(window.location.href).toBe('/dashboard');
      });
    });

    it('should redirect to stored path if exists', async () => {
      vi.mocked(window.sessionStorage.getItem).mockReturnValue('/settings/profile');

      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'user@example.com' },
      });

      render(
        <PublicRoute>
          <div>Login Page</div>
        </PublicRoute>
      );

      await waitFor(() => {
        expect(window.location.href).toBe('/settings/profile');
      });

      expect(window.sessionStorage.removeItem).toHaveBeenCalledWith('redirectAfterLogin');
    });

    it('should not render children when authenticated', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'user@example.com' },
      });

      const { container } = render(
        <PublicRoute>
          <div>Login Page</div>
        </PublicRoute>
      );

      expect(container.firstChild).toBeNull();
    });
  });

  describe('unauthenticated users', () => {
    it('should render children when not authenticated', () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: false,
        isLoading: false,
        user: null,
      });

      render(
        <PublicRoute>
          <div data-testid="public-content">Login Page</div>
        </PublicRoute>
      );

      expect(screen.getByTestId('public-content')).toBeInTheDocument();
    });

    it('should accept custom redirect path', async () => {
      mockUseAuthStore.mockReturnValue({
        isAuthenticated: true,
        isLoading: false,
        user: { id: '1', email: 'user@example.com' },
      });

      render(
        <PublicRoute redirectTo="/home">
          <div>Login Page</div>
        </PublicRoute>
      );

      await waitFor(() => {
        expect(window.location.href).toBe('/home');
      });
    });
  });
});
