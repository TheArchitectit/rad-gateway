'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { AppLayout } from '@/components/templates/AppLayout';
import { DataTable } from '@/components/organisms/DataTable';
import { Button } from '@/components/atoms/Button';
import { StatusBadge } from '@/components/molecules/StatusBadge';
import { Plus } from 'lucide-react';

interface Provider {
  id: string;
  name: string;
  type: string;
  status: 'healthy' | 'degraded' | 'unhealthy';
  circuitState: 'closed' | 'open' | 'half-open';
  latency: string;
  cost: string;
}

const mockProviders: Provider[] = [
  { id: '1', name: 'OpenAI', type: 'openai', status: 'healthy', circuitState: 'closed', latency: '120ms', cost: '$0.002/token' },
  { id: '2', name: 'Anthropic', type: 'anthropic', status: 'healthy', circuitState: 'closed', latency: '150ms', cost: '$0.003/token' },
  { id: '3', name: 'Gemini', type: 'gemini', status: 'degraded', circuitState: 'closed', latency: '200ms', cost: '$0.001/token' },
];

export default function ProvidersPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');

  const filteredProviders = mockProviders.filter(
    (p) => p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
           p.type.toLowerCase().includes(searchQuery.toLowerCase())
  );

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
          searchable
          searchValue={searchQuery}
          onSearch={setSearchQuery}
          columns={[
            { key: 'name', header: 'Name' },
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
            { key: 'cost', header: 'Cost' },
          ]}
          emptyState={{
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
