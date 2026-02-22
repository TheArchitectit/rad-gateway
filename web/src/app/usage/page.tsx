'use client';

import { useMemo, useState } from 'react';
import { Download } from 'lucide-react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';
import { Badge } from '@/components/atoms/Badge';
import {
  useCreateExport,
  useExportStatus,
  useUsage,
  useUsageByDimension,
  useUsageSummary,
} from '@/queries';

const timeRanges = [
  { value: '24h', label: 'Last 24 hours', hours: 24 },
  { value: '7d', label: 'Last 7 days', hours: 24 * 7 },
  { value: '30d', label: 'Last 30 days', hours: 24 * 30 },
] as const;

type TimeRange = (typeof timeRanges)[number]['value'];

function getTimeWindow(range: TimeRange): { startTime: string; endTime: string } {
  const selected = timeRanges.find((item) => item.value === range) || timeRanges[0];
  const end = new Date();
  const start = new Date(end.getTime() - selected.hours * 60 * 60 * 1000);
  return {
    startTime: start.toISOString(),
    endTime: end.toISOString(),
  };
}

export default function UsagePage() {
  const [timeRange, setTimeRange] = useState<TimeRange>('24h');
  const [activeExportId, setActiveExportId] = useState<string | undefined>(undefined);

  const window = useMemo(() => getTimeWindow(timeRange), [timeRange]);

  const {
    data: summary,
    isLoading: summaryLoading,
    error: summaryError,
  } = useUsageSummary(window);
  const {
    data: usageList,
    isLoading: usageLoading,
    error: usageError,
  } = useUsage({ ...window, page: 1, pageSize: 25 });

  const providerDimension = useUsageByDimension('providerId', window);
  const createExport = useCreateExport({
    onSuccess: (response) => setActiveExportId(response.exportId),
  });
  const exportStatus = useExportStatus(activeExportId);

  const providerRows = useMemo(() => {
    const grouped = providerDimension.data;
    const entries = Object.entries(grouped).map(([providerName, metrics]) => ({
      providerName,
      requestCount: Number(metrics['requestCount'] || 0),
      totalTokens: Number(metrics['totalTokens'] || 0),
      costUsd: Number(metrics['costUsd'] || 0),
    }));

    if (entries.length > 0) {
      return entries.sort((a, b) => b.requestCount - a.requestCount);
    }

    const fallbackMap = new Map<string, { requestCount: number; totalTokens: number; costUsd: number }>();
    (usageList?.data || []).forEach((record) => {
      const key = record.providerId || 'unknown';
      const current = fallbackMap.get(key) || { requestCount: 0, totalTokens: 0, costUsd: 0 };
      current.requestCount += 1;
      current.totalTokens += record.totalTokens;
      current.costUsd += record.costUsd || 0;
      fallbackMap.set(key, current);
    });

    return Array.from(fallbackMap.entries()).map(([providerName, metrics]) => ({
      providerName,
      ...metrics,
    }));
  }, [providerDimension.data, usageList?.data]);

  const hasError = summaryError || usageError;
  const listRecords = usageList?.data || [];

  const triggerExport = async (format: 'json' | 'csv') => {
    await createExport.mutateAsync({
      startTime: window.startTime,
      endTime: window.endTime,
      format,
      includeCost: true,
    });
  };

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex flex-col justify-between gap-3 md:flex-row md:items-center">
          <div>
            <h1 className="text-3xl font-bold text-[var(--ink-900)]">Usage Command Deck</h1>
            <p className="text-[var(--ink-500)]">Operational usage intelligence with export-ready telemetry.</p>
          </div>
          <div className="flex flex-wrap gap-3">
            <select
              value={timeRange}
              onChange={(event) => setTimeRange(event.target.value as TimeRange)}
              className="ui-input max-w-[180px]"
            >
              {timeRanges.map((range) => (
                <option key={range.value} value={range.value}>
                  {range.label}
                </option>
              ))}
            </select>
            <Button
              variant="secondary"
              onClick={() => void triggerExport('csv')}
              loading={createExport.isPending}
            >
              <Download className="mr-2 h-4 w-4" />
              Export CSV
            </Button>
          </div>
        </div>

        {hasError && (
          <Card className="border border-[rgba(152,43,33,0.3)] bg-[rgba(152,43,33,0.08)]">
            <p className="text-sm text-[var(--status-critical)]">
              {(summaryError || usageError)?.message || 'Failed to load usage telemetry.'}
            </p>
          </Card>
        )}

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-5">
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Total Requests</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {summaryLoading ? '...' : (summary?.totalRequests || 0).toLocaleString()}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Total Tokens</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {summaryLoading ? '...' : `${((summary?.totalTokens || 0) / 1_000_000).toFixed(2)}M`}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Total Cost</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {summaryLoading ? '...' : `$${(summary?.totalCostUsd || 0).toFixed(2)}`}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Avg Latency</div>
            <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">
              {summaryLoading ? '...' : `${Math.round(summary?.avgDurationMs || 0)}ms`}
            </div>
          </Card>
          <Card>
            <div className="text-sm text-[var(--ink-500)]">Error Rate</div>
            <div className="mt-1 text-2xl font-bold text-[var(--status-critical)]">
              {summaryLoading ? '...' : `${(summary?.errorRate || 0).toFixed(2)}%`}
            </div>
          </Card>
        </div>

        <Card title="Usage by Provider">
          <div className="space-y-4">
            {providerRows.length === 0 && !usageLoading && (
              <p className="text-sm text-[var(--ink-500)]">No provider usage data available for this range.</p>
            )}

            {providerRows.map((provider) => (
              <div
                key={provider.providerName}
                className="flex items-center justify-between rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.06)] px-4 py-3"
              >
                <div>
                  <p className="font-medium text-[var(--ink-900)]">{provider.providerName}</p>
                  <p className="text-sm text-[var(--ink-500)]">
                    {provider.requestCount.toLocaleString()} requests · {(provider.totalTokens / 1_000_000).toFixed(2)}M tokens
                  </p>
                </div>
                <p className="font-semibold text-[var(--ink-900)]">${provider.costUsd.toFixed(2)}</p>
              </div>
            ))}
          </div>
        </Card>

        <Card title="Recent Usage Records">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-[var(--line-strong)]">
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Started</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Provider</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Model</th>
                  <th className="px-4 py-3 text-right text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Tokens</th>
                  <th className="px-4 py-3 text-right text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Cost</th>
                  <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--line-soft)]">
                {usageLoading && (
                  <tr>
                    <td colSpan={6} className="px-4 py-6 text-sm text-[var(--ink-500)]">
                      Loading usage records...
                    </td>
                  </tr>
                )}

                {!usageLoading && listRecords.length === 0 && (
                  <tr>
                    <td colSpan={6} className="px-4 py-6 text-sm text-[var(--ink-500)]">
                      No usage records in this range.
                    </td>
                  </tr>
                )}

                {listRecords.map((record) => (
                  <tr key={record.id} className="hover:bg-[rgba(43,32,21,0.05)]">
                    <td className="px-4 py-3 text-sm text-[var(--ink-900)]">
                      {new Date(record.startedAt).toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-sm text-[var(--ink-900)]">{record.providerId || 'unknown'}</td>
                    <td className="px-4 py-3 text-sm text-[var(--ink-900)]">{record.selectedModel || record.incomingModel}</td>
                    <td className="px-4 py-3 text-right text-sm text-[var(--ink-900)]">
                      {record.totalTokens.toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-right text-sm text-[var(--ink-900)]">
                      ${(record.costUsd || 0).toFixed(4)}
                    </td>
                    <td className="px-4 py-3">
                      <Badge color={record.responseStatus === 'success' ? 'success' : 'error'}>
                        {record.responseStatus}
                      </Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>

        {(createExport.data || exportStatus.data) && (
          <Card title="Export Status">
            <div className="space-y-2 text-sm text-[var(--ink-700)]">
              <p>
                Export ID: <span className="font-semibold text-[var(--ink-900)]">{(exportStatus.data || createExport.data)?.exportId}</span>
              </p>
              <p>
                Status: <span className="font-semibold text-[var(--ink-900)]">{(exportStatus.data || createExport.data)?.status}</span>
              </p>
              {(exportStatus.data || createExport.data)?.downloadUrl && (
                <a
                  href={(exportStatus.data || createExport.data)?.downloadUrl}
                  className="text-[var(--status-info)] underline-offset-2 hover:underline"
                >
                  Download export payload
                </a>
              )}
            </div>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
