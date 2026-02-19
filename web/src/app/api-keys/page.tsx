'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { AppLayout } from '@/components/templates/AppLayout';
import { DataTable } from '@/components/organisms/DataTable';
import { Button } from '@/components/atoms/Button';
import { Badge } from '@/components/atoms/Badge';
import { Plus, Copy } from 'lucide-react';

interface APIKey {
  id: string;
  name: string;
  key: string;
  status: 'active' | 'revoked';
  createdAt: string;
  lastUsed: string;
}

const mockAPIKeys: APIKey[] = [
  { id: '1', name: 'Production API Key', key: 'sk-rad...prod', status: 'active', createdAt: '2026-02-15', lastUsed: '2 hours ago' },
  { id: '2', name: 'Development API Key', key: 'sk-rad...dev', status: 'active', createdAt: '2026-02-14', lastUsed: '5 minutes ago' },
  { id: '3', name: 'Test API Key', key: 'sk-rad...test', status: 'revoked', createdAt: '2026-02-10', lastUsed: '3 days ago' },
];

export default function APIKeysPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');

  const filteredKeys = mockAPIKeys.filter(
    (k) => k.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleCopy = (key: string) => {
    navigator.clipboard.writeText(key);
  };

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
          data={filteredKeys}
          keyExtractor={(item) => item.id}
          searchable
          searchValue={searchQuery}
          onSearch={setSearchQuery}
          columns={[
            { key: 'name', header: 'Name' },
            { key: 'key', header: 'Key', render: (item) => (
              <div className="flex items-center gap-2">
                <code className="bg-gray-100 px-2 py-1 rounded text-sm">{item.key}</code>
                <button
                  onClick={() => handleCopy(item.key)}
                  className="p-1 hover:bg-gray-100 rounded"
                >
                  <Copy className="w-4 h-4 text-gray-500" />
                </button>
              </div>
            )},
            { key: 'status', header: 'Status', render: (item) => (
              <Badge color={item.status === 'active' ? 'success' : 'error'}>
                {item.status === 'active' ? 'Active' : 'Revoked'}
              </Badge>
            )},
            { key: 'created', header: 'Created', render: (item) => item.createdAt },
            { key: 'lastUsed', header: 'Last Used' },
          ]}
          emptyState={{
            title: 'No API keys found',
            description: 'Create your first API key to start making requests.',
            action: {
              label: 'Create API Key',
              onClick: () => router.push('/api-keys/new'),
            },
          }}
        />
      </div>
    </AppLayout>
  );
}
