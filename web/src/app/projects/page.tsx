'use client';

import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AlertCircle, Plus, RefreshCw } from 'lucide-react';
import { AppLayout } from '@/components/templates/AppLayout';
import { DataTable } from '@/components/organisms/DataTable';
import { Button } from '@/components/atoms/Button';
import { StatusBadge } from '@/components/molecules/StatusBadge';
import { useProjects } from '@/queries';
import type { Workspace } from '@/types';

interface ProjectRow {
  id: string;
  name: string;
  slug: string;
  description: string;
  status: 'active' | 'inactive';
  createdAt: string;
}

export default function ProjectsPage() {
  const router = useRouter();
  const [searchQuery, setSearchQuery] = useState('');

  const { data, isLoading, isError, error, refetch } = useProjects({ page: 1, pageSize: 200 });

  const rows = useMemo<ProjectRow[]>(() => {
    const projects = data?.data || [];
    return projects.map((p: Workspace) => ({
      id: p.id,
      name: p.name,
      slug: p.slug,
      description: p.description || 'No description',
      status: (p as Workspace & { status?: string }).status === 'active' ? 'active' : 'inactive',
      createdAt: new Date(p.createdAt).toLocaleDateString(),
    }));
  }, [data]);

  const filtered = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter(
      (p) => p.name.toLowerCase().includes(q) || p.slug.toLowerCase().includes(q) || p.description.toLowerCase().includes(q)
    );
  }, [rows, searchQuery]);

  if (isError && !isLoading) {
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

          <div className="bg-red-50 border border-red-200 rounded-lg p-6">
            <div className="flex items-start gap-4">
              <AlertCircle className="w-6 h-6 text-red-500 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <h3 className="text-lg font-medium text-red-800">Failed to load projects</h3>
                <p className="mt-1 text-red-600">{error?.message || 'Could not fetch projects.'}</p>
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
            <h1 className="text-2xl font-bold text-gray-900">Projects</h1>
            <p className="text-gray-500">Manage your workspaces</p>
          </div>
          <Button onClick={() => router.push('/projects/new')}>
            <Plus className="w-4 h-4 mr-2" />
            New Project
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
            {
              key: 'name',
              header: 'Name',
              render: (item) => (
                <div>
                  <div className="font-medium text-gray-900">{item.name}</div>
                  <div className="text-sm text-gray-500">{item.slug}</div>
                </div>
              ),
            },
            { key: 'description', header: 'Description' },
            {
              key: 'status',
              header: 'Status',
              render: (item) => <StatusBadge status={item.status} />,
            },
            { key: 'createdAt', header: 'Created' },
          ]}
          emptyState={{
            title: 'No projects found',
            description: searchQuery
              ? 'No projects match your search. Try a different query.'
              : 'Create your first project to organize your resources.',
            ...(!searchQuery
              ? {
                  action: {
                    label: 'Create Project',
                    onClick: () => router.push('/projects/new'),
                  },
                }
              : {}),
          }}
        />
      </div>
    </AppLayout>
  );
}
