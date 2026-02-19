'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { AppLayout } from '@/components/templates/AppLayout';
import { DataTable } from '@/components/organisms/DataTable';
import { Button } from '@/components/atoms/Button';
import { Badge } from '@/components/atoms/Badge';
import { Plus } from 'lucide-react';

interface Project {
  id: string;
  name: string;
  description: string;
  status: 'active' | 'inactive';
  apiKeys: number;
  createdAt: string;
}

const mockProjects: Project[] = [
  { id: '1', name: 'Production', description: 'Live production environment', status: 'active', apiKeys: 3, createdAt: '2026-02-01' },
  { id: '2', name: 'Development', description: 'Development and testing', status: 'active', apiKeys: 2, createdAt: '2026-02-05' },
  { id: '3', name: 'Demo', description: 'Demo environment', status: 'inactive', apiKeys: 1, createdAt: '2026-02-10' },
];

export default function ProjectsPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');

  const filteredProjects = mockProjects.filter(
    (p) => p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
           p.description.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Projects</h1>
            <p className="text-gray-500">Manage your workspaces</p>
          </div>
          <Button onClick={() => router.push('/projects/new')}>
            <Plus className="w-4 h-4 mr-2" />
            New Project
          </Button>
        </div>

        <DataTable
          data={filteredProjects}
          keyExtractor={(item) => item.id}
          searchable
          searchValue={searchQuery}
          onSearch={setSearchQuery}
          columns={[
            { key: 'name', header: 'Name' },
            { key: 'description', header: 'Description' },
            { key: 'status', header: 'Status', render: (item) => (
              <Badge color={item.status === 'active' ? 'success' : 'default'}>
                {item.status}
              </Badge>
            )},
            { key: 'apiKeys', header: 'API Keys', render: (item) => (
              <span className="text-gray-900">{item.apiKeys}</span>
            )},
            { key: 'created', header: 'Created', render: (item) => item.createdAt },
          ]}
          emptyState={{
            title: 'No projects found',
            description: 'Create your first project to organize your resources.',
            action: {
              label: 'Create Project',
              onClick: () => router.push('/projects/new'),
            },
          }}
        />
      </div>
    </AppLayout>
  );
}
