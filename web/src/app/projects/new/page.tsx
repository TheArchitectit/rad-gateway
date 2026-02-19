'use client';

import { useRouter } from 'next/navigation';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { ProjectForm, ProjectFormData } from '@/components/forms/ProjectForm';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { showNotification } from '@/stores/uiStore';
import { APIError } from '@/api/client';

export default function NewProjectPage() {
  const router = useRouter();
  const createWorkspace = useWorkspaceStore((state) => state.createWorkspace);
  const isLoading = useWorkspaceStore((state) => state.isLoading);

  const handleSubmit = async (data: ProjectFormData) => {
    try {
      await createWorkspace({
        name: data.name,
        slug: data.slug,
        ...(data.description && { description: data.description }),
        ...(data.logo && { logo: data.logo }),
        settings: {
          theme: data.theme,
          timezone: data.timezone,
          currency: data.currency,
          dateFormat: data.dateFormat,
        },
      });
      showNotification.success('Project created', `Successfully created project "${data.name}"`);
      router.push('/projects');
    } catch (error) {
      const message = error instanceof APIError 
        ? error.message 
        : 'Failed to create project';
      showNotification.error('Creation failed', message);
    }
  };

  const handleCancel = () => {
    router.push('/projects');
  };

  return (
    <AppLayout>
      <div className="max-w-4xl mx-auto">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-gray-900">Create Project</h1>
          <p className="text-gray-500 mt-1">Create a new workspace for your resources</p>
        </div>

        <Card>
          <ProjectForm
            onSubmit={handleSubmit}
            onCancel={handleCancel}
            isLoading={isLoading}
            submitLabel="Create Project"
          />
        </Card>
      </div>
    </AppLayout>
  );
}
