'use client';

import { useRouter } from 'next/navigation';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { ProviderForm, ProviderFormData } from '@/components/forms/ProviderForm';
import { useCreateProvider, useTestProviderConnection } from '@/queries/providers';
import { showNotification } from '@/stores/uiStore';
import { APIError } from '@/api/client';

export default function NewProviderPage() {
  const router = useRouter();
  const createMutation = useCreateProvider();
  const testMutation = useTestProviderConnection();

  const handleSubmit = async (data: ProviderFormData) => {
    try {
      await createMutation.mutateAsync(data);
      showNotification.success('Provider created', `Successfully created provider "${data.name}"`);
      router.push('/providers');
    } catch (error) {
      const message = error instanceof APIError 
        ? error.message 
        : 'Failed to create provider';
      showNotification.error('Creation failed', message);
    }
  };

  const handleCancel = () => {
    router.push('/providers');
  };

  const handleTestConnection = async (data: ProviderFormData) => {
    try {
      const result = await testMutation.mutateAsync(data.slug);
      return result;
    } catch (error) {
      throw error;
    }
  };

  return (
    <AppLayout>
      <div className="max-w-4xl mx-auto">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-gray-900">Create Provider</h1>
          <p className="text-gray-500 mt-1">Add a new AI provider to your gateway</p>
        </div>

        <Card>
          <ProviderForm
            onSubmit={handleSubmit}
            onCancel={handleCancel}
            onTestConnection={handleTestConnection}
            isLoading={createMutation.isPending}
            submitLabel="Create Provider"
          />
        </Card>
      </div>
    </AppLayout>
  );
}
