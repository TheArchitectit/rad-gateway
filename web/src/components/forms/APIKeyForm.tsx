'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '../atoms/Button';
import { FormField } from './FormField';
import { SelectField } from './SelectField';
import { Loader2, Eye, EyeOff, Copy, Check } from 'lucide-react';

const apiKeySchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name too long'),
  workspaceId: z.string().min(1, 'Workspace is required'),
  expiresAt: z.string().optional(),
  rateLimit: z.number().int().min(1).max(10000).optional(),
  allowedModels: z.array(z.string()).optional(),
  allowedAPIs: z.array(z.string()).optional(),
  metadata: z.record(z.unknown()).optional(),
});

export type APIKeyFormData = z.infer<typeof apiKeySchema>;

interface APIKeyFormProps {
  defaultValues?: Partial<APIKeyFormData>;
  onSubmit: (data: APIKeyFormData) => void;
  onCancel: () => void;
  isLoading?: boolean;
  submitLabel?: string;
  workspaces?: { id: string; name: string }[];
  availableModels?: string[];
}

export function APIKeyForm({
  defaultValues,
  onSubmit,
  onCancel,
  isLoading = false,
  submitLabel = 'Create API Key',
  workspaces = [],
  availableModels = [],
}: APIKeyFormProps) {
  const [showAdvanced, setShowAdvanced] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<APIKeyFormData>({
    resolver: zodResolver(apiKeySchema),
    defaultValues: {
      name: '',
      workspaceId: '',
      allowedAPIs: [],
      allowedModels: [],
      ...defaultValues,
    },
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <FormField
        label="Name"
        error={errors.name?.message}
        hint="A descriptive name for this API key"
        required
      >
        <input
          type="text"
          {...register('name')}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          placeholder="e.g., Production App Key"
        />
      </FormField>

      <SelectField
        label="Workspace"
        error={errors.workspaceId?.message}
        hint="The workspace this key belongs to"
        required
        {...register('workspaceId')}
      >
        <option value="">Select a workspace</option>
        {workspaces.map((ws) => (
          <option key={ws.id} value={ws.id}>
            {ws.name}
          </option>
        ))}
      </SelectField>

      <FormField
        label="Expiration (Optional)"
        error={errors.expiresAt?.message}
        hint="When this key will automatically expire"
      >
        <input
          type="datetime-local"
          {...register('expiresAt')}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
        />
      </FormField>

      <div className="border-t border-gray-200 pt-6">
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="flex items-center text-sm text-indigo-600 hover:text-indigo-800"
        >
          {showAdvanced ? 'Hide' : 'Show'} Advanced Options
        </button>
      </div>

      {showAdvanced && (
        <div className="space-y-6 animate-in fade-in slide-in-from-top-2 duration-200">
          <FormField
            label="Rate Limit (requests per minute)"
            error={errors.rateLimit?.message}
            hint="Maximum requests per minute (1-10000)"
          >
            <input
              type="number"
              {...register('rateLimit', { valueAsNumber: true })}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              min={1}
              max={10000}
              placeholder="60"
            />
          </FormField>

          <FormField
            label="Allowed Models"
            error={errors.allowedModels?.message}
            hint="Restrict this key to specific models (leave empty for all)"
          >
            <div className="space-y-2">
              {availableModels.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {availableModels.map((model) => (
                    <label key={model} className="flex items-center space-x-2 bg-gray-50 px-3 py-2 rounded">
                      <input
                        type="checkbox"
                        value={model}
                        {...register('allowedModels')}
                        className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                      />
                      <span className="text-sm">{model}</span>
                    </label>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-gray-500">No models available</p>
              )}
            </div>
          </FormField>

          <FormField
            label="Allowed APIs"
            error={errors.allowedAPIs?.message}
            hint="Restrict which API endpoints this key can access"
          >
            <div className="flex flex-wrap gap-2">
              {['chat', 'completions', 'embeddings', 'images', 'audio'].map((api) => (
                <label key={api} className="flex items-center space-x-2 bg-gray-50 px-3 py-2 rounded">
                  <input
                    type="checkbox"
                    value={api}
                    {...register('allowedAPIs')}
                    className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  <span className="text-sm capitalize">{api}</span>
                </label>
              ))}
            </div>
          </FormField>
        </div>
      )}

      <div className="flex justify-end space-x-3 pt-6 border-t border-gray-200">
        <Button type="button" variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Creating...
            </>
          ) : (
            submitLabel
          )}
        </Button>
      </div>
    </form>
  );
}

interface ShowKeyModalProps {
  keySecret: string;
  keyName: string;
  onClose: () => void;
}

export function ShowKeyModal({ keySecret, keyName, onClose }: ShowKeyModalProps) {
  const [showKey, setShowKey] = useState(false);
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(keySecret);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
        <div className="mb-4">
          <h2 className="text-xl font-bold text-gray-900">API Key Created</h2>
          <p className="text-gray-500 mt-1">
            Your API key for "{keyName}" has been created successfully.
          </p>
        </div>

        <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 mb-6">
          <p className="text-amber-800 text-sm font-medium">
            This is the only time you will see the full key. Please copy it now.
          </p>
        </div>

        <div className="bg-gray-100 rounded-lg p-4 mb-6">
          <div className="flex items-center justify-between">
            <code className="text-sm font-mono text-gray-800 truncate flex-1 mr-4">
              {showKey ? keySecret : keySecret.slice(0, 10) + '...'}
            </code>
            <div className="flex items-center space-x-2">
              <button
                type="button"
                onClick={() => setShowKey(!showKey)}
                className="p-2 hover:bg-gray-200 rounded"
              >
                {showKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
              <button
                type="button"
                onClick={handleCopy}
                className="p-2 hover:bg-gray-200 rounded"
              >
                {copied ? <Check className="w-4 h-4 text-green-600" /> : <Copy className="w-4 h-4" />}
              </button>
            </div>
          </div>
        </div>

        <div className="flex justify-end">
          <Button onClick={onClose}>Done</Button>
        </div>
      </div>
    </div>
  );
}
