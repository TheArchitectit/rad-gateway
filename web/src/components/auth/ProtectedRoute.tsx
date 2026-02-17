/**
 * RAD Gateway Admin UI - ProtectedRoute Component
 * Auth Integration Developer - Security Hardened
 *
 * Route protection component that redirects unauthenticated users to login.
 * Supports role-based access control and loading states.
 */

import React, { useEffect } from 'react';
import { useAuthStore } from '../../stores/authStore';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requiredRole?: 'admin' | 'developer' | 'viewer';
  requiredPermissions?: Array<{ resource: string; action: string }>;
  fallback?: React.ReactNode;
  loadingComponent?: React.ReactNode;
}

/**
 * ProtectedRoute - Protects routes requiring authentication.
 *
 * Security features:
 * - Redirects unauthenticated users to login
 * - Supports role-based access control
 * - Validates permissions before rendering
 * - Stores intended destination for post-login redirect
 * - Shows loading state during auth initialization
 *
 * @example
 * <ProtectedRoute>
 *   <Dashboard />
 * </ProtectedRoute>
 *
 * @example
 * <ProtectedRoute requiredRole="admin">
 *   <AdminPanel />
 * </ProtectedRoute>
 */
export function ProtectedRoute({
  children,
  requiredRole,
  requiredPermissions,
  fallback,
  loadingComponent,
}: ProtectedRouteProps) {
  const { isAuthenticated, isLoading, user, hasPermission } = useAuthStore();

  // Store the current path for redirect after login
  useEffect(() => {
    if (!isLoading && !isAuthenticated && typeof window !== 'undefined') {
      const currentPath = window.location.pathname + window.location.search;
      if (currentPath !== '/login') {
        sessionStorage.setItem('redirectAfterLogin', currentPath);
      }
    }
  }, [isAuthenticated, isLoading]);

  // Handle redirect on auth failure
  useEffect(() => {
    if (!isLoading && !isAuthenticated && typeof window !== 'undefined') {
      window.location.href = '/login';
    }
  }, [isAuthenticated, isLoading]);

  // Show loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        {loadingComponent || (
          <div className="flex flex-col items-center space-y-4">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600" />
            <p className="text-gray-600">Loading...</p>
          </div>
        )}
      </div>
    );
  }

  // Redirect if not authenticated
  if (!isAuthenticated) {
    return null;
  }

  // Check role requirement
  if (requiredRole) {
    const roleHierarchy = { admin: 3, developer: 2, viewer: 1 };
    const userRoleLevel = roleHierarchy[user?.role || 'viewer'] || 0;
    const requiredRoleLevel = roleHierarchy[requiredRole] || 0;

    if (userRoleLevel < requiredRoleLevel) {
      return (
        <AccessDenied
          message={`This area requires ${requiredRole} access.`}
          fallback={fallback}
        />
      );
    }
  }

  // Check permission requirements
  if (requiredPermissions && requiredPermissions.length > 0) {
    const hasAllPermissions = requiredPermissions.every(({ resource, action }) =>
      hasPermission(resource, action)
    );

    if (!hasAllPermissions) {
      return (
        <AccessDenied
          message="You don't have the required permissions to access this area."
          fallback={fallback}
        />
      );
    }
  }

  // All checks passed, render children
  return <>{children}</>;
}

interface AccessDeniedProps {
  message?: string;
  fallback?: React.ReactNode;
}

/**
 * AccessDenied - Displays access denied message.
 */
function AccessDenied({ message, fallback }: AccessDeniedProps) {
  if (fallback) {
    return <>{fallback}</>;
  }

  return (
    <div className="flex flex-col items-center justify-center min-h-screen p-4">
      <div className="text-center max-w-md">
        <div className="mb-6">
          <svg
            className="mx-auto h-16 w-16 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            aria-hidden="true"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
            />
          </svg>
        </div>
        <h2 className="text-2xl font-bold text-gray-900 mb-2">Access Denied</h2>
        <p className="text-gray-600 mb-6">
          {message || "You don't have permission to access this area."}
        </p>
        <div className="flex justify-center space-x-4">
          <button
            onClick={() => window.history.back()}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            Go Back
          </button>
          <a
            href="/"
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            Dashboard
          </a>
        </div>
      </div>
    </div>
  );
}

interface PublicRouteProps {
  children: React.ReactNode;
  redirectTo?: string;
}

/**
 * PublicRoute - Redirects authenticated users away from public pages.
 *
 * @example
 * <PublicRoute redirectTo="/dashboard">
 *   <LoginPage />
 * </PublicRoute>
 */
export function PublicRoute({ children, redirectTo = '/dashboard' }: PublicRouteProps) {
  const { isAuthenticated, isLoading } = useAuthStore();

  useEffect(() => {
    if (!isLoading && isAuthenticated && typeof window !== 'undefined') {
      // Check for stored redirect path
      const redirectAfterLogin = sessionStorage.getItem('redirectAfterLogin');
      if (redirectAfterLogin) {
        sessionStorage.removeItem('redirectAfterLogin');
        window.location.href = redirectAfterLogin;
      } else {
        window.location.href = redirectTo;
      }
    }
  }, [isAuthenticated, isLoading, redirectTo]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (isAuthenticated) {
    return null;
  }

  return <>{children}</>;
}

export default ProtectedRoute;
