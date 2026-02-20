import React from 'react';
import { Card } from '../atoms/Card';
import { Pagination } from '../molecules/Pagination';
import { EmptyState } from '../molecules/EmptyState';
import { Search } from 'lucide-react';

interface Column<T> {
  key: string;
  header: string;
  width?: string;
  render?: (item: T) => React.ReactNode;
}

interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (item: T) => string;
  loading?: boolean;
  searchable?: boolean;
  searchValue?: string;
  onSearch?: (value: string) => void;
  pagination?: {
    currentPage: number;
    totalPages: number;
    totalItems: number;
    itemsPerPage: number;
    onPageChange: (page: number) => void;
  };
  emptyState?: {
    title: string;
    description?: string;
    action?: { label: string; onClick: () => void };
  };
}

export function DataTable<T>({
  data,
  columns,
  keyExtractor,
  loading = false,
  searchable = false,
  searchValue = '',
  onSearch,
  pagination,
  emptyState,
}: DataTableProps<T>) {
  if (loading) {
    return (
      <Card>
        <div className="animate-pulse space-y-4 p-6">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="flex gap-4">
              {[...Array(4)].map((_, j) => (
                <div key={j} className="h-4 flex-1 rounded bg-[rgba(43,32,21,0.2)]"></div>
              ))}
            </div>
          ))}
        </div>
      </Card>
    );
  }

  if (data.length === 0 && emptyState) {
    return (
      <Card>
        <EmptyState
          title={emptyState.title}
          {...(emptyState.description && { description: emptyState.description })}
          {...(emptyState.action && { action: emptyState.action })}
          icon={<Search className="w-12 h-12" />}
        />
      </Card>
    );
  }

  return (
    <Card>
      {searchable && onSearch && (
        <div className="px-6 pt-6 pb-2">
          <input
            type="text"
            placeholder="Search..."
            value={searchValue}
            onChange={(e) => onSearch(e.target.value)}
            className="ui-input max-w-md"
          />
        </div>
      )}

      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="border-b border-[var(--line-strong)] bg-[rgba(43,32,21,0.08)]">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  className="px-6 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em] text-[var(--ink-500)]"
                  style={{ width: column.width }}
                >
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-[var(--line-soft)]">
            {data.map((item) => (
              <tr key={keyExtractor(item)} className="hover:bg-[rgba(43,32,21,0.06)]">
                {columns.map((column) => (
                  <td key={`${keyExtractor(item)}-${column.key}`} className="whitespace-nowrap px-6 py-4 text-[var(--ink-900)]">
                    {column.render
                      ? column.render(item)
                      : String(((item as unknown as Record<string, unknown>)[column.key] ?? ''))}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {pagination && (
        <Pagination
          currentPage={pagination.currentPage}
          totalPages={pagination.totalPages}
          totalItems={pagination.totalItems}
          itemsPerPage={pagination.itemsPerPage}
          onPageChange={pagination.onPageChange}
        />
      )}
    </Card>
  );
}
