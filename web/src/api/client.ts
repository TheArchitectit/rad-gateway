/**
 * RAD Gateway Admin UI - API Client
 * State Management Engineer - Phase 2 Implementation
 *
 * Simple fetch-based API client with error handling and authentication.
 * Keeps it minimal - we can add complexity later if needed.
 */

import { ApiError } from '../types';

// API base URL configuration
// In development, we use the proxy configured in next.config.js
// In production, use the actual backend URL
const isDevelopment = process.env.NODE_ENV === 'development';
const API_BASE_URL = isDevelopment
  ? '/api/proxy'
  : (process.env['NEXT_PUBLIC_API_URL'] || 'http://172.16.30.45:8090');

interface RequestConfig extends RequestInit {
  params?: Record<string, string | number | boolean | undefined> | undefined;
}

class APIClient {
  private baseUrl: string;
  private authToken: string | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
  }

  setAuthToken(token: string | null) {
    this.authToken = token;
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };

    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    }

    return headers;
  }

  private buildUrl(endpoint: string, params?: Record<string, string | number | boolean | undefined>): string {
    const url = new URL(`${this.baseUrl}${endpoint}`);

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.set(key, String(value));
        }
      });
    }

    return url.toString();
  }

  private async handleResponse<T>(response: Response): Promise<T> {
    if (!response.ok) {
      const errorData: ApiError = await response.json().catch(() => ({
        error: {
          code: 'unknown_error',
          message: 'An unexpected error occurred',
        },
      }));

      throw new APIError(
        errorData.error.message,
        errorData.error.code,
        response.status,
        errorData.error.details,
        errorData.error.requestId
      );
    }

    // Handle 204 No Content
    if (response.status === 204) {
      return undefined as T;
    }

    return response.json();
  }

  async get<T>(endpoint: string, config: RequestConfig = {}): Promise<T> {
    const url = this.buildUrl(endpoint, config.params);

    const response = await fetch(url, {
      ...config,
      method: 'GET',
      headers: {
        ...this.getHeaders(),
        ...config.headers,
      },
    });

    return this.handleResponse<T>(response);
  }

  async post<T>(endpoint: string, data?: unknown, config: RequestConfig = {}): Promise<T> {
    const url = this.buildUrl(endpoint, config.params);

    const response = await fetch(url, {
      ...config,
      method: 'POST',
      headers: {
        ...this.getHeaders(),
        ...config.headers,
      },
      body: data ? JSON.stringify(data) : null,
    });

    return this.handleResponse<T>(response);
  }

  async put<T>(endpoint: string, data?: unknown, config: RequestConfig = {}): Promise<T> {
    const url = this.buildUrl(endpoint, config.params);

    const response = await fetch(url, {
      ...config,
      method: 'PUT',
      headers: {
        ...this.getHeaders(),
        ...config.headers,
      },
      body: data ? JSON.stringify(data) : null,
    });

    return this.handleResponse<T>(response);
  }

  async patch<T>(endpoint: string, data?: unknown, config: RequestConfig = {}): Promise<T> {
    const url = this.buildUrl(endpoint, config.params);

    const response = await fetch(url, {
      ...config,
      method: 'PATCH',
      headers: {
        ...this.getHeaders(),
        ...config.headers,
      },
      body: data ? JSON.stringify(data) : null,
    });

    return this.handleResponse<T>(response);
  }

  async delete<T>(endpoint: string, config: RequestConfig = {}): Promise<T> {
    const url = this.buildUrl(endpoint, config.params);

    const response = await fetch(url, {
      ...config,
      method: 'DELETE',
      headers: {
        ...this.getHeaders(),
        ...config.headers,
      },
    });

    return this.handleResponse<T>(response);
  }
}

export class APIError extends Error {
  constructor(
    message: string,
    public code: string,
    public status: number,
    public details?: Record<string, unknown>,
    public requestId?: string
  ) {
    super(message);
    this.name = 'APIError';
  }

  isUnauthorized(): boolean {
    return this.status === 401;
  }

  isForbidden(): boolean {
    return this.status === 403;
  }

  isNotFound(): boolean {
    return this.status === 404;
  }

  isValidationError(): boolean {
    return this.status === 400 || this.status === 422;
  }
}

export const apiClient = new APIClient(API_BASE_URL);

// ============================================================================
// Admin API Endpoints
// ============================================================================

export const adminAPI = {
  // Health
  getHealth: () => apiClient.get<HealthResponse>('/health'),
  getDetailedHealth: () => apiClient.get<DetailedHealthResponse>('/v0/admin/health/detailed'),
  getGatewayStatus: () => apiClient.get<GatewayStatus>('/v0/admin/status'),

  // Configuration
  getConfig: () => apiClient.get<GatewayConfig>('/v0/admin/config'),
  updateConfig: (data: ConfigUpdateRequest) => apiClient.put<GatewayConfig>('/v0/admin/config', data),
  reloadConfig: () => apiClient.post<{ reloaded: boolean; timestamp: string }>('/v0/admin/config/reload'),

  // Model Routes
  getModelRoutes: () => apiClient.get<{ routes: ModelRoute[] }>('/v0/admin/model-routes'),
  getModelRoute: (modelId: string) => apiClient.get<ModelRoute>(`/v0/admin/model-routes/${modelId}`),
  createModelRoute: (data: ModelRoute) => apiClient.post<ModelRoute>('/v0/admin/model-routes', data),
  deleteModelRoute: (modelId: string) => apiClient.delete<void>(`/v0/admin/model-routes/${modelId}`),

  // Providers
  getProviders: () => apiClient.get<{ providers: Provider[] }>('/v0/admin/providers'),
  checkProviderHealth: (providerName: string) =>
    apiClient.post<ProviderHealth>(`/v0/admin/providers/${providerName}/health`),

  // Logs/Usage
  getLogs: (params?: {
    start_time?: string;
    end_time?: string;
    api_key_name?: string;
    provider?: string;
    status?: string;
    cursor?: string;
    limit?: number;
  }) => apiClient.get<PaginatedResponse<UsageRecord>>('/v0/admin/logs', params ? { params } : undefined),

  // Maintenance
  getMaintenanceMode: () => apiClient.get<any>('/v0/admin/maintenance'),
  setMaintenanceMode: (data: any) => apiClient.put<any>('/v0/admin/maintenance', data),
};

// Import types for API functions
import {
  HealthResponse,
  DetailedHealthResponse,
  GatewayStatus,
  GatewayConfig,
  ConfigUpdateRequest,
  ModelRoute,
  Provider,
  ProviderHealth,
  UsageRecord,
  PaginatedResponse,
} from '../types';
