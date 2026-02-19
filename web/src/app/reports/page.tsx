'use client';

import { useState } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';

export default function ReportsPage() {
  const [loading, setLoading] = useState(false);
  const [report, setReport] = useState('');
  const [error, setError] = useState('');

  const loadUsage = async () => {
    setLoading(true);
    setError('');
    setReport('');
    try {
      const response = await fetch('/v0/admin/reports/usage');
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }
      const data = await response.json();
      setReport(JSON.stringify(data, null, 2));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load report');
    } finally {
      setLoading(false);
    }
  };

  const loadPerformance = async () => {
    setLoading(true);
    setError('');
    setReport('');
    try {
      const response = await fetch('/v0/admin/reports/performance');
      if (!response.ok) {
        throw new Error(`Request failed (${response.status})`);
      }
      const data = await response.json();
      setReport(JSON.stringify(data, null, 2));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load report');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppLayout>
      <div className="space-y-6 max-w-5xl">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Advanced Reports</h1>
          <p className="text-gray-500">Usage and performance analytics endpoints</p>
        </div>

        <Card className="p-6 flex gap-3">
          <Button onClick={loadUsage} disabled={loading}>Load Usage Report</Button>
          <Button variant="secondary" onClick={loadPerformance} disabled={loading}>Load Performance Report</Button>
        </Card>

        {error && <Card className="p-4 border border-red-300 bg-red-50 text-red-700">{error}</Card>}

        {report && (
          <Card className="p-4">
            <pre className="bg-gray-100 rounded p-3 text-sm overflow-auto">{report}</pre>
          </Card>
        )}
      </div>
    </AppLayout>
  );
}
