'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AlertCircle, Copy, Plus, RefreshCw } from 'lucide-react';
import { AppLayout } from '@/components/templates/AppLayout';
import { DataTable } from '@/components/organisms/DataTable';
import { Button } from '@/components/atoms/Button';
import { StatusBadge } from '@/components/molecules/StatusBadge';
import { useAPIKeys } from '@/queries';
import type { APIKeyResponse } from '@/queries';

interface APIKeyRow {
  id: string;
  name: string;
  keyPreview: string;
  status: 'active' | 'inactive';
  createdAt: string;
  lastUsed: string;
}

function formatLastUsed(value?: string): string {
  if (!value) return 'Never';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Never';
  const deltaMs = Date.now() - date.getTime();
  const mins = Math.floor(deltaMs / 60000);
  if (mins < 1) return 'Just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

export default function APIKeysPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');

  const { data, isLoading, isError, error, refetch } = useAPIKeys({ page: 1, pageSize: 200 });

  const rows = useMemo<APIKeyRow[]>(() => {
    const keys = data?.data || [];
    return keys.map((k: APIKeyResponse) => ({
      id: k.id,
      name: k.name,
      keyPreview: k.keyPreview,
      status: k.status === 'active' ? 'active' : 'inactive',
      createdAt: new Date(k.createdAt).toLocaleDateString(),
      lastUsed: formatLastUsed(k.lastUsedAt),
    }));
  }, [data]);

  const filtered = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter((k) => k.name.toLowerCase().includes(q) || k.keyPreview.toLowerCase().includes(q));
  }, [rows, searchQuery]);

  if (isError && !isLoading) {
    return (
      <AppLayout>
        <div className="space-y-6">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">API Keys</h1>
              <p className="text-gray-500">Manage API keys for accessing the gateway</p>
            </div>
            <Button onClick={() => router.push('/api-keys/new')}>
              <Plus className="w-4 h-4 mr-2" />
              Create API Key
            </Button>
          </div>

          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <div className="flex items-start gap-4">
              <AlertCircle className="w-6 h-6 text-red-500 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <h3 className="text-lg font-medium text-red-800">Failed to load API keys</h3>
                <p className="mt-1 text-red-600">{error?.message || 'Could not fetch API keys.'}</p>
                <Button variant="secondary" className="mt-4" onClick={() => void refetch()}>
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
            <h1 className="text-2xl font-bold text-gray-900">API Keys</h1>
            <p className="text-gray-500">Manage API keys for accessing the gateway</p>
          </div>
          <Button onClick={() => router.push('/api-keys/new')}>
            <Plus className="w-4 h-4 mr-2" />
            Create API Key
          </Button>
        </div>

        <DataTable
          data={filtered}
          keyExtractor={(item) => item.id}
          loading={isLoading}
          searchable
          searchValue={searchQuery}
          onSearch={setSearchQuery}
          columns={[
            { key: 'name', header: 'Name' },
            {
              key: 'keyPreview',
              header: 'Key',
              render: (item) => (
                <div className="flex items-center gap-2">
                  <code className="bg-gray-100 px-2 py-1 rounded text-sm">{item.keyPreview}</code>
                  <button
                    onClick={() => void navigator.clipboard.writeText(item.keyPreview)}
                    className="p-1 hover:bg-gray-100 rounded"
                  >
                    <Copy className="w-4 h-4 text-gray-500" />
                  </button>
                </div>
              ),
            },
            {
              key: 'status',
              header: 'Status',
              render: (item) => <StatusBadge status={item.status} />,
            },
            { key: 'createdAt', header: 'Created' },
            { key: 'lastUsed', header: 'Last Used' },
          ]}
          emptyState={{
            title: 'No API keys found',
            description: searchQuery
              ? 'No API keys match your search. Try a different query.'
              : 'Create your first API key to start making requests.',
            ...(!searchQuery
              ? {
                  action: {
                    label: 'Create API Key',
                    onClick: () => router.push('/api-keys/new'),
                  },
                }
              : {}),
          }}
        />
      </div>
    </AppLayout>
  );
}
