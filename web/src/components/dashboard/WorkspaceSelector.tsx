/**
 * RAD Gateway Admin UI - Workspace Selector Component
 * State Management Engineer - Phase 2 Implementation
 *
 * Workspace selection dropdown with favorites.
 */

import { useState } from 'react';
import { useWorkspace } from '../../hooks/useWorkspace';
import { Workspace } from '../../types';
import { Skeleton } from '../common/Skeleton';

interface WorkspaceSelectorProps {
  className?: string;
}

export function WorkspaceSelector({ className = '' }: WorkspaceSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const {
    current,
    workspaces,
    recent,
    favorites,
    isLoading,
    setCurrent,
    toggleFavorite,
  } = useWorkspace();

  const handleSelect = (workspace: Workspace) => {
    setCurrent(workspace);
    setIsOpen(false);
  };

  if (isLoading) {
    return (
      <div className={className}>
        <Skeleton className="h-10 w-48" />
      </div>
    );
  }

  return (
    <div className={`relative ${className}`}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center space-x-2 px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-800"
      >
        <span className="font-medium text-gray-900 dark:text-white">
          {current?.name || 'Select Workspace'}
        </span>
        <svg
          className={`w-4 h-4 text-gray-500 transition-transform ${
            isOpen ? 'rotate-180' : ''
          }`}
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M19 9l-7 7-7-7"
          />
        </svg>
      </button>

      {isOpen && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />
          <div className="absolute top-full left-0 mt-2 w-64 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 z-50">
            {/* Recent Workspaces */}
            {recent.length > 0 && (
              <div className="p-2">
                <div className="px-3 py-1 text-xs font-medium text-gray-500 dark:text-gray-400">
                  Recent
                </div>
                {recent.slice(0, 3).map((workspace) => (
                  <WorkspaceItem
                    key={workspace.id}
                    workspace={workspace}
                    isSelected={current?.id === workspace.id}
                    isFavorite={favorites.includes(workspace.id)}
                    onSelect={handleSelect}
                    onToggleFavorite={toggleFavorite}
                  />
                ))}
              </div>
            )}

            {/* All Workspaces */}
            <div className="p-2 border-t border-gray-200 dark:border-gray-700">
              <div className="px-3 py-1 text-xs font-medium text-gray-500 dark:text-gray-400">
                All Workspaces
              </div>
              {workspaces.map((workspace) => (
                <WorkspaceItem
                  key={workspace.id}
                  workspace={workspace}
                  isSelected={current?.id === workspace.id}
                  isFavorite={favorites.includes(workspace.id)}
                  onSelect={handleSelect}
                  onToggleFavorite={toggleFavorite}
                />
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

interface WorkspaceItemProps {
  workspace: Workspace;
  isSelected: boolean;
  isFavorite: boolean;
  onSelect: (workspace: Workspace) => void;
  onToggleFavorite: (id: string) => void;
}

function WorkspaceItem({
  workspace,
  isSelected,
  isFavorite,
  onSelect,
  onToggleFavorite,
}: WorkspaceItemProps) {
  return (
    <div
      className={`flex items-center justify-between px-3 py-2 rounded cursor-pointer ${
        isSelected
          ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400'
          : 'hover:bg-gray-100 dark:hover:bg-gray-700'
      }`}
    >
      <button
        onClick={() => onSelect(workspace)}
        className="flex-1 text-left text-sm font-medium text-gray-900 dark:text-white"
      >
        {workspace.name}
      </button>
      <button
        onClick={(e) => {
          e.stopPropagation();
          onToggleFavorite(workspace.id);
        }}
        className={`p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-600 ${
          isFavorite ? 'text-yellow-500' : 'text-gray-400'
        }`}
        aria-label={isFavorite ? 'Remove from favorites' : 'Add to favorites'}
      >
        <svg
          className="w-4 h-4"
          fill={isFavorite ? 'currentColor' : 'none'}
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
          />
        </svg>
      </button>
    </div>
  );
}
