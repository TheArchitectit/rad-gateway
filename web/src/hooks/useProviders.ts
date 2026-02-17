/**
 * RAD Gateway Admin UI - Providers Hooks
 * State Management Engineer - Phase 2 Implementation
 *
 * Custom hooks for provider state and operations.
 */

import { useCallback, useEffect, useState } from 'react';
import { adminAPI } from '../api/client';
import { Provider, ProviderHealth } from '../types';

interface UseProvidersReturn {
  providers: Provider[];
  isLoading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

/**
 * Hook to fetch and manage providers list.
 */
export function useProviders(): UseProvidersReturn {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchProviders = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      const response = await adminAPI.getProviders();
      setProviders(response.providers);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch providers';
      setError(message);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchProviders();
  }, [fetchProviders]);

  return {
    providers,
    isLoading,
    error,
    refresh: fetchProviders,
  };
}

interface UseProviderReturn {
  provider: Provider | null;
  isLoading: boolean;
  error: string | null;
  checkHealth: () => Promise<ProviderHealth | null>;
}

/**
 * Hook to get a specific provider by name.
 */
export function useProvider(name: string): UseProviderReturn {
  const [provider, setProvider] = useState<Provider | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!name) return;

    const fetchProvider = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const response = await adminAPI.getProviders();
        const found = response.providers.find((p) => p.name === name);
        setProvider(found || null);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to fetch provider';
        setError(message);
      } finally {
        setIsLoading(false);
      }
    };

    fetchProvider();
  }, [name]);

  const checkHealth = useCallback(async () => {
    if (!name) return null;

    try {
      const result = await adminAPI.checkProviderHealth(name);
      return result;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Health check failed');
      return null;
    }
  }, [name]);

  return {
    provider,
    isLoading,
    error,
    checkHealth,
  };
}

interface UseProviderStatsReturn {
  healthyCount: number;
  degradedCount: number;
  unhealthyCount: number;
  disabledCount: number;
  totalCount: number;
  openCircuitCount: number;
}

/**
 * Hook to get provider statistics.
 */
export function useProviderStats(): UseProviderStatsReturn {
  const { providers } = useProviders();

  const stats = providers.reduce(
    (acc, provider) => {
      acc.totalCount++;

      switch (provider.status) {
        case 'healthy':
          acc.healthyCount++;
          break;
        case 'degraded':
          acc.degradedCount++;
          break;
        case 'unhealthy':
          acc.unhealthyCount++;
          break;
        case 'disabled':
          acc.disabledCount++;
          break;
      }

      if (provider.circuitBreaker === 'open') {
        acc.openCircuitCount++;
      }

      return acc;
    },
    {
      healthyCount: 0,
      degradedCount: 0,
      unhealthyCount: 0,
      disabledCount: 0,
      totalCount: 0,
      openCircuitCount: 0,
    }
  );

  return stats;
}

/**
 * Hook to get providers by status.
 */
export function useProvidersByStatus(
  status: Provider['status']
): Provider[] {
  const { providers } = useProviders();
  return providers.filter((p) => p.status === status);
}
