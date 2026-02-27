'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { Activity, AlertCircle, ArrowRight, Plus, Server, Trash2, Zap } from 'lucide-react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';
import { StatusBadge } from '@/components/molecules/StatusBadge';
import { useProviders, useUsage, useUsageSummary } from '@/queries';
import type { Provider } from '@/types';

type RoomLayout = 'grid' | 'compact' | 'list';

interface ControlRoomPreset {
  id: string;
  name: string;
  description: string;
  tags: string[];
  layout: RoomLayout;
  isDefault?: boolean;
}

const STORAGE_KEY = 'rad-control-rooms-v1';
const LIVE_WINDOW_HOURS = 6;

const defaultRooms: ControlRoomPreset[] = [
  {
    id: 'room-main',
    name: 'Main Engine Room',
    description: 'Global command view for all providers and traffic.',
    tags: ['scope:all', 'layout:grid'],
    layout: 'grid',
    isDefault: true,
  },
  {
    id: 'room-cost',
    name: 'Cost Sentinel',
    description: 'Focused on usage economics and error drift.',
    tags: ['metric:cost', 'metric:error'],
    layout: 'compact',
  },
  {
    id: 'room-openai',
    name: 'OpenAI Pressure Deck',
    description: 'Dedicated monitoring lane for OpenAI traffic.',
    tags: ['provider:openai'],
    layout: 'list',
  },
];

function parseTagInput(raw: string): string[] {
  return raw
    .split(',')
    .map((part) => part.trim().toLowerCase())
    .filter((part) => part.length > 0);
}

function getWindowRange(): { startTime: string; endTime: string } {
  const end = new Date();
  const start = new Date(end.getTime() - LIVE_WINDOW_HOURS * 60 * 60 * 1000);
  return {
    startTime: start.toISOString(),
    endTime: end.toISOString(),
  };
}

function filterProvidersByRoom(providers: Provider[], room: ControlRoomPreset | undefined): Provider[] {
  if (!room || room.tags.includes('scope:all')) {
    return providers;
  }

  const providerTags = room.tags
    .filter((tag) => tag.startsWith('provider:'))
    .map((tag) => tag.replace('provider:', '').trim());
  const statusTags = room.tags
    .filter((tag) => tag.startsWith('status:'))
    .map((tag) => tag.replace('status:', '').trim());

  return providers.filter((provider) => {
    const providerName = provider.name.toLowerCase();

    if (providerTags.length > 0 && !providerTags.some((tag) => providerName.includes(tag))) {
      return false;
    }

    if (statusTags.length > 0 && !statusTags.includes(provider.status)) {
      return false;
    }

    return true;
  });
}

export default function ControlRoomsPage() {
  const [rooms, setRooms] = useState<ControlRoomPreset[]>(defaultRooms);
  const [activeRoomId, setActiveRoomId] = useState(defaultRooms[0]?.id || '');
  const [newRoomName, setNewRoomName] = useState('');
  const [newRoomDescription, setNewRoomDescription] = useState('');
  const [newRoomTags, setNewRoomTags] = useState('');

  const range = useMemo(() => getWindowRange(), []);
  const { data: providersData, isLoading: providersLoading } = useProviders();
  const { data: usageSummary, isLoading: summaryLoading } = useUsageSummary(range);
  const { data: usageRecords, isLoading: recordsLoading } = useUsage({ ...range, page: 1, pageSize: 15 });

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      return;
    }

    try {
      const parsed = JSON.parse(stored) as ControlRoomPreset[];
      if (Array.isArray(parsed) && parsed.length > 0) {
        setRooms(parsed);
        setActiveRoomId(parsed[0]?.id || '');
      }
    } catch {
      setRooms(defaultRooms);
      setActiveRoomId(defaultRooms[0]?.id || '');
    }
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(rooms));
  }, [rooms]);

  const providers = providersData?.providers || [];
  const activeRoom = rooms.find((room) => room.id === activeRoomId);
  const roomProviders = filterProvidersByRoom(providers, activeRoom);
  const roomStatusCounts = {
    healthy: roomProviders.filter((provider) => provider.status === 'healthy').length,
    degraded: roomProviders.filter((provider) => provider.status === 'degraded').length,
    unhealthy: roomProviders.filter((provider) => provider.status === 'unhealthy').length,
  };

  const requestsPerMinute = (usageSummary?.totalRequests || 0) / (LIVE_WINDOW_HOURS * 60);
  const avgLatency = usageSummary?.avgDurationMs || 0;
  const errorRate = usageSummary?.errorRate || 0;

  const alerts = useMemo(() => {
    const computed: Array<{ level: 'warning' | 'critical' | 'info'; message: string; time: string }> = [];

    roomProviders.forEach((provider) => {
      if (provider.status === 'unhealthy') {
        computed.push({
          level: 'critical',
          message: `${provider.displayName || provider.name} is unhealthy`,
          time: 'live',
        });
      } else if (provider.status === 'degraded') {
        computed.push({
          level: 'warning',
          message: `${provider.displayName || provider.name} latency is degraded`,
          time: 'live',
        });
      }
    });

    if (errorRate > 2) {
      computed.push({
        level: 'critical',
        message: `Error rate elevated at ${errorRate.toFixed(2)}%`,
        time: 'window',
      });
    }

    if (avgLatency > 900) {
      computed.push({
        level: 'warning',
        message: `Latency threshold breach: ${Math.round(avgLatency)}ms`,
        time: 'window',
      });
    }

    if (computed.length === 0) {
      computed.push({
        level: 'info',
        message: 'No active alerts in this control room.',
        time: 'live',
      });
    }

    return computed.slice(0, 6);
  }, [avgLatency, errorRate, roomProviders]);

  const createRoom = () => {
    if (!newRoomName.trim()) {
      return;
    }

    const room: ControlRoomPreset = {
      id: `room-${Date.now()}`,
      name: newRoomName.trim(),
      description: newRoomDescription.trim() || 'Custom control room',
      tags: parseTagInput(newRoomTags),
      layout: 'grid',
    };

    setRooms((previous) => [room, ...previous]);
    setActiveRoomId(room.id);
    setNewRoomName('');
    setNewRoomDescription('');
    setNewRoomTags('');
  };

  const deleteRoom = (roomId: string) => {
    const remaining = rooms.filter((room) => room.id !== roomId);
    setRooms(remaining.length > 0 ? remaining : defaultRooms);
    if (activeRoomId === roomId) {
      const next = remaining[0] || defaultRooms[0];
      setActiveRoomId(next?.id || '');
    }
  };

  const loading = providersLoading || summaryLoading || recordsLoading;

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-col gap-4">
          <div>
            <h1 className="text-3xl font-bold text-[var(--ink-900)]">Control Rooms</h1>
            <p className="text-[var(--ink-500)]">
              Foglight-style operational stations tuned by tags, scope, and monitoring posture.
            </p>
          </div>

          <Card>
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <input
                value={newRoomName}
                onChange={(event) => setNewRoomName(event.target.value)}
                placeholder="Room name"
                className="ui-input"
              />
              <input
                value={newRoomDescription}
                onChange={(event) => setNewRoomDescription(event.target.value)}
                placeholder="Description"
                className="ui-input"
              />
              <input
                value={newRoomTags}
                onChange={(event) => setNewRoomTags(event.target.value)}
                placeholder="Tags (provider:openai,status:degraded)"
                className="ui-input"
              />
              <Button onClick={createRoom}>
                <Plus className="mr-2 h-4 w-4" />
                Create Room
              </Button>
            </div>
          </Card>
        </div>

        <Card title="Available Rooms">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {rooms.map((room) => {
              const selected = room.id === activeRoomId;
              return (
                <button
                  key={room.id}
                  type="button"
                  onClick={() => setActiveRoomId(room.id)}
                  className={`rounded-lg border p-4 text-left transition-colors ${
                    selected
                      ? 'border-[var(--brass-500)] bg-[rgba(177,133,50,0.16)]'
                      : 'border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] hover:bg-[rgba(43,32,21,0.12)]'
                  }`}
                >
                  <div className="mb-2 flex items-center justify-between gap-2">
                    <p className="font-semibold text-[var(--ink-900)]">{room.name}</p>
                    <div className="flex items-center gap-1">
                      {room.isDefault && <StatusBadge status="active" />}
                      {!room.isDefault && (
                        <span
                          role="button"
                          tabIndex={0}
                          onClick={(event) => {
                            event.stopPropagation();
                            deleteRoom(room.id);
                          }}
                          onKeyDown={(event) => {
                            if (event.key === 'Enter' || event.key === ' ') {
                              event.preventDefault();
                              deleteRoom(room.id);
                            }
                          }}
                          className="rounded-md p-1 text-[var(--ink-500)] hover:bg-[rgba(43,32,21,0.12)]"
                        >
                          <Trash2 className="h-4 w-4" />
                        </span>
                      )}
                    </div>
                  </div>
                  <p className="mb-2 text-sm text-[var(--ink-500)]">{room.description}</p>
                  <p className="text-xs uppercase tracking-[0.08em] text-[var(--ink-500)]">
                    {room.tags.length > 0 ? room.tags.join(' · ') : 'No filters'}
                  </p>
                </button>
              );
            })}
          </div>
        </Card>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Requests / Min</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {loading ? '...' : requestsPerMinute.toFixed(1)}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Avg Latency</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {loading ? '...' : `${Math.round(avgLatency)}ms`}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Error Rate</div>
            <div className="mt-1 text-2xl font-bold text-[var(--status-critical)]">
              {loading ? '...' : `${errorRate.toFixed(2)}%`}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Healthy Providers</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {loading ? '...' : `${roomStatusCounts.healthy}/${roomProviders.length}`}
            </div>
          </Card>
        </div>

        <Card title="Provider Health Mesh">
          <div className="space-y-3">
            {roomProviders.length === 0 && !loading && (
              <p className="text-sm text-[var(--ink-500)]">No providers match this control room scope.</p>
            )}

            {roomProviders.map((provider) => (
              <div
                key={provider.id}
                className="flex items-center justify-between rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] px-4 py-3"
              >
                <div>
                  <p className="font-medium text-[var(--ink-900)]">{provider.displayName || provider.name}</p>
                  <p className="text-sm text-[var(--ink-500)]">
                    Circuit {provider.circuitBreaker} · {provider.latencyMs ? `${provider.latencyMs}ms` : 'No latency sample'}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <StatusBadge status={provider.status} showPulse={provider.status === 'healthy'} />
                  <StatusBadge status={provider.circuitBreaker} />
                </div>
              </div>
            ))}
          </div>
        </Card>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <Card title="Recent Alerts">
            <div className="space-y-3">
              {alerts.map((alert, index) => (
                <div
                  key={`${alert.message}-${index}`}
                  className="flex items-start gap-3 rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] px-3 py-2"
                >
                  <AlertCircle
                    className={`mt-0.5 h-5 w-5 ${
                      alert.level === 'critical'
                        ? 'text-[var(--status-critical)]'
                        : alert.level === 'warning'
                          ? 'text-[var(--status-warning)]'
                          : 'text-[var(--status-info)]'
                    }`}
                  />
                  <div className="flex-1">
                    <p className="text-sm font-medium text-[var(--ink-900)]">{alert.message}</p>
                    <p className="text-xs text-[var(--ink-500)]">{alert.time}</p>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          <Card title="Telemetry Events">
            <div className="space-y-3">
              {(usageRecords?.data || []).slice(0, 6).map((record) => (
                <div key={record.id} className="flex items-center justify-between rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] px-3 py-2">
                  <div className="flex items-center gap-2">
                    {record.incomingApi === 'chat' ? (
                      <Activity className="h-4 w-4 text-[var(--status-info)]" />
                    ) : record.incomingApi === 'responses' ? (
                      <Zap className="h-4 w-4 text-[var(--brass-700)]" />
                    ) : (
                      <Server className="h-4 w-4 text-[var(--ink-500)]" />
                    )}
                    <span className="text-sm text-[var(--ink-700)]">
                      {record.incomingApi.toUpperCase()} · {record.selectedModel || record.incomingModel}
                    </span>
                  </div>
                  <span className="text-xs text-[var(--ink-500)]">{record.durationMs}ms</span>
                </div>
              ))}

              {!recordsLoading && (usageRecords?.data || []).length === 0 && (
                <p className="text-sm text-[var(--ink-500)]">No recent telemetry events in selected window.</p>
              )}
            </div>
          </Card>
        </div>
      </div>
    </AppLayout>
  );
}
