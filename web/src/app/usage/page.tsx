'use client';

import { useState } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { Button } from '@/components/atoms/Button';
import { Badge } from '@/components/atoms/Badge';
import { Download } from 'lucide-react';

const timeRanges = [
  { value: '24h', label: 'Last 24 hours' },
  { value: '7d', label: 'Last 7 days' },
  { value: '30d', label: 'Last 30 days' },
];

export default function UsagePage() {
  const [timeRange, setTimeRange] = useState('24h');

  const mockData = {
    totalRequests: 12453,
    totalTokens: 2458000,
    totalCost: 124.50,
    avgLatency: '145ms',
    successRate: '99.2%',
  };

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Usage</h1>
            <p className="text-gray-500">View your API usage and costs</p>
          </div>
          <div className="flex gap-3">
            <select
              value={timeRange}
              onChange={(e) => setTimeRange(e.target.value)}
              className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {timeRanges.map((range) => (
                <option key={range.value} value={range.value}>
                  {range.label}
                </option>
              ))}
            </select>
            <Button variant="secondary">
              <Download className="w-4 h-4 mr-2" />
              Export
            </Button>
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
          <Card>
            <div className="text-sm text-gray-500">Total Requests</div>
            <div className="text-2xl font-bold mt-1">{mockData.totalRequests.toLocaleString()}</div>
          </Card>
          <Card>
            <div className="text-sm text-gray-500">Total Tokens</div>
            <div className="text-2xl font-bold mt-1">{(mockData.totalTokens / 1000000).toFixed(2)}M</div>
          </Card>
          <Card>
            <div className="text-sm text-gray-500">Total Cost</div>
            <div className="text-2xl font-bold mt-1">${mockData.totalCost.toFixed(2)}</div>
          </Card>
          <Card>
            <div className="text-sm text-gray-500">Avg Latency</div>
            <div className="text-2xl font-bold mt-1">{mockData.avgLatency}</div>
          </Card>
          <Card>
            <div className="text-sm text-gray-500">Success Rate</div>
            <div className="text-2xl font-bold mt-1 text-green-600">{mockData.successRate}</div>
          </Card>
        </div>

        <Card title="Usage by Provider">
          <div className="space-y-4">
            {[
              { name: 'OpenAI', requests: 8234, tokens: 1650000, cost: 82.34 },
              { name: 'Anthropic', requests: 3219, tokens: 648000, cost: 32.19 },
              { name: 'Gemini', requests: 1000, tokens: 160000, cost: 9.97 },
            ].map((provider) => (
              <div key={provider.name} className="flex items-center justify-between py-3 border-b border-gray-100 last:border-0">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center text-blue-600 font-semibold">
                    {provider.name[0]}
                  </div>
                  <div>
                    <p className="font-medium">{provider.name}</p>
                    <p className="text-sm text-gray-500">{provider.requests.toLocaleString()} requests</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="font-medium">${provider.cost.toFixed(2)}</p>
                  <p className="text-sm text-gray-500">{(provider.tokens / 1000000).toFixed(2)}M tokens</p>
                </div>
              </div>
            ))}
          </div>
        </Card>

        <Card title="Recent Usage">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-3 px-4 text-sm font-medium text-gray-500">Time</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-gray-500">Provider</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-gray-500">Model</th>
                  <th className="text-right py-3 px-4 text-sm font-medium text-gray-500">Tokens</th>
                  <th className="text-right py-3 px-4 text-sm font-medium text-gray-500">Cost</th>
                  <th className="text-left py-3 px-4 text-sm font-medium text-gray-500">Status</th>
                </tr>
              </thead>
              <tbody>
                {[1, 2, 3, 4, 5].map((i) => (
                  <tr key={i} className="border-b border-gray-100 last:border-0">
                    <td className="py-3 px-4 text-sm text-gray-900">{i * 5}m ago</td>
                    <td className="py-3 px-4 text-sm text-gray-900">OpenAI</td>
                    <td className="py-3 px-4 text-sm text-gray-900">gpt-4</td>
                    <td className="py-3 px-4 text-sm text-gray-900 text-right">1,234</td>
                    <td className="py-3 px-4 text-sm text-gray-900 text-right">$0.024</td>
                    <td className="py-3 px-4">
                      <Badge color="success">Success</Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </div>
    </AppLayout>
  );
}
