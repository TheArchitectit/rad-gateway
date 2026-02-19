'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '../atoms/Button';
import { FormField } from './FormField';
import { SelectField } from './SelectField';
import { Loader2, Upload } from 'lucide-react';

const projectSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name too long'),
  slug: z.string().min(1, 'Slug is required').regex(/^[a-z0-9-]+$/, 'Slug must be lowercase letters, numbers, and hyphens'),
  description: z.string().max(500, 'Description too long').optional(),
  logo: z.string().optional(),
  theme: z.enum(['light', 'dark', 'system']).default('system'),
  timezone: z.string().default('UTC'),
  currency: z.enum(['USD', 'EUR', 'GBP']).default('USD'),
  dateFormat: z.enum(['YYYY-MM-DD', 'MM/DD/YYYY', 'DD/MM/YYYY']).default('YYYY-MM-DD'),
});

export type ProjectFormData = z.infer<typeof projectSchema>;

interface ProjectFormProps {
  defaultValues?: Partial<ProjectFormData>;
  onSubmit: (data: ProjectFormData) => void;
  onCancel: () => void;
  isLoading?: boolean;
  submitLabel?: string;
}

export function ProjectForm({
  defaultValues,
  onSubmit,
  onCancel,
  isLoading = false,
  submitLabel = 'Create Project',
}: ProjectFormProps) {
  const [logoPreview, setLogoPreview] = useState<string | null>(defaultValues?.logo || null);

  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors },
  } = useForm<ProjectFormData>({
    resolver: zodResolver(projectSchema),
    defaultValues: {
      name: '',
      slug: '',
      description: '',
      theme: 'system',
      timezone: 'UTC',
      currency: 'USD',
      dateFormat: 'YYYY-MM-DD',
      ...defaultValues,
    },
  });

  const handleLogoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onloadend = () => {
        const base64 = reader.result as string;
        setLogoPreview(base64);
        setValue('logo', base64);
      };
      reader.readAsDataURL(file);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      <div className="flex items-start space-x-6">
        <div className="flex-shrink-0">
          <FormField label="Logo" error={undefined} hint="Optional project logo">
            <div className="relative">
              <div className="w-24 h-24 rounded-lg border-2 border-dashed border-gray-300 flex items-center justify-center bg-gray-50 overflow-hidden">
                {logoPreview ? (
                  <img src={logoPreview} alt="Logo preview" className="w-full h-full object-cover" />
                ) : (
                  <Upload className="w-8 h-8 text-gray-400" />
                )}
              </div>
              <input
                type="file"
                accept="image/*"
                onChange={handleLogoChange}
                className="absolute inset-0 opacity-0 cursor-pointer"
              />
            </div>
          </FormField>
        </div>

        <div className="flex-1 space-y-4">
          <FormField
            label="Name"
            error={errors.name?.message}
            hint="Project display name"
            required
          >
            <input
              type="text"
              {...register('name')}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              placeholder="e.g., Production"
            />
          </FormField>

          <FormField
            label="Slug"
            error={errors.slug?.message}
            hint="Unique identifier (auto-generated)"
            required
          >
            <input
              type="text"
              {...register('slug')}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              placeholder="e.g., production"
            />
          </FormField>
        </div>
      </div>

      <FormField
        label="Description"
        error={errors.description?.message}
        hint="Brief description of the project"
      >
        <textarea
          {...register('description')}
          rows={3}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent resize-none"
          placeholder="What is this project for?"
        />
      </FormField>

      <div className="border-t border-gray-200 pt-6">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Settings</h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <SelectField
            label="Theme"
            error={errors.theme?.message}
            hint="UI theme preference"
            {...register('theme')}
          >
            <option value="light">Light</option>
            <option value="dark">Dark</option>
            <option value="system">System</option>
          </SelectField>

          <FormField
            label="Timezone"
            error={errors.timezone?.message}
            hint="Default timezone for this project"
          >
            <input
              type="text"
              {...register('timezone')}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              placeholder="e.g., America/New_York"
            />
          </FormField>

          <SelectField
            label="Currency"
            error={errors.currency?.message}
            hint="Currency for cost tracking"
            {...register('currency')}
          >
            <option value="USD">USD ($)</option>
            <option value="EUR">EUR (€)</option>
            <option value="GBP">GBP (£)</option>
          </SelectField>

          <SelectField
            label="Date Format"
            error={errors.dateFormat?.message}
            hint="Date display format"
            {...register('dateFormat')}
          >
            <option value="YYYY-MM-DD">YYYY-MM-DD</option>
            <option value="MM/DD/YYYY">MM/DD/YYYY</option>
            <option value="DD/MM/YYYY">DD/MM/YYYY</option>
          </SelectField>
        </div>
      </div>

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
