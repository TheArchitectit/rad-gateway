'use client';

import { useMemo } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { MetricCard } from '@/components/dashboard/MetricCard';
import { Button } from '@/components/atoms/Button';
import { StatusBadge } from '@/components/molecules/StatusBadge';
import { useRouter } from 'next/navigation';
import { AlertTriangle, Activity, DollarSign, Gauge, Key, Plus, Server } from 'lucide-react';
import { useAPIKeys, useProjects, useProviders, useUsage, useUsageSummary } from '@/queries';

function isoHoursAgo(hours: number): string {
  return new Date(Date.now() - hours * 60 * 60 * 1000).toISOString();
}

export default function DashboardPage() {
  const router = useRouter();
  const range = useMemo(
    () => ({
      startTime: isoHoursAgo(24),
      endTime: new Date().toISOString(),
    }),
    []
  );

  const { data: providersData, isLoading: providersLoading } = useProviders();
  const { data: apiKeysData, isLoading: apiKeysLoading } = useAPIKeys({ page: 1, pageSize: 200 });
  const { data: projectsData, isLoading: projectsLoading } = useProjects({ page: 1, pageSize: 200 });
  const { data: usageSummary, isLoading: usageSummaryLoading } = useUsageSummary(range);
  const { data: usageData, isLoading: usageLoading } = useUsage({ ...range, page: 1, pageSize: 6 });

  const providers = providersData?.providers || [];
  const apiKeys = apiKeysData?.data || [];
  const projects = projectsData?.data || [];
  const recentRecords = usageData?.data || [];

  const activeProviders = providers.filter((provider) => provider.status !== 'disabled').length;
  const healthyProviders = providers.filter((provider) => provider.status === 'healthy').length;
  const activeKeys = apiKeys.filter((apiKey) => apiKey.status === 'active').length;
  const openIncidents = providers.filter((provider) => provider.status === 'unhealthy').length;

  const summaryLoading = providersLoading || apiKeysLoading || projectsLoading || usageSummaryLoading;

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-3xl font-bold text-[var(--ink-900)]">Operations Dashboard</h1>
            <p className="text-[var(--ink-500)]">
              Live command view for providers, usage flow, and reliability posture.
            </p>
          </div>
          <div className="flex gap-3">
            <Button onClick={() => router.push('/api-keys/new')}>
              <Plus className="w-4 h-4 mr-2" />
              New API Key
            </Button>
            <Button variant="secondary" onClick={() => router.push('/providers/new')}>
              <Plus className="w-4 h-4 mr-2" />
              Add Provider
            </Button>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <MetricCard
            title="Active Providers"
            value={activeProviders}
            icon={<Server className="w-5 h-5" />}
            isLoading={summaryLoading}
            {...(providers.length > 0
              ? {
                  change: {
                    value: Math.round((healthyProviders / providers.length) * 100),
                    positive: healthyProviders === providers.length,
                  },
                }
              : {})}
            description="Healthy capacity over configured endpoints"
          />
          <MetricCard
            title="API Calls (24h)"
            value={(usageSummary?.totalRequests || 0).toLocaleString()}
            icon={<Activity className="w-5 h-5" />}
            isLoading={usageSummaryLoading}
            description="From usage summary window"
          />
          <MetricCard
            title="Active API Keys"
            value={activeKeys}
            icon={<Key className="w-5 h-5" />}
            isLoading={apiKeysLoading}
            description={`${apiKeys.length} total keys across workspaces`}
          />
          <MetricCard
            title="Cost (24h)"
            value={`$${(usageSummary?.totalCostUsd || 0).toFixed(2)}`}
            icon={<DollarSign className="w-5 h-5" />}
            isLoading={usageSummaryLoading}
            description="Aggregated spend in current range"
          />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card
            title="System Health"
            header={
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-[var(--ink-900)]">System Health</h3>
                {openIncidents > 0 ? (
                  <span className="inline-flex items-center gap-1 rounded-full border border-[rgba(152,43,33,0.35)] bg-[rgba(152,43,33,0.15)] px-2 py-0.5 text-xs font-semibold text-[var(--status-critical)]">
                    <AlertTriangle className="h-3.5 w-3.5" />
                    {openIncidents} Incident{openIncidents === 1 ? '' : 's'}
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1 rounded-full border border-[rgba(47,122,79,0.35)] bg-[rgba(47,122,79,0.15)] px-2 py-0.5 text-xs font-semibold text-[var(--status-normal)]">
                    <Gauge className="h-3.5 w-3.5" />
                    Stable
                  </span>
                )}
              </div>
            }
          >
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-[var(--ink-700)]">Gateway Runtime</span>
                <StatusBadge status={openIncidents > 0 ? 'degraded' : 'healthy'} />
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-[var(--ink-700)]">Workspace Inventory</span>
                <span className="text-sm font-semibold text-[var(--ink-900)]">
                  {projectsLoading ? '...' : `${projects.length} active projects`}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-[var(--ink-700)]">Provider Reliability</span>
                <span className="text-sm font-semibold text-[var(--ink-900)]">
                  {providers.length === 0 ? 'No providers' : `${healthyProviders}/${providers.length} healthy`}
                </span>
              </div>
              <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] p-3 text-xs text-[var(--ink-500)]">
                Health model follows Foglight-style severity discipline: healthy, degraded, unhealthy, disabled.
              </div>
            </div>
          </Card>

          <Card title="Recent Activity">
            <div className="space-y-3">
              {usageLoading && (
                <p className="text-sm text-[var(--ink-500)]">Loading recent usage events...</p>
              )}

              {!usageLoading && recentRecords.length === 0 && (
                <p className="text-sm text-[var(--ink-500)]">No recent usage records yet.</p>
              )}

              {recentRecords.slice(0, 5).map((record) => (
                <div key={record.id} className="flex items-center gap-3 text-sm">
                  <div className="h-2 w-2 rounded-full bg-[var(--brass-500)]" />
                  <span className="flex-1 text-[var(--ink-700)]">
                    {record.incomingApi.toUpperCase()} request on {record.incomingModel}
                  </span>
                  <span className="text-[var(--ink-500)]">{record.durationMs}ms</span>
                </div>
              ))}
            </div>
          </Card>
        </div>

        <Card title="Provider Fleet">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {providers.length === 0 && <p className="text-sm text-[var(--ink-500)]">No providers registered.</p>}

            {providers.map((provider) => (
              <div
                key={provider.id}
                className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] p-3"
              >
                <div className="mb-2 flex items-center justify-between gap-3">
                  <p className="font-semibold text-[var(--ink-900)]">{provider.displayName || provider.name}</p>
                  <StatusBadge status={provider.status} />
                </div>
                <p className="text-xs text-[var(--ink-500)]">
                  Circuit: {provider.circuitBreaker} · Latency: {provider.latencyMs ? `${provider.latencyMs}ms` : 'n/a'}
                </p>
              </div>
            ))}
          </div>
        </Card>
      </div>
    </AppLayout>
  );
}
