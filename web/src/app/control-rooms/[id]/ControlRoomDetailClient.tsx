'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Activity,
  AlertCircle,
  ArrowLeft,
  ChevronDown,
  ChevronUp,
  Clock,
  Edit3,
  Filter,
  LayoutGrid,
  List,
  Save,
  Trash2,
  X,
} from 'lucide-react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';
import { StatusBadge } from '@/components/molecules/StatusBadge';
import { DataTable } from '@/components/organisms/DataTable';
import { ColumnDef } from '@tanstack/react-table';
import { useProviders, useUsage, useUsageSummary, UsageRecordResponse } from '@/queries';
import type { Provider } from '@/types';
import { UsageLineChart } from '@/components/charts/UsageLineChart';
import { TokenBarChart } from '@/components/charts/TokenBarChart';

type RoomLayout = 'grid' | 'compact' | 'list';
type TimeRange = '1h' | '6h' | '24h' | '7d';

interface ControlRoomPreset {
  id: string;
  name: string;
  description: string;
  tags: string[];
  layout: RoomLayout;
  isDefault?: boolean;
}

const STORAGE_KEY = 'rad-control-rooms-v1';

const timeRangeMap: Record<TimeRange, number> = {
  '1h': 1,
  '6h': 6,
  '24h': 24,
  '7d': 168,
};

const defaultRooms: ControlRoomPreset[] = [
  {
    id: 'room-main',
    name: 'Main Engine Room',
    description: 'Global command view for all providers and traffic.',
    tags: ['scope:all'],
    layout: 'grid',
    isDefault: true,
  },
];

function getWindowRange(hours: number): { startTime: string; endTime: string } {
  const end = new Date();
  const start = new Date(end.getTime() - hours * 60 * 60 * 1000);
  return {
    startTime: start.toISOString(),
    endTime: end.toISOString(),
  };
}

function filterProvidersByTags(providers: Provider[], tags: string[]): Provider[] {
  if (tags.includes('scope:all') || tags.length === 0) {
    return providers;
  }

  const providerTags = tags
    .filter((tag) => tag.startsWith('provider:'))
    .map((tag) => tag.replace('provider:', '').trim().toLowerCase());
  const statusTags = tags
    .filter((tag) => tag.startsWith('status:'))
    .map((tag) => tag.replace('status:', '').trim().toLowerCase());

  return providers.filter((provider) => {
    const providerName = provider.name.toLowerCase();

    if (providerTags.length > 0 && !providerTags.some((tag) => providerName.includes(tag))) {
      return false;
    }

    if (statusTags.length > 0 && !statusTags.includes(provider.status.toLowerCase())) {
      return false;
    }

    return true;
  });
}

interface ControlRoomDetailClientProps {
  id: string;
}

export default function ControlRoomDetailClient({ id }: ControlRoomDetailClientProps) {
  const router = useRouter();
  const [rooms, setRooms] = useState<ControlRoomPreset[]>(defaultRooms);
  const [room, setRoom] = useState<ControlRoomPreset | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>('6h');
  const [isEditing, setIsEditing] = useState(false);
  const [editForm, setEditForm] = useState<ControlRoomPreset | null>(null);
  const [showFilterPanel, setShowFilterPanel] = useState(false);
  const [activeFilters, setActiveFilters] = useState<string[]>([]);

  // Load rooms from localStorage
  useEffect(() => {
    if (typeof window === 'undefined') return;

    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored) {
      try {
        const parsed = JSON.parse(stored) as ControlRoomPreset[];
        if (Array.isArray(parsed) && parsed.length > 0) {
          setRooms(parsed);
        }
      } catch {
        // Use defaults
      }
    }
  }, []);

  // Find current room
  useEffect(() => {
    const found = rooms.find((r) => r.id === id);
    if (found) {
      setRoom(found);
      setEditForm(found);
      setActiveFilters(found.tags);
    }
  }, [id, rooms]);

  const range = useMemo(() => getWindowRange(timeRangeMap[timeRange]), [timeRange]);
  const { data: providersData, isLoading: providersLoading } = useProviders();
  const { data: usageSummary, isLoading: summaryLoading } = useUsageSummary(range);
  const { data: usageRecords, isLoading: recordsLoading } = useUsage({
    ...range,
    page: 1,
    pageSize: 100,
  });

  const providers = providersData?.providers || [];
  const filteredProviders = useMemo(
    () => filterProvidersByTags(providers, activeFilters),
    [providers, activeFilters]
  );

  // Transform usage data for charts
  const chartData = useMemo(() => {
    if (!usageRecords?.data) return [];
    return usageRecords.data.map((record) => ({
      timestamp: record.startedAt,
      requests: 1,
      errors: record.errorCode ? 1 : 0,
      input: record.promptTokens || 0,
      output: record.completionTokens || 0,
      reasoning: 0,
      cached: 0,
    }));
  }, [usageRecords]);

  // Provider table columns
  const providerColumns: ColumnDef<Provider>[] = [
    {
      accessorKey: 'name',
      header: 'Provider',
      cell: ({ row }) => (
        <div>
          <div className="font-medium text-[var(--ink-900)]">
            {row.original.displayName || row.original.name}
          </div>
          <div className="text-xs text-[var(--ink-500)]">{row.original.name}</div>
        </div>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }) => <StatusBadge status={row.original.status} showPulse={row.original.status === 'healthy'} />,
    },
    {
      accessorKey: 'circuitBreaker',
      header: 'Circuit',
      cell: ({ row }) => <StatusBadge status={row.original.circuitBreaker} />,
    },
    {
      accessorKey: 'latencyMs',
      header: 'Latency',
      cell: ({ row }) => (
        <span className="text-sm">
          {row.original.latencyMs ? `${row.original.latencyMs}ms` : '—'}
        </span>
      ),
    },
    {
      accessorKey: 'requestCount24h',
      header: 'Requests (24h)',
      cell: ({ row }) => <span className="text-sm text-[var(--ink-500)]">{row.original.requestCount24h.toLocaleString()}</span>,
    },
  ];

  // Usage records table columns
  const usageColumns: ColumnDef<UsageRecordResponse>[] = [
    {
      accessorKey: 'incomingApi',
      header: 'API',
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          {row.original.incomingApi === 'chat' ? (
            <Activity className="h-4 w-4 text-[var(--status-info)]" />
          ) : (
            <div className="h-4 w-4 rounded bg-[var(--brass-500)]" />
          )}
          <span className="text-sm uppercase">{row.original.incomingApi}</span>
        </div>
      ),
    },
    {
      accessorKey: 'selectedModel',
      header: 'Model',
      cell: ({ row }) => (
        <span className="text-sm text-[var(--ink-700)]">
          {row.original.selectedModel || row.original.incomingModel}
        </span>
      ),
    },
    {
      accessorKey: 'durationMs',
      header: 'Duration',
      cell: ({ row }) => <span className="text-sm">{row.original.durationMs}ms</span>,
    },
    {
      accessorKey: 'tokens',
      header: 'Tokens',
      cell: ({ row }) => {
        const total = row.original.totalTokens || 0;
        return <span className="text-sm">{total.toLocaleString()}</span>;
      },
    },
    {
      accessorKey: 'startedAt',
      header: 'Time',
      cell: ({ row }) => (
        <span className="text-xs text-[var(--ink-500)]">
          {new Date(row.original.startedAt).toLocaleTimeString()}
        </span>
      ),
    },
  ];

  const saveRoom = () => {
    if (!editForm) return;

    const updatedRooms = rooms.map((r) => (r.id === editForm.id ? editForm : r));
    setRooms(updatedRooms);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(updatedRooms));
    setRoom(editForm);
    setActiveFilters(editForm.tags);
    setIsEditing(false);
  };

  const deleteRoom = () => {
    const updated = rooms.filter((r) => r.id !== id);
    setRooms(updated.length > 0 ? updated : defaultRooms);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(updated.length > 0 ? updated : defaultRooms));
    router.push('/control-rooms');
  };

  const addFilter = (filter: string) => {
    if (!activeFilters.includes(filter)) {
      setActiveFilters([...activeFilters, filter]);
    }
  };

  const removeFilter = (filter: string) => {
    setActiveFilters(activeFilters.filter((f) => f !== filter));
  };

  const loading = providersLoading || summaryLoading || recordsLoading;

  if (!room) {
    return (
      <AppLayout>
        <div className="flex h-64 items-center justify-center">
          <div className="text-center">
            <AlertCircle className="mx-auto mb-4 h-12 w-12 text-[var(--status-warning)]" />
            <h2 className="text-xl font-semibold text-[var(--ink-900)]">Control Room Not Found</h2>
            <p className="mt-2 text-[var(--ink-500)]">The requested control room does not exist.</p>
            <Button className="mt-4" onClick={() => router.push('/control-rooms')}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Control Rooms
            </Button>
          </div>
        </div>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <Button variant="secondary" onClick={() => router.push('/control-rooms')}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              {isEditing ? (
                <input
                  type="text"
                  value={editForm?.name || ''}
                  onChange={(e) =>
                    setEditForm((prev) => (prev ? { ...prev, name: e.target.value } : null))
                  }
                  className="ui-input text-xl font-bold"
                />
              ) : (
                <h1 className="text-2xl font-bold text-[var(--ink-900)]">{room.name}</h1>
              )}
              {isEditing ? (
                <input
                  type="text"
                  value={editForm?.description || ''}
                  onChange={(e) =>
                    setEditForm((prev) => (prev ? { ...prev, description: e.target.value } : null))
                  }
                  className="ui-input mt-1 text-sm"
                  placeholder="Description"
                />
              ) : (
                <p className="text-[var(--ink-500)]">{room.description}</p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {isEditing ? (
              <>
                <Button variant="secondary" onClick={() => setIsEditing(false)}>
                  <X className="mr-2 h-4 w-4" />
                  Cancel
                </Button>
                <Button onClick={saveRoom}>
                  <Save className="mr-2 h-4 w-4" />
                  Save
                </Button>
              </>
            ) : (
              <>
                {!room.isDefault && (
                  <Button variant="danger" onClick={deleteRoom}>
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                  </Button>
                )}
                <Button variant="secondary" onClick={() => setIsEditing(true)}>
                  <Edit3 className="mr-2 h-4 w-4" />
                  Edit
                </Button>
              </>
            )}
          </div>
        </div>

        {/* Filters Bar */}
        <Card>
          <div className="flex flex-wrap items-center gap-4">
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-[var(--ink-500)]" />
              <span className="text-sm font-medium text-[var(--ink-700)]">Active Filters:</span>
            </div>
            <div className="flex flex-wrap gap-2">
              {activeFilters.length === 0 && (
                <span className="text-sm text-[var(--ink-500)]">No filters (showing all)</span>
              )}
              {activeFilters.map((filter) => (
                <span
                  key={filter}
                  className="inline-flex items-center gap-1 rounded-full bg-[var(--brass-100)] px-3 py-1 text-xs font-medium text-[var(--brass-800)]"
                >
                  {filter}
                  <button
                    onClick={() => removeFilter(filter)}
                    className="rounded-full p-0.5 hover:bg-[var(--brass-200)]"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
            </div>
            <Button
              variant="secondary"
              className="ml-auto"
              onClick={() => setShowFilterPanel(!showFilterPanel)}
            >
              {showFilterPanel ? (
                <ChevronUp className="mr-2 h-4 w-4" />
              ) : (
                <ChevronDown className="mr-2 h-4 w-4" />
              )}
              Filters
            </Button>
          </div>

          {showFilterPanel && (
            <div className="mt-4 border-t border-[var(--line-soft)] pt-4">
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                <div>
                  <label className="mb-2 block text-sm font-medium text-[var(--ink-700)]">
                    Provider
                  </label>
                  <select
                    className="ui-input"
                    onChange={(e) => e.target.value && addFilter(`provider:${e.target.value}`)}
                    value=""
                  >
                    <option value="">Select provider...</option>
                    {providers.map((p) => (
                      <option key={p.id} value={p.name.toLowerCase()}>
                        {p.displayName || p.name}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="mb-2 block text-sm font-medium text-[var(--ink-700)]">
                    Status
                  </label>
                  <select
                    className="ui-input"
                    onChange={(e) => e.target.value && addFilter(`status:${e.target.value}`)}
                    value=""
                  >
                    <option value="">Select status...</option>
                    <option value="healthy">Healthy</option>
                    <option value="degraded">Degraded</option>
                    <option value="unhealthy">Unhealthy</option>
                  </select>
                </div>
                <div>
                  <label className="mb-2 block text-sm font-medium text-[var(--ink-700)]">
                    Time Range
                  </label>
                  <select
                    className="ui-input"
                    value={timeRange}
                    onChange={(e) => setTimeRange(e.target.value as TimeRange)}
                  >
                    <option value="1h">Last 1 hour</option>
                    <option value="6h">Last 6 hours</option>
                    <option value="24h">Last 24 hours</option>
                    <option value="7d">Last 7 days</option>
                  </select>
                </div>
                <div>
                  <label className="mb-2 block text-sm font-medium text-[var(--ink-700)]">
                    Layout
                  </label>
                  <div className="flex gap-2">
                    <Button
                      variant={room.layout === 'grid' ? 'primary' : 'secondary'}
                      className="flex-1"
                      onClick={() =>
                        isEditing
                          ? setEditForm((prev) => (prev ? { ...prev, layout: 'grid' } : null))
                          : null
                      }
                    >
                      <LayoutGrid className="mr-2 h-4 w-4" />
                      Grid
                    </Button>
                    <Button
                      variant={room.layout === 'list' ? 'primary' : 'secondary'}
                      className="flex-1"
                      onClick={() =>
                        isEditing
                          ? setEditForm((prev) => (prev ? { ...prev, layout: 'list' } : null))
                          : null
                      }
                    >
                      <List className="mr-2 h-4 w-4" />
                      List
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </Card>

        {/* Metrics Overview */}
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Filtered Providers</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {loading ? '...' : `${filteredProviders.length}/${providers.length}`}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Total Requests</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {loading ? '...' : (usageSummary?.totalRequests || 0).toLocaleString()}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Avg Latency</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {loading ? '...' : `${Math.round(usageSummary?.avgDurationMs || 0)}ms`}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Error Rate</div>
            <div
              className={`mt-1 text-2xl font-bold ${
                (usageSummary?.errorRate || 0) > 2
                  ? 'text-[var(--status-critical)]'
                  : 'text-[var(--ink-900)]'
              }`}
            >
              {loading ? '...' : `${(usageSummary?.errorRate || 0).toFixed(2)}%`}
            </div>
          </Card>
        </div>

        {/* Charts */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <Card title="Request Volume">
            <div className="h-64">
              {loading ? (
                <div className="flex h-full items-center justify-center">
                  <Clock className="h-8 w-8 animate-spin text-[var(--ink-400)]" />
                </div>
              ) : (
                <UsageLineChart
                  data={chartData}
                  title="Requests"
                />
              )}
            </div>
          </Card>
          <Card title="Token Usage">
            <div className="h-64">
              {loading ? (
                <div className="flex h-full items-center justify-center">
                  <Clock className="h-8 w-8 animate-spin text-[var(--ink-400)]" />
                </div>
              ) : (
                <TokenBarChart data={chartData} title="Tokens" />
              )}
            </div>
          </Card>
        </div>

        {/* Providers Table */}
        <Card title={`Providers (${filteredProviders.length})`}>
          <DataTable
            columns={providerColumns}
            data={filteredProviders}
            pageSize={10}
          />
        </Card>

        {/* Recent Usage */}
        <Card title="Recent Usage Records">
          <DataTable
            columns={usageColumns}
            data={(usageRecords?.data?.slice(0, 20) || []) as UsageRecordResponse[]}
            pageSize={10}
          />
        </Card>
      </div>
    </AppLayout>
  );
}
