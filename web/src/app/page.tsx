'use client';

import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { MetricCard } from '@/components/dashboard/MetricCard';
import { Button } from '@/components/atoms/Button';
import { useRouter } from 'next/navigation';
import { Server, Key, Activity, DollarSign, Plus } from 'lucide-react';

export default function DashboardPage() {
  const router = useRouter();

  return (
    <AppLayout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
            <p className="text-gray-500">Overview of your AI gateway</p>
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
          value="3"
          icon={<Server className="w-5 h-5" />}
          change={{ value: 12, positive: true }}
        />
        <MetricCard
          title="API Calls Today"
          value="12,543"
          icon={<Activity className="w-5 h-5" />}
          change={{ value: 8, positive: true }}
        />
        <MetricCard
          title="Active API Keys"
          value="8"
          icon={<Key className="w-5 h-5" />}
        />
        <MetricCard
          title="Cost Today"
          value="$124.50"
          icon={<DollarSign className="w-5 h-5" />}
          change={{ value: 5, positive: false }}
        />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card title="System Health">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">API Gateway</span>
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  Healthy
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">Database</span>
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  Connected
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-gray-600">Rate Limiting</span>
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  Active
                </span>
              </div>
            </div>
          </Card>

          <Card title="Recent Activity">
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="flex items-center gap-3 text-sm">
                  <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                  <span className="text-gray-600 flex-1">API request processed</span>
                  <span className="text-gray-400">{i * 5}m ago</span>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </AppLayout>
  );
}
