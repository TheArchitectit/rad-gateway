/**
 * RAD Gateway Admin UI - API Keys Hooks
 * State Management Engineer - Phase 2 Implementation
 *
 * Custom hooks for API key state and operations.
 */

import { useCallback, useEffect, useState } from 'react';
import { APIKey, CreateAPIKeyDTO } from '../types';

// Mock data for Phase 2
const mockApiKeys: APIKey[] = [
  {
    id: 'key-1',
    name: 'Production Key',
    keyPreview: 'rad_****_prod',
    permissions: ['read', 'write'],
    rateLimit: 1000,
    currentUsage: 0,
    createdAt: new Date().toISOString(),
    isActive: true,
    workspaceId: 'ws-1',
  },
  {
    id: 'key-2',
    name: 'Development Key',
    keyPreview: 'rad_****_dev',
    permissions: ['read', 'write', 'delete'],
    rateLimit: 100,
    currentUsage: 45,
    createdAt: new Date().toISOString(),
    isActive: true,
    workspaceId: 'ws-2',
  },
  {
    id: 'key-3',
    name: 'Read-only Key',
    keyPreview: 'rad_****_readonly',
    permissions: ['read'],
    rateLimit: 500,
    usageLimit: 10000,
    currentUsage: 2341,
    createdAt: new Date().toISOString(),
    isActive: true,
    workspaceId: 'ws-1',
  },
];

interface UseApiKeysReturn {
  apiKeys: APIKey[];
  isLoading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  create: (data: CreateAPIKeyDTO) => Promise<APIKey>;
  update: (id: string, updates: Partial<APIKey>) => Promise<void>;
  delete: (id: string) => Promise<void>;
  toggleActive: (id: string) => Promise<void>;
}

/**
 * Hook to fetch and manage API keys.
 */
export function useApiKeys(): UseApiKeysReturn {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchApiKeys = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    try {
      // TODO: Replace with actual API call
      await new Promise((resolve) => setTimeout(resolve, 300));
      setApiKeys(mockApiKeys);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch API keys';
      setError(message);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchApiKeys();
  }, [fetchApiKeys]);

  const create = useCallback(
    async (data: CreateAPIKeyDTO): Promise<APIKey> => {
      // TODO: Replace with actual API call
      await new Promise((resolve) => setTimeout(resolve, 500));

      const newKey: APIKey = {
        id: `key-${Date.now()}`,
        name: data.name,
        keyPreview: `rad_****_${Math.random().toString(36).substr(2, 4)}`,
        permissions: data.permissions,
        rateLimit: data.rateLimit || 100,
        ...(data.usageLimit !== undefined && { usageLimit: data.usageLimit }),
        currentUsage: 0,
        createdAt: new Date().toISOString(),
        isActive: true,
        workspaceId: 'current-ws',
      };

      setApiKeys((prev) => [...prev, newKey]);
      return newKey;
    },
    []
  );

  const update = useCallback(async (id: string, updates: Partial<APIKey>) => {
    // TODO: Replace with actual API call
    await new Promise((resolve) => setTimeout(resolve, 300));

    setApiKeys((prev) =>
      prev.map((key) => (key.id === id ? { ...key, ...updates } : key))
    );
  }, []);

  const deleteKey = useCallback(async (id: string) => {
    // TODO: Replace with actual API call
    await new Promise((resolve) => setTimeout(resolve, 300));

    setApiKeys((prev) => prev.filter((key) => key.id !== id));
  }, []);

  const toggleActive = useCallback(
    async (id: string) => {
      const key = apiKeys.find((k) => k.id === id);
      if (key) {
        await update(id, { isActive: !key.isActive });
      }
    },
    [apiKeys, update]
  );

  return {
    apiKeys,
    isLoading,
    error,
    refresh: fetchApiKeys,
    create,
    update,
    delete: deleteKey,
    toggleActive,
  };
}

interface UseApiKeyReturn {
  apiKey: APIKey | null;
  isLoading: boolean;
  error: string | null;
  update: (updates: Partial<APIKey>) => Promise<void>;
  delete: () => Promise<void>;
  regenerate: () => Promise<string>;
}

/**
 * Hook to get a specific API key by ID.
 */
export function useApiKey(id: string): UseApiKeyReturn {
  const { apiKeys, isLoading, error, update, delete: deleteKey } = useApiKeys();
  const [apiKey, setApiKey] = useState<APIKey | null>(null);

  useEffect(() => {
    const found = apiKeys.find((k) => k.id === id);
    setApiKey(found || null);
  }, [apiKeys, id]);

  const updateKey = useCallback(
    async (updates: Partial<APIKey>) => {
      await update(id, updates);
    },
    [id, update]
  );

  const deleteKeyById = useCallback(async () => {
    await deleteKey(id);
  }, [id, deleteKey]);

  const regenerate = useCallback(async () => {
    // TODO: Replace with actual API call
    await new Promise((resolve) => setTimeout(resolve, 500));

    const newKeyValue = `rad_${Math.random().toString(36).substr(2, 20)}`;
    // In real implementation, this would return the new key value once
    return newKeyValue;
  }, []);

  return {
    apiKey,
    isLoading,
    error,
    update: updateKey,
    delete: deleteKeyById,
    regenerate,
  };
}

interface UseApiKeyStatsReturn {
  totalCount: number;
  activeCount: number;
  inactiveCount: number;
  totalUsage: number;
}

/**
 * Hook to get API key statistics.
 */
export function useApiKeyStats(): UseApiKeyStatsReturn {
  const { apiKeys } = useApiKeys();

  return apiKeys.reduce(
    (acc, key) => {
      acc.totalCount++;
      if (key.isActive) {
        acc.activeCount++;
      } else {
        acc.inactiveCount++;
      }
      acc.totalUsage += key.currentUsage;
      return acc;
    },
    {
      totalCount: 0,
      activeCount: 0,
      inactiveCount: 0,
      totalUsage: 0,
    }
  );
}
