'use client';

import { useState } from 'react';
import {
  Check,
  ChevronRight,
  ChevronLeft,
  Key,
  Shield,
  Eye,
  AlertCircle,
  Loader2,
} from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';

import type { CreateAPIKeyRequest } from '@/queries/apikeys';

interface CreateKeyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (data: CreateAPIKeyRequest) => Promise<{ keySecret: string }>;
  workspaceId: string;
  isCreating: boolean;
}

type Step = 'details' | 'permissions' | 'review';

const steps: { id: Step; label: string; description: string }[] = [
  {
    id: 'details',
    label: 'Details',
    description: 'Name and describe your API key',
  },
  {
    id: 'permissions',
    label: 'Permissions',
    description: 'Set access permissions',
  },
  {
    id: 'review',
    label: 'Review',
    description: 'Review and create',
  },
];

const permissionOptions = [
  { id: 'read', label: 'Read', description: 'Access read-only endpoints' },
  { id: 'write', label: 'Write', description: 'Create and modify resources' },
  { id: 'admin', label: 'Admin', description: 'Full administrative access' },
];

const apiScopeOptions = [
  { id: 'chat', label: 'Chat Completions', category: 'Core' },
  { id: 'embeddings', label: 'Embeddings', category: 'Core' },
  { id: 'images', label: 'Image Generation', category: 'Core' },
  { id: 'audio', label: 'Audio', category: 'Core' },
  { id: 'admin', label: 'Admin API', category: 'Management' },
];

export function CreateKeyDialog({
  open,
  onOpenChange,
  onCreate,
  workspaceId,
  isCreating,
}: CreateKeyDialogProps) {
  const [currentStep, setCurrentStep] = useState<Step>('details');
  const [formData, setFormData] = useState<{
    name: string;
    description: string;
    permissions: string[];
    allowedAPIs: string[];
    expiresIn: string;
  }>({
    name: '',
    description: '',
    permissions: ['read'],
    allowedAPIs: ['chat'],
    expiresIn: 'never',
  });
  const [error, setError] = useState<string | null>(null);

  const currentStepIndex = steps.findIndex((s) => s.id === currentStep);

  const handleNext = () => {
    setError(null);
    if (currentStep === 'details') {
      if (!formData.name.trim()) {
        setError('Please enter a name for the API key');
        return;
      }
      setCurrentStep('permissions');
    } else if (currentStep === 'permissions') {
      setCurrentStep('review');
    }
  };

  const handleBack = () => {
    setError(null);
    if (currentStep === 'permissions') {
      setCurrentStep('details');
    } else if (currentStep === 'review') {
      setCurrentStep('permissions');
    }
  };

  const handleCreate = async () => {
    setError(null);
    try {
      const expiresAt =
        formData.expiresIn === 'never'
          ? undefined
          : new Date(
              Date.now() + parseInt(formData.expiresIn) * 24 * 60 * 60 * 1000
            ).toISOString();

      await onCreate({
        name: formData.name,
        workspaceId,
        ...(formData.description && { metadata: { description: formData.description } }),
        ...(expiresAt && { expiresAt }),
        allowedAPIs: formData.allowedAPIs,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create API key');
    }
  };

  const togglePermission = (permission: string) => {
    setFormData((prev) => {
      const permissions = prev.permissions.includes(permission)
        ? prev.permissions.filter((p) => p !== permission)
        : [...prev.permissions, permission];
      return { ...prev, permissions };
    });
  };

  const toggleAPI = (api: string) => {
    setFormData((prev) => {
      const allowedAPIs = prev.allowedAPIs.includes(api)
        ? prev.allowedAPIs.filter((a) => a !== api)
        : [...prev.allowedAPIs, api];
      return { ...prev, allowedAPIs };
    });
  };

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      permissions: ['read'],
      allowedAPIs: ['chat'],
      expiresIn: 'never',
    });
    setCurrentStep('details');
    setError(null);
  };

  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      resetForm();
    }
    onOpenChange(newOpen);
  };

  // Step Indicator Component
  const StepIndicator = () => (
    <div className="mb-8">
      <div className="flex items-center justify-between">
        {steps.map((step, index) => {
          const isActive = step.id === currentStep;
          const isCompleted = index < currentStepIndex;


          return (
            <div key={step.id} className="flex flex-1 items-center">
              <div className="flex flex-col items-center">
                <div
                  className={`flex h-10 w-10 items-center justify-center rounded-full border-2 transition-all duration-300 ${
                    isActive
                      ? 'border-[hsl(var(--primary))] bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))]'
                      : isCompleted
                      ? 'border-[hsl(var(--primary))] bg-[hsl(var(--primary))] text-[hsl(var(--primary-foreground))]'
                      : 'border-[hsl(var(--muted))] bg-transparent text-[hsl(var(--muted-foreground))]'
                  }`}
                >
                  {isCompleted ? (
                    <Check className="h-5 w-5" />
                  ) : (
                    <span className="text-sm font-semibold">{index + 1}</span>
                  )}
                </div>
                <div className="mt-2 text-center">
                  <p
                    className={`text-xs font-medium ${
                      isActive || isCompleted
                        ? 'text-[hsl(var(--foreground))]'
                        : 'text-[hsl(var(--muted-foreground))]'
                    }`}
                  >
                    {step.label}
                  </p>
                  <p className="text-[10px] text-[hsl(var(--muted-foreground))] hidden sm:block">
                    {step.description}
                  </p>
                </div>
              </div>
              {index < steps.length - 1 && (
                <div
                  className={`h-1 flex-1 mx-4 transition-all duration-300 ${
                    isCompleted
                      ? 'bg-[hsl(var(--primary))]'
                      : 'bg-[hsl(var(--muted))]'
                  }`}
                />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );

  // Step 1: Details
  const DetailsStep = () => (
    <div className="space-y-6">
      <div className="space-y-2">
        <Label htmlFor="key-name">
          API Key Name <span className="text-[hsl(var(--destructive))]">*</span>
        </Label>
        <Input
          id="key-name"
          placeholder="e.g., Production API Key"
          value={formData.name}
          onChange={(e) =>
            setFormData((prev) => ({ ...prev, name: e.target.value }))
          }
          className="ui-input"
        />
        <p className="text-xs text-[hsl(var(--muted-foreground))]">
          Give your API key a descriptive name to identify its purpose.
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="key-description">Description (Optional)</Label>
        <Textarea
          id="key-description"
          placeholder="e.g., Used for production environment integrations"
          value={formData.description}
          onChange={(e) =>
            setFormData((prev) => ({ ...prev, description: e.target.value }))
          }
          rows={3}
          className="ui-input resize-none"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="key-expiry">Expiration</Label>
        <Select
          value={formData.expiresIn}
          onValueChange={(value) =>
            setFormData((prev) => ({ ...prev, expiresIn: value }))
          }
        >
          <SelectTrigger className="ui-input">
            <SelectValue placeholder="Select expiration" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="never">Never expires</SelectItem>
            <SelectItem value="30">30 days</SelectItem>
            <SelectItem value="90">90 days</SelectItem>
            <SelectItem value="180">180 days</SelectItem>
            <SelectItem value="365">1 year</SelectItem>
          </SelectContent>
        </Select>
        <p className="text-xs text-[hsl(var(--muted-foreground))]">
          Expired keys will be automatically revoked.
        </p>
      </div>
    </div>
  );

  // Step 2: Permissions
  const PermissionsStep = () => (
    <div className="space-y-6">
      <div className="space-y-4">
        <div>
          <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
            <Shield className="h-4 w-4" />
            Permission Level
          </h3>
          <div className="space-y-3">
            {permissionOptions.map((permission) => (
              <div
                key={permission.id}
                className="flex items-start space-x-3 rounded-lg border border-[hsl(var(--border))] p-3 hover:bg-[hsl(var(--accent))] transition-colors cursor-pointer"
                onClick={() => togglePermission(permission.id)}
              >
                <Checkbox
                  checked={formData.permissions.includes(permission.id)}
                  onCheckedChange={() => togglePermission(permission.id)}
                  id={`perm-${permission.id}`}
                />
                <div className="flex-1">
                  <Label
                    htmlFor={`perm-${permission.id}`}
                    className="font-medium cursor-pointer"
                  >
                    {permission.label}
                  </Label>
                  <p className="text-xs text-[hsl(var(--muted-foreground))]">
                    {permission.description}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <Separator />

        <div>
          <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
            <Eye className="h-4 w-4" />
            Allowed APIs
          </h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            {apiScopeOptions.map((api) => (
              <div
                key={api.id}
                className="flex items-center space-x-2 rounded-md border border-[hsl(var(--border))] p-2 hover:bg-[hsl(var(--accent))] transition-colors cursor-pointer"
                onClick={() => toggleAPI(api.id)}
              >
                <Checkbox
                  checked={formData.allowedAPIs.includes(api.id)}
                  onCheckedChange={() => toggleAPI(api.id)}
                  id={`api-${api.id}`}
                />
                <Label
                  htmlFor={`api-${api.id}`}
                  className="text-sm cursor-pointer flex-1"
                >
                  {api.label}
                </Label>
                <Badge variant="outline" className="text-[10px]">
                  {api.category}
                </Badge>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );

  // Step 3: Review
  const ReviewStep = () => (
    <div className="space-y-6">
      <div className="rounded-lg border border-[hsl(var(--border))] p-4 space-y-4 bg-[hsl(var(--muted))]/10">
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wider text-[hsl(var(--muted-foreground))] mb-1">
            Name
          </h4>
          <p className="text-sm font-medium">{formData.name}</p>
        </div>

        {formData.description && (
          <div>
            <h4 className="text-xs font-semibold uppercase tracking-wider text-[hsl(var(--muted-foreground))] mb-1">
              Description
            </h4>
            <p className="text-sm">{formData.description}</p>
          </div>
        )}

        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wider text-[hsl(var(--muted-foreground))] mb-1">
            Expiration
          </h4>
          <p className="text-sm capitalize">
            {formData.expiresIn === 'never'
              ? 'Never expires'
              : `${formData.expiresIn} days`}
          </p>
        </div>

        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wider text-[hsl(var(--muted-foreground))] mb-1">
            Permissions
          </h4>
          <div className="flex flex-wrap gap-2">
            {formData.permissions.map((perm) => (
              <Badge key={perm} variant="secondary">
                {perm}
              </Badge>
            ))}
          </div>
        </div>

        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wider text-[hsl(var(--muted-foreground))] mb-1">
            Allowed APIs
          </h4>
          <div className="flex flex-wrap gap-2">
            {formData.allowedAPIs.map((api) => (
              <Badge key={api} variant="outline">
                {api}
              </Badge>
            ))}
          </div>
        </div>
      </div>

      <div className="flex items-start gap-3 rounded-md bg-amber-50 border border-amber-200 p-3 text-amber-800">
        <AlertCircle className="h-5 w-5 flex-shrink-0 mt-0.5" />
        <div className="text-sm">
          <p className="font-medium">Important</p>
          <p className="mt-1">
            The API key will only be shown once after creation. Make sure to copy
            and store it securely.
          </p>
        </div>
      </div>
    </div>
  );

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[600px] ui-panel">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-xl font-display">
            <Key className="h-5 w-5 text-[hsl(var(--primary))]" />
            Create API Key
          </DialogTitle>
          <DialogDescription>
            Create a new API key for accessing the RAD Gateway.
          </DialogDescription>
        </DialogHeader>

        <StepIndicator />

        {error && (
          <div className="rounded-md bg-red-50 border border-red-200 p-3 text-red-700 text-sm mb-4">
            {error}
          </div>
        )}

        <div className="py-2">
          {currentStep === 'details' && <DetailsStep />}
          {currentStep === 'permissions' && <PermissionsStep />}
          {currentStep === 'review' && <ReviewStep />}
        </div>

        <DialogFooter className="flex flex-col-reverse sm:flex-row gap-2">
          {currentStep !== 'details' && (
            <Button
              variant="outline"
              onClick={handleBack}
              disabled={isCreating}
              className="w-full sm:w-auto"
            >
              <ChevronLeft className="mr-2 h-4 w-4" />
              Back
            </Button>
          )}
          {currentStep !== 'review' ? (
            <Button onClick={handleNext} className="w-full sm:w-auto">
              Next
              <ChevronRight className="ml-2 h-4 w-4" />
            </Button>
          ) : (
            <Button
              onClick={handleCreate}
              disabled={isCreating}
              className="w-full sm:w-auto"
            >
              {isCreating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  <Key className="mr-2 h-4 w-4" />
                  Create API Key
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
