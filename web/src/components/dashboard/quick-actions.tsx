'use client';

import * as React from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import {
  Plus,
  Key,
  Server,
  FolderOpen,
  MoreHorizontal,
  Settings,
  FileText,
  Users,
  LucideIcon,
} from 'lucide-react';

export interface QuickAction {
  id: string;
  label: string;
  icon: LucideIcon;
  href?: string;
  onClick?: () => void;
  variant?: 'default' | 'secondary' | 'outline' | 'ghost';
  description?: string;
}

export interface QuickActionsProps {
  actions?: QuickAction[];
  showOverflow?: boolean;
  className?: string;
}

/**
 * Default quick actions for the dashboard
 */
export const defaultQuickActions: QuickAction[] = [
  {
    id: 'new-api-key',
    label: 'New API Key',
    icon: Key,
    href: '/api-keys/new',
    variant: 'default',
    description: 'Create a new API key',
  },
  {
    id: 'add-provider',
    label: 'Add Provider',
    icon: Server,
    href: '/providers/new',
    variant: 'secondary',
    description: 'Configure a new AI provider',
  },
];

/**
 * Overflow actions shown in dropdown
 */
export const overflowActions: QuickAction[] = [
  {
    id: 'new-project',
    label: 'New Project',
    icon: FolderOpen,
    href: '/projects/new',
    description: 'Create a new workspace project',
  },
  {
    id: 'view-logs',
    label: 'View Logs',
    icon: FileText,
    href: '/logs',
    description: 'View system logs',
  },
  {
    id: 'manage-users',
    label: 'Manage Users',
    icon: Users,
    href: '/users',
    description: 'Manage user access',
  },
  {
    id: 'settings',
    label: 'Settings',
    icon: Settings,
    href: '/settings',
    description: 'System configuration',
  },
];

export function QuickActions({
  actions = defaultQuickActions,
  showOverflow = true,
  className,
}: QuickActionsProps) {
  const router = useRouter();

  const handleAction = (action: QuickAction) => {
    if (action.onClick) {
      action.onClick();
    } else if (action.href) {
      router.push(action.href);
    }
  };

  return (
    <div className={cn("flex items-center gap-2", className)}>
      {/* Primary Actions */}
      {actions.slice(0, 2).map((action) => (
        <Button
          key={action.id}
          variant={action.variant || 'default'}
          size="default"
          onClick={() => handleAction(action)}
          className={cn(
            "gap-2 transition-all duration-200",
            action.variant === 'default' && [
              "bg-[var(--brass-500)] hover:bg-[var(--brass-600)]",
              "text-white shadow-md hover:shadow-lg",
              "border-0"
            ],
            action.variant === 'secondary' && [
              "bg-[var(--copper-500)]/10 hover:bg-[var(--copper-500)]/20",
              "text-[var(--copper-600)] border-[var(--copper-500)]/30"
            ]
          )}
        >
          <action.icon className="h-4 w-4" />
          {action.label}
        </Button>
      ))}

      {/* Overflow Dropdown */}
      {showOverflow && (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="outline"
              size="icon"
              className={cn(
                "border-[var(--line-soft)] bg-transparent",
                "hover:bg-[var(--brass-500)]/10 hover:border-[var(--brass-500)]/30",
                "transition-all duration-200"
              )}
            >
              <MoreHorizontal className="h-4 w-4 text-[var(--ink-500)]" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent 
            align="end" 
            className="w-56 border-[var(--line-soft)] bg-[var(--panel-bg)]"
          >
            <DropdownMenuItem 
              className="text-xs font-medium text-[var(--ink-500)] uppercase tracking-wider cursor-default"
              disabled
            >
              Quick Actions
            </DropdownMenuItem>
            <DropdownMenuSeparator className="bg-[var(--line-soft)]" />
            
            {actions.slice(2).map((action) => (
              <DropdownMenuItem
                key={action.id}
                onClick={() => handleAction(action)}
                className={cn(
                  "gap-2 cursor-pointer",
                  "focus:bg-[var(--brass-500)]/10 focus:text-[var(--ink-900)]"
                )}
              >
                <action.icon className="h-4 w-4 text-[var(--brass-500)]" />
                <span className="text-[var(--ink-700)]">{action.label}</span>
              </DropdownMenuItem>
            ))}

            {actions.length > 2 && overflowActions.length > 0 && (
              <DropdownMenuSeparator className="bg-[var(--line-soft)]" />
            )}

            {overflowActions.map((action) => (
              <DropdownMenuItem
                key={action.id}
                onClick={() => handleAction(action)}
                className={cn(
                  "gap-2 cursor-pointer",
                  "focus:bg-[var(--brass-500)]/10 focus:text-[var(--ink-900)]"
                )}
              >
                <action.icon className="h-4 w-4 text-[var(--copper-500)]" />
                <span className="text-[var(--ink-700)]">{action.label}</span>
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}

/**
 * Compact version for mobile or constrained spaces
 */
export function QuickActionsCompact({ className }: { className?: string }) {
  const router = useRouter();

  return (
    <div className={cn("flex items-center gap-1", className)}>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => router.push('/api-keys/new')}
        className="h-8 px-2 text-[var(--ink-600)] hover:text-[var(--brass-600)] hover:bg-[var(--brass-500)]/10"
      >
        <Key className="h-4 w-4 mr-1" />
        <span className="hidden sm:inline">API Key</span>
      </Button>
      
      <Button
        variant="ghost"
        size="sm"
        onClick={() => router.push('/providers/new')}
        className="h-8 px-2 text-[var(--ink-600)] hover:text-[var(--copper-600)] hover:bg-[var(--copper-500)]/10"
      >
        <Server className="h-4 w-4 mr-1" />
        <span className="hidden sm:inline">Provider</span>
      </Button>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-[var(--ink-500)] hover:text-[var(--ink-900)]"
          >
            <Plus className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="border-[var(--line-soft)] bg-[var(--panel-bg)]">
          {overflowActions.map((action) => (
            <DropdownMenuItem
              key={action.id}
              onClick={() => router.push(action.href || '/')}
              className="gap-2 cursor-pointer focus:bg-[var(--brass-500)]/10"
            >
              <action.icon className="h-4 w-4 text-[var(--steel-500)]" />
              <span className="text-[var(--ink-700)]">{action.label}</span>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
