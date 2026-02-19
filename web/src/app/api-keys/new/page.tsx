'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { APIKeyForm, APIKeyFormData, ShowKeyModal } from '@/components/forms/APIKeyForm';
import { useCreateAPIKey } from '@/queries/apikeys';
import { useWorkspaces } from '@/stores/workspaceStore';
import { showNotification } from '@/stores/uiStore';
import { APIError } from '@/api/client';

export default function NewAPIKeyPage() {
  const router = useRouter();
  const [createdKey, setCreatedKey] = useState<{ keySecret: string; name: string } | null>(null);
  const createMutation = useCreateAPIKey();
  const workspaces = useWorkspaces();

  const handleSubmit = async (data: APIKeyFormData) => {
    try {
      const result = await createMutation.mutateAsync(data);
      setCreatedKey({
        keySecret: result.keySecret,
        name: result.name,
      });
    } catch (error) {
      const message = error instanceof APIError 
        ? error.message 
        : 'Failed to create API key';
      showNotification.error('Creation failed', message);
    }
  };

  const handleCancel = () => {
    router.push('/api-keys');
  };

  const handleCloseModal = () => {
    setCreatedKey(null);
    router.push('/api-keys');
  };

  const workspaceOptions = workspaces.map((w) => ({ id: w.id, name: w.name }));

  return (
    <AppLayout>
      <div className="max-w-4xl mx-auto">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-gray-900">Create API Key</h1>
          <p className="text-gray-500 mt-1">Generate a new API key for accessing the gateway</p>
        </div>

        <Card>
          <APIKeyForm
            onSubmit={handleSubmit}
            onCancel={handleCancel}
            isLoading={createMutation.isPending}
            submitLabel="Create API Key"
            workspaces={workspaceOptions}
            availableModels={['gpt-4', 'gpt-4o', 'gpt-4o-mini', 'claude-3-opus', 'claude-3-sonnet', 'gemini-pro']}
          />
        </Card>

        {createdKey && (
          <ShowKeyModal
            keySecret={createdKey.keySecret}
            keyName={createdKey.name}
            onClose={handleCloseModal}
          />
        )}
      </div>
    </AppLayout>
  );
}
