'use client';

import { useState, useMemo } from 'react';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getPaginationRowModel,
  getFilteredRowModel,
  type SortingState,
  type ColumnDef,
  flexRender,
  type RowSelectionState,
} from '@tanstack/react-table';
import {
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Copy,
  Edit,
  Trash2,
  Shield,
  ShieldAlert,
  ShieldCheck,
  ShieldX,
  MoreHorizontal,
} from 'lucide-react';

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

import type { APIKeyResponse } from '@/queries/apikeys';

interface APIKeyRow {
  id: string;
  name: string;
  keyPreview: string;
  status: 'active' | 'revoked' | 'expired';
  createdAt: string;
  lastUsedAt: string | null;
  permissions?: string[];
  createdBy?: string;
}

interface ApiKeysTableProps {
  data: APIKeyResponse[];
  isLoading: boolean;
  onEdit?: (key: APIKeyResponse) => void;
  onRevoke?: (keyId: string) => void;
  onDelete?: (keyId: string) => void;
  onCopy?: (key: string) => void;
  searchQuery?: string;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

function formatLastUsed(value?: string | null): string {
  if (!value) return 'Never';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Never';
  const deltaMs = Date.now() - date.getTime();
  const mins = Math.floor(deltaMs / 60000);
  if (mins < 1) return 'Just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function getStatusIcon(status: string) {
  switch (status) {
    case 'active':
      return <ShieldCheck className="h-3.5 w-3.5" />;
    case 'revoked':
      return <ShieldX className="h-3.5 w-3.5" />;
    case 'expired':
      return <ShieldAlert className="h-3.5 w-3.5" />;
    default:
      return <Shield className="h-3.5 w-3.5" />;
  }
}

function getStatusBadge(status: string) {
  const variants: Record<string, { variant: 'default' | 'secondary' | 'destructive' | 'outline'; label: string }> = {
    active: { variant: 'default', label: 'Active' },
    revoked: { variant: 'destructive', label: 'Revoked' },
    expired: { variant: 'secondary', label: 'Expired' },
  };

  const config = variants[status] || { variant: 'secondary', label: status };

  return (
    <Badge variant={config.variant as any} className="flex items-center gap-1.5">
      {getStatusIcon(status)}
      {config.label}
    </Badge>
  );
}

export function ApiKeysTable({
  data,
  isLoading,
  onEdit,
  onRevoke,
  onDelete,
  onCopy,
  searchQuery = '',
}: ApiKeysTableProps) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: 10 });

  const rows = useMemo<APIKeyRow[]>(() => {
    return data.map((k) => ({
      id: k.id,
      name: k.name,
      keyPreview: k.keyPreview,
      status: k.status,
      createdAt: k.createdAt,
      lastUsedAt: k.lastUsedAt || null,
      permissions: k.allowedAPIs || [],
      ...(k.createdBy ? { createdBy: k.createdBy } : {}),
    }));
  }, [data]);

  const columns = useMemo<ColumnDef<APIKeyRow>[]>(
    () => [
      {
        id: 'select',
        header: ({ table }) => (
          <Checkbox
            checked={
              table.getIsAllPageRowsSelected() ||
              (table.getIsSomePageRowsSelected() && 'indeterminate')
            }
            onCheckedChange={(value) =>
              table.toggleAllPageRowsSelected(!!value)
            }
            aria-label="Select all"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        ),
        enableSorting: false,
        enableHiding: false,
      },
      {
        accessorKey: 'name',
        header: ({ column }) => {
          const isSorted = column.getIsSorted();
          return (
            <Button
              variant="ghost"
              onClick={() => column.toggleSorting(isSorted === 'asc')}
              className="h-8 px-2 -ml-2 font-medium"
            >
              Name
              {isSorted === 'asc' ? (
                <ChevronUp className="ml-2 h-4 w-4" />
              ) : isSorted === 'desc' ? (
                <ChevronDown className="ml-2 h-4 w-4" />
              ) : (
                <ChevronsUpDown className="ml-2 h-4 w-4 opacity-50" />
              )}
            </Button>
          );
        },
        cell: ({ row }) => (
          <div className="font-medium text-[hsl(var(--foreground))]">
            {row.getValue('name')}
          </div>
        ),
      },
      {
        accessorKey: 'keyPreview',
        header: 'Key',
        cell: ({ row }) => (
          <code className="rounded bg-[hsl(var(--muted))]/20 px-2 py-1 text-sm font-mono text-[hsl(var(--muted-foreground))]">
            {row.getValue('keyPreview')}
          </code>
        ),
      },
      {
        accessorKey: 'status',
        header: ({ column }) => {
          const isSorted = column.getIsSorted();
          return (
            <Button
              variant="ghost"
              onClick={() => column.toggleSorting(isSorted === 'asc')}
              className="h-8 px-2 -ml-2 font-medium"
            >
              Status
              {isSorted === 'asc' ? (
                <ChevronUp className="ml-2 h-4 w-4" />
              ) : isSorted === 'desc' ? (
                <ChevronDown className="ml-2 h-4 w-4" />
              ) : (
                <ChevronsUpDown className="ml-2 h-4 w-4 opacity-50" />
              )}
            </Button>
          );
        },
        cell: ({ row }) => getStatusBadge(row.getValue('status')),
      },
      {
        accessorKey: 'createdAt',
        header: ({ column }) => {
          const isSorted = column.getIsSorted();
          return (
            <Button
              variant="ghost"
              onClick={() => column.toggleSorting(isSorted === 'asc')}
              className="h-8 px-2 -ml-2 font-medium"
            >
              Created
              {isSorted === 'asc' ? (
                <ChevronUp className="ml-2 h-4 w-4" />
              ) : isSorted === 'desc' ? (
                <ChevronDown className="ml-2 h-4 w-4" />
              ) : (
                <ChevronsUpDown className="ml-2 h-4 w-4 opacity-50" />
              )}
            </Button>
          );
        },
        cell: ({ row }) => formatDate(row.getValue('createdAt')),
      },
      {
        accessorKey: 'lastUsedAt',
        header: ({ column }) => {
          const isSorted = column.getIsSorted();
          return (
            <Button
              variant="ghost"
              onClick={() => column.toggleSorting(isSorted === 'asc')}
              className="h-8 px-2 -ml-2 font-medium"
            >
              Last Used
              {isSorted === 'asc' ? (
                <ChevronUp className="ml-2 h-4 w-4" />
              ) : isSorted === 'desc' ? (
                <ChevronDown className="ml-2 h-4 w-4" />
              ) : (
                <ChevronsUpDown className="ml-2 h-4 w-4 opacity-50" />
              )}
            </Button>
          );
        },
        cell: ({ row }) => {
          const lastUsed = row.getValue('lastUsedAt') as string | null;
          return (
            <span className="text-[hsl(var(--muted-foreground))]">
              {formatLastUsed(lastUsed)}
            </span>
          );
        },
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const key = row.original;
          const originalKey = data.find((k) => k.id === key.id);

          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <MoreHorizontal className="h-4 w-4" />
                  <span className="sr-only">Open menu</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuItem
                  onClick={() => onCopy?.(key.keyPreview)}
                  className="cursor-pointer"
                >
                  <Copy className="mr-2 h-4 w-4" />
                  Copy Key
                </DropdownMenuItem>
                {key.status === 'active' && (
                  <>
                    <DropdownMenuItem
                      onClick={() => originalKey && onEdit?.(originalKey)}
                      className="cursor-pointer"
                    >
                      <Edit className="mr-2 h-4 w-4" />
                      Edit
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onClick={() => onRevoke?.(key.id)}
                      className="cursor-pointer text-[hsl(var(--destructive))]"
                    >
                      <ShieldX className="mr-2 h-4 w-4" />
                      Revoke
                    </DropdownMenuItem>
                  </>
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => onDelete?.(key.id)}
                  className="cursor-pointer text-[hsl(var(--destructive))]"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          );
        },
      },
    ],
    [data, onEdit, onRevoke, onDelete, onCopy]
  );

  const table = useReactTable({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    onSortingChange: setSorting,
    onRowSelectionChange: setRowSelection,
    onPaginationChange: setPagination,
    state: {
      sorting,
      rowSelection,
      pagination,
      globalFilter: searchQuery,
    },
    enableRowSelection: true,
    enableMultiRowSelection: true,
  });

  const selectedCount = Object.keys(rowSelection).length;

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-8 w-32" />
        </div>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                {Array.from({ length: 7 }).map((_, i) => (
                  <TableHead key={i}>
                    <Skeleton className="h-4 w-full" />
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={i}>
                  {Array.from({ length: 7 }).map((_, j) => (
                    <TableCell key={j}>
                      <Skeleton className="h-4 w-full" />
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Selection Bar */}
      {selectedCount > 0 && (
        <div className="flex items-center justify-between rounded-md bg-[hsl(var(--primary))]/10 p-3 border border-[hsl(var(--primary))]/20">
          <span className="text-sm font-medium">
            {selectedCount} {selectedCount === 1 ? 'key' : 'keys'} selected
          </span>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                const selectedKeys = Object.keys(rowSelection);
                if (selectedKeys.length > 0 && selectedKeys[0]) {
                  onRevoke?.(selectedKeys[0]);
                }
              }}
            >
              Revoke Selected
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.toggleAllPageRowsSelected(false)}
            >
              Clear Selection
            </Button>
          </div>
        </div>
      )}

      {/* Table */}
      <div className="rounded-md border border-[hsl(var(--border))]">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow
                key={headerGroup.id}
                className="bg-[hsl(var(--muted))]/30 hover:bg-[hsl(var(--muted))]/40"
              >
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead key={header.id} className="font-semibold">
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  key={row.id}
                  data-state={row.getIsSelected() && 'selected'}
                  className="transition-colors"
                >
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center text-[hsl(var(--muted-foreground))]"
                >
                  {searchQuery
                    ? 'No API keys match your search.'
                    : 'No API keys found.'}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-[hsl(var(--muted-foreground))]">
          <span>
            Page {table.getState().pagination.pageIndex + 1} of{' '}
            {table.getPageCount() || 1}
          </span>
          <span>|</span>
          <span>{data.length} total keys</span>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-[hsl(var(--muted-foreground))]">
              Rows per page:
            </span>
            <Select
              value={String(table.getState().pagination.pageSize)}
              onValueChange={(value) => {
                table.setPageSize(Number(value));
              }}
            >
              <SelectTrigger className="h-8 w-20">
                <SelectValue placeholder="10" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="10">10</SelectItem>
                <SelectItem value="25">25</SelectItem>
                <SelectItem value="50">50</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
            >
              Next
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
