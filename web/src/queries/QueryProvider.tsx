/**
 * RAD Gateway Admin UI - TanStack Query Provider
 * Data Fetching Developer - Phase 3 Implementation
 *
 * Provides React Query context for the application.
 */

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode, useState } from 'react';

interface QueryProviderProps {
  children: ReactNode;
}

/**
 * Create a QueryProvider with optimized defaults for our use cases.
 * Includes:
 * - Optimistic updates support
 * - Smart refetching strategies
 * - Error retry logic
 */
export function QueryProvider({ children }: QueryProviderProps) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        // Data freshness strategy
        staleTime: 30 * 1000, // 30 seconds
        gcTime: 5 * 60 * 1000, // 5 minutes (formerly cacheTime)

        // Refetching strategy
        refetchOnWindowFocus: true,
        refetchOnReconnect: true,
        refetchOnMount: true,

        // Error handling
        retry: 3,
        retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),

        // Pagination - use placeholderData for smooth infinite scroll
        placeholderData: (previousData: unknown) => previousData,
      },
      mutations: {
        // Optimistic updates will be handled per-mutation
        retry: 1,
      },
    },
  }));

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

export { QueryClient };
