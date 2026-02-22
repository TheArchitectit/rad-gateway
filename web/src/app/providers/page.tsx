'use client';

import { useState, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { AppLayout } from '@/components/templates/AppLayout';
import { DataTable } from '@/components/organisms/DataTable';
import { Button } from '@/components/atoms/Button';
import { StatusBadge } from '@/components/molecules/StatusBadge';
import { Plus, AlertCircle, RefreshCw } from 'lucide-react';
import { useProviders, useRefreshProviders } from '@/queries';
import type { Provider } from '@/types';

interface ProviderRow {
  id: string;
  name: string;
  displayName: string;
  type: string;
  status: 'healthy' | 'degraded' | 'unhealthy' | 'disabled';
  circuitState: 'closed' | 'open' | 'half-open';
  latency: string;
  requestCount: number;
}

export default function ProvidersPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');

  const { data, isLoading, error, isError } = useProviders();
  const refreshProviders = useRefreshProviders();

  const providers: ProviderRow[] = useMemo(() => {
    if (!data?.providers) return [];

    return data.providers.map((p: Provider) => ({
      id: p.id,
      name: p.name,
      displayName: p.displayName || p.name,
      type: p.name.toLowerCase().includes('openai') ? 'openai' :
            p.name.toLowerCase().includes('anthropic') ? 'anthropic' :
            p.name.toLowerCase().includes('gemini') ? 'gemini' : 'other',
      status: p.status,
      circuitState: p.circuitBreaker,
      latency: p.latencyMs ? `${p.latencyMs}ms` : 'N/A',
      requestCount: p.requestCount24h,
    }));
  }, [data]);

  const filteredProviders = useMemo(() => {
    if (!searchQuery) return providers;
    const query = searchQuery.toLowerCase();
    return providers.filter(
      (p) => p.name.toLowerCase().includes(query) ||
             p.type.toLowerCase().includes(query) ||
             p.displayName.toLowerCase().includes(query)
    );
  }, [providers, searchQuery]);

  if (isError && !isLoading) {
    return (
      <AppLayout>
        <div className="space-y-6">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Providers</h1>
              <p className="text-gray-500">Manage your AI providers</p>
            </div>
            <Button onClick={() => router.push('/providers/new')}>
              <Plus className="w-4 h-4 mr-2" />
              Add Provider
            </Button>
          </div>

          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <div className="flex items-start gap-4">
              <AlertCircle className="w-6 h-6 text-red-500 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <h3 className="text-lg font-medium text-red-800">Failed to load providers</h3>
                <p className="mt-1 text-red-600">
                  {error?.message || 'An error occurred while fetching providers. Please try again.'}
                </p>
                <Button
                  variant="secondary"
                  className="mt-4"
                  onClick={() => refreshProviders()}
                >
                  <RefreshCw className="w-4 h-4 mr-2" />
                  Retry
                </Button>
              </div>
            </div>
          </div>
        </div>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Providers</h1>
            <p className="text-gray-500">Manage your AI providers</p>
          </div>
          <Button onClick={() => router.push('/providers/new')}>
            <Plus className="w-4 h-4 mr-2" />
            Add Provider
          </Button>
        </div>

        <DataTable
          data={filteredProviders}
          keyExtractor={(item) => item.id}
          loading={isLoading}
          searchable
          searchValue={searchQuery}
          onSearch={setSearchQuery}
          columns={[
            { key: 'name', header: 'Name', render: (item) => (
              <div>
                <div className="font-medium text-gray-900">{item.displayName}</div>
                <div className="text-sm text-gray-500">{item.name}</div>
              </div>
            )},
            { key: 'type', header: 'Type', render: (item) => (
              <span className="capitalize">{item.type}</span>
            )},
            { key: 'status', header: 'Status', render: (item) => (
              <StatusBadge status={item.status} showPulse={item.status === 'healthy'} />
            )},
            { key: 'circuit', header: 'Circuit', render: (item) => (
              <StatusBadge status={item.circuitState} />
            )},
            { key: 'latency', header: 'Latency' },
            { key: 'requests', header: 'Requests (24h)', render: (item) => (
              <span className="text-gray-900">{item.requestCount.toLocaleString()}</span>
            )},
          ]}
          emptyState={searchQuery ? {
            title: 'No providers found',
            description: 'No providers match your search. Try a different query.',
          } : {
            title: 'No providers found',
            description: 'Add your first AI provider to start processing requests.',
            action: {
              label: 'Add Provider',
              onClick: () => router.push('/providers/new'),
            },
          }}
        />
      </div>
    </AppLayout>
  );
}
