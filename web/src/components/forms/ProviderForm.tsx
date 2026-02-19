'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '../atoms/Button';
import { FormField } from './FormField';
import { SelectField } from './SelectField';
import { Loader2, TestTube, Eye, EyeOff } from 'lucide-react';

const providerSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name too long'),
  slug: z.string().min(1, 'Slug is required').regex(/^[a-z0-9-]+$/, 'Slug must be lowercase letters, numbers, and hyphens'),
  providerType: z.enum(['openai', 'anthropic', 'gemini'], { required_error: 'Provider type is required' }),
  baseUrl: z.string().url('Must be a valid URL').optional().or(z.literal('')),
  apiKey: z.string().min(1, 'API key is required'),
  priority: z.number().int().min(0).max(100).default(0),
  weight: z.number().int().min(1).max(100).default(1),
});

export type ProviderFormData = z.infer<typeof providerSchema>;

interface ProviderFormProps {
  defaultValues?: Partial<ProviderFormData>;
  onSubmit: (data: ProviderFormData) => void;
  onCancel: () => void;
  onTestConnection?: (data: ProviderFormData) => Promise<{ success: boolean; message: string; latency?: number }>;
  isLoading?: boolean;
  submitLabel?: string;
}

export function ProviderForm({
  defaultValues,
  onSubmit,
  onCancel,
  onTestConnection,
  isLoading = false,
  submitLabel = 'Create Provider',
}: ProviderFormProps) {
  const [showApiKey, setShowApiKey] = useState(false);
  const [isTesting, setIsTesting] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string; latency?: number } | null>(null);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isDirty },
  } = useForm<ProviderFormData>({
    resolver: zodResolver(providerSchema),
    defaultValues: {
      name: '',
      slug: '',
      providerType: 'openai',
      baseUrl: '',
      apiKey: '',
      priority: 0,
      weight: 1,
      ...defaultValues,
    },
  });

  const providerType = watch('providerType');

  const handleTestConnection = async () => {
    if (!onTestConnection) return;
    
    const data = watch();
    setIsTesting(true);
    setTestResult(null);
    
    try {
      const result = await onTestConnection(data);
      setTestResult(result);
    } catch (error) {
      setTestResult({
        success: false,
        message: error instanceof Error ? error.message : 'Connection test failed',
      });
    } finally {
      setIsTesting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <FormField
          label="Name"
          error={errors.name?.message}
          hint="Display name for this provider"
        >
          <input
            type="text"
            {...register('name')}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            placeholder="e.g., OpenAI Production"
          />
        </FormField>

        <FormField
          label="Slug"
          error={errors.slug?.message}
          hint="Unique identifier (auto-generated from name)"
        >
          <input
            type="text"
            {...register('slug')}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            placeholder="e.g., openai-production"
          />
        </FormField>
      </div>

      <SelectField
        label="Provider Type"
        error={errors.providerType?.message}
        hint="The AI provider service"
        {...register('providerType')}
      >
        <option value="openai">OpenAI</option>
        <option value="anthropic">Anthropic</option>
        <option value="gemini">Google Gemini</option>
      </SelectField>

      <FormField
        label="API Key"
        error={errors.apiKey?.message}
        hint={`Your ${providerType} API key`}
      >
        <div className="relative">
          <input
            type={showApiKey ? 'text' : 'password'}
            {...register('apiKey')}
            className="w-full px-3 py-2 pr-10 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            placeholder="sk-..."
          />
          <button
            type="button"
            onClick={() => setShowApiKey(!showApiKey)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
          >
            {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
          </button>
        </div>
      </FormField>

      <FormField
        label="Base URL (Optional)"
        error={errors.baseUrl?.message}
        hint={`Default: ${getDefaultBaseUrl(providerType)}`}
      >
        <input
          type="url"
          {...register('baseUrl')}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          placeholder={getDefaultBaseUrl(providerType)}
        />
      </FormField>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <FormField
          label="Priority"
          error={errors.priority?.message}
          hint="Higher priority providers are preferred (0-100)"
        >
          <input
            type="number"
            {...register('priority', { valueAsNumber: true })}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            min={0}
            max={100}
          />
        </FormField>

        <FormField
          label="Weight"
          error={errors.weight?.message}
          hint="Load balancing weight (1-100)"
        >
          <input
            type="number"
            {...register('weight', { valueAsNumber: true })}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
            min={1}
            max={100}
          />
        </FormField>
      </div>

      {testResult && (
        <div
          className={`p-4 rounded-lg ${
            testResult.success
              ? 'bg-green-50 text-green-800 border border-green-200'
              : 'bg-red-50 text-red-800 border border-red-200'
          }`}
        >
          <p className="font-medium">{testResult.success ? 'Connection successful' : 'Connection failed'}</p>
          <p className="text-sm mt-1">{testResult.message}</p>
          {testResult.latency && (
            <p className="text-sm mt-1">Latency: {testResult.latency}ms</p>
          )}
        </div>
      )}

      <div className="flex justify-between items-center pt-4 border-t border-gray-200">
        {onTestConnection && (
          <Button
            type="button"
            variant="secondary"
            onClick={handleTestConnection}
            disabled={isTesting || !isDirty}
          >
            {isTesting ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                Testing...
              </>
            ) : (
              <>
                <TestTube className="w-4 h-4 mr-2" />
                Test Connection
              </>
            )}
          </Button>
        )}

        <div className="flex space-x-3">
          <Button type="button" variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" disabled={isLoading}>
            {isLoading ? (
              <>
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                {submitLabel}...
              </>
            ) : (
              submitLabel
            )}
          </Button>
        </div>
      </div>
    </form>
  );
}

function getDefaultBaseUrl(providerType: string): string {
  switch (providerType) {
    case 'openai':
      return 'https://api.openai.com/v1';
    case 'anthropic':
      return 'https://api.anthropic.com';
    case 'gemini':
      return 'https://generativelanguage.googleapis.com';
    default:
      return '';
  }
}
