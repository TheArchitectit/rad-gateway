'use client';

import { useMemo, useState } from 'react';
import { FileDown, Loader2 } from 'lucide-react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';
import { apiClient } from '@/api/client';

interface UsageReportItem {
  requestId: string;
  workspaceId: string;
  incomingApi: string;
  incomingModel: string;
  selectedModel?: string;
  responseStatus: string;
  durationMs: number;
  totalTokens: number;
  costUsd: number;
  createdAt: string;
}

interface UsageReportResponse {
  reportType: 'usage';
  generatedAt: string;
  summary: {
    totalRequests: number;
    totalTokens: number;
    totalCostUsd: number;
    successRate: number;
  };
  items: UsageReportItem[];
}

interface PerformanceReportResponse {
  reportType: 'performance';
  generatedAt: string;
  metrics: {
    ttftMs: { p50: number; p95: number; p99: number };
    tokensPerSecond: { p50: number; p95: number; p99: number };
    latencyMs: { p50: number; p95: number; p99: number };
    errorRate: number;
  };
}

interface ExportResponse {
  exportId: string;
  status: string;
  format: string;
  downloadUrl: string;
  expiresAt: string;
}

function getDefaultWindow(): { startTime: string; endTime: string } {
  const end = new Date();
  const start = new Date(end.getTime() - 7 * 24 * 60 * 60 * 1000);
  return {
    startTime: start.toISOString(),
    endTime: end.toISOString(),
  };
}

function toISOOrFallback(raw: string, fallback: string): string {
  const parsed = new Date(raw);
  if (Number.isNaN(parsed.getTime())) {
    return fallback;
  }
  return parsed.toISOString();
}

export default function ReportsPage() {
  const defaultWindow = useMemo(() => getDefaultWindow(), []);
  const [workspaceId, setWorkspaceId] = useState('');
  const [startTime, setStartTime] = useState(defaultWindow.startTime.slice(0, 16));
  const [endTime, setEndTime] = useState(defaultWindow.endTime.slice(0, 16));

  const [loadingType, setLoadingType] = useState<'usage' | 'performance' | 'export' | null>(null);
  const [error, setError] = useState('');
  const [usageReport, setUsageReport] = useState<UsageReportResponse | null>(null);
  const [performanceReport, setPerformanceReport] = useState<PerformanceReportResponse | null>(null);
  const [lastExport, setLastExport] = useState<ExportResponse | null>(null);

  const queryParams = useMemo(() => {
    const params: Record<string, string> = {
      startTime: toISOOrFallback(startTime, defaultWindow.startTime),
      endTime: toISOOrFallback(endTime, defaultWindow.endTime),
    };
    if (workspaceId.trim()) {
      params['workspaceId'] = workspaceId.trim();
    }
    return params;
  }, [startTime, endTime, workspaceId]);

  const loadUsage = async () => {
    setLoadingType('usage');
    setError('');
    setPerformanceReport(null);

    try {
      const report = await apiClient.get<UsageReportResponse>('/v0/admin/reports/usage', {
        params: queryParams,
      });
      setUsageReport(report);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load usage report');
    } finally {
      setLoadingType(null);
    }
  };

  const loadPerformance = async () => {
    setLoadingType('performance');
    setError('');
    setUsageReport(null);

    try {
      const report = await apiClient.get<PerformanceReportResponse>('/v0/admin/reports/performance', {
        params: queryParams,
      });
      setPerformanceReport(report);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load performance report');
    } finally {
      setLoadingType(null);
    }
  };

  const exportCurrent = async (format: 'json' | 'csv') => {
    setLoadingType('export');
    setError('');

    try {
      const exported = await apiClient.post<ExportResponse>(
        '/v0/admin/reports/export',
        undefined,
        { params: { format } }
      );
      setLastExport(exported);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create report export');
    } finally {
      setLoadingType(null);
    }
  };

  return (
    <AppLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-[var(--ink-900)]">Advanced Reports</h1>
          <p className="text-[var(--ink-500)]">
            Generate usage and performance dossiers with export-ready snapshots.
          </p>
        </div>

        <Card>
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <div>
              <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                Workspace ID
              </label>
              <input
                value={workspaceId}
                onChange={(event) => setWorkspaceId(event.target.value)}
                placeholder="Optional workspace"
                className="ui-input"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                Start Time
              </label>
              <input
                type="datetime-local"
                value={startTime}
                onChange={(event) => setStartTime(event.target.value)}
                className="ui-input"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">
                End Time
              </label>
              <input
                type="datetime-local"
                value={endTime}
                onChange={(event) => setEndTime(event.target.value)}
                className="ui-input"
              />
            </div>
            <div className="flex flex-wrap items-end gap-2">
              <Button onClick={() => void loadUsage()} loading={loadingType === 'usage'}>
                Load Usage
              </Button>
              <Button variant="secondary" onClick={() => void loadPerformance()} loading={loadingType === 'performance'}>
                Load Performance
              </Button>
            </div>
          </div>
        </Card>

        <Card title="Exports">
          <div className="flex flex-wrap items-center gap-3">
            <Button
              variant="secondary"
              onClick={() => void exportCurrent('json')}
              loading={loadingType === 'export'}
            >
              <FileDown className="mr-2 h-4 w-4" />
              Export JSON
            </Button>
            <Button
              variant="secondary"
              onClick={() => void exportCurrent('csv')}
              loading={loadingType === 'export'}
            >
              <FileDown className="mr-2 h-4 w-4" />
              Export CSV
            </Button>

            {loadingType === 'export' && (
              <span className="inline-flex items-center gap-2 text-sm text-[var(--ink-500)]">
                <Loader2 className="h-4 w-4 animate-spin" />
                Preparing export...
              </span>
            )}

            {lastExport && (
              <div className="text-sm text-[var(--ink-700)]">
                Export <span className="font-semibold text-[var(--ink-900)]">{lastExport.exportId}</span> is {lastExport.status} ·{' '}
                <a className="text-[var(--status-info)] underline-offset-2 hover:underline" href={lastExport.downloadUrl}>
                  download
                </a>
              </div>
            )}
          </div>
        </Card>

        {error && (
          <Card className="border border-[rgba(152,43,33,0.3)] bg-[rgba(152,43,33,0.08)]">
            <p className="text-sm text-[var(--status-critical)]">{error}</p>
          </Card>
        )}

        {usageReport && (
          <>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
              <Card>
                <div className="text-sm text-[var(--ink-500)]">Requests</div>
                <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">{usageReport.summary.totalRequests.toLocaleString()}</div>
              </Card>
              <Card>
                <div className="text-sm text-[var(--ink-500)]">Tokens</div>
                <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">{usageReport.summary.totalTokens.toLocaleString()}</div>
              </Card>
              <Card>
                <div className="text-sm text-[var(--ink-500)]">Cost</div>
                <div className="mt-1 text-2xl font-bold text-[var(--ink-900)]">${usageReport.summary.totalCostUsd.toFixed(2)}</div>
              </Card>
              <Card>
                <div className="text-sm text-[var(--ink-500)]">Success Rate</div>
                <div className="mt-1 text-2xl font-bold text-[var(--status-normal)]">{(usageReport.summary.successRate * 100).toFixed(2)}%</div>
              </Card>
            </div>

            <Card title="Usage Report Records">
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-[var(--line-strong)]">
                      <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Request</th>
                      <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">API</th>
                      <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Model</th>
                      <th className="px-4 py-3 text-right text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Duration</th>
                      <th className="px-4 py-3 text-right text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Tokens</th>
                      <th className="px-4 py-3 text-right text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Cost</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[var(--line-soft)]">
                    {usageReport.items.slice(0, 20).map((item) => (
                      <tr key={item.requestId} className="hover:bg-[rgba(43,32,21,0.06)]">
                        <td className="px-4 py-3 text-sm text-[var(--ink-900)]">{item.requestId}</td>
                        <td className="px-4 py-3 text-sm text-[var(--ink-900)]">{item.incomingApi.toUpperCase()}</td>
                        <td className="px-4 py-3 text-sm text-[var(--ink-900)]">{item.selectedModel || item.incomingModel}</td>
                        <td className="px-4 py-3 text-right text-sm text-[var(--ink-900)]">{item.durationMs}ms</td>
                        <td className="px-4 py-3 text-right text-sm text-[var(--ink-900)]">{item.totalTokens.toLocaleString()}</td>
                        <td className="px-4 py-3 text-right text-sm text-[var(--ink-900)]">${item.costUsd.toFixed(4)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>
          </>
        )}

        {performanceReport && (
          <Card title="Performance Metrics">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
              <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.05)] p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">TTFT p95</p>
                <p className="mt-1 text-2xl font-semibold text-[var(--ink-900)]">{performanceReport.metrics.ttftMs.p95.toFixed(1)}ms</p>
              </div>
              <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.05)] p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Latency p95</p>
                <p className="mt-1 text-2xl font-semibold text-[var(--ink-900)]">{performanceReport.metrics.latencyMs.p95.toFixed(1)}ms</p>
              </div>
              <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.05)] p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Latency p99</p>
                <p className="mt-1 text-2xl font-semibold text-[var(--ink-900)]">{performanceReport.metrics.latencyMs.p99.toFixed(1)}ms</p>
              </div>
              <div className="rounded-lg border border-[var(--line-soft)] bg-[rgba(43,32,21,0.05)] p-4">
                <p className="text-xs font-semibold uppercase tracking-[0.1em] text-[var(--ink-500)]">Error Rate</p>
                <p className="mt-1 text-2xl font-semibold text-[var(--status-critical)]">
                  {(performanceReport.metrics.errorRate * 100).toFixed(2)}%
                </p>
              </div>
            </div>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
