'use client';

import { useEffect, useState } from 'react';
import { AppLayout } from '@/components/templates/AppLayout';
import { Card } from '@/components/atoms/Card';
import { StatusBadge } from '@/components/molecules/StatusBadge';
import { Activity, Server, Zap, AlertCircle } from 'lucide-react';

interface LiveMetric {
  label: string;
  value: string;
  change: number;
  trend: 'up' | 'down' | 'neutral';
}

interface ProviderHealth {
  id: string;
  name: string;
  status: 'healthy' | 'degraded' | 'unhealthy';
  circuitState: 'closed' | 'open' | 'half-open';
  latency: number;
  requestsPerMinute: number;
}

export default function ControlRoomsPage() {
  const [metrics, setMetrics] = useState<LiveMetric[]>([
    { label: 'Requests/min', value: '1,234', change: 12, trend: 'up' },
    { label: 'Avg Latency', value: '145ms', change: -5, trend: 'down' },
    { label: 'Error Rate', value: '0.8%', change: 0, trend: 'neutral' },
    { label: 'Active Conns', value: '456', change: 23, trend: 'up' },
  ]);

  const [providers] = useState<ProviderHealth[]>([
    { id: '1', name: 'OpenAI', status: 'healthy', circuitState: 'closed', latency: 120, requestsPerMinute: 523 },
    { id: '2', name: 'Anthropic', status: 'healthy', circuitState: 'closed', latency: 150, requestsPerMinute: 412 },
    { id: '3', name: 'Gemini', status: 'degraded', circuitState: 'closed', latency: 200, requestsPerMinute: 299 },
  ]);

  useEffect(() => {
    const interval = setInterval(() => {
      setMetrics(prev => prev.map(m => ({
        ...m,
        value: Math.random() > 0.5 
          ? (parseInt(m.value.replace(/[^0-9]/g, '')) + Math.floor(Math.random() * 100)).toLocaleString()
          : m.value
      })));
    }, 3000);

    return () => clearInterval(interval);
  }, []);

  return (
    <AppLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Control Rooms</h1>
          <p className="text-gray-500">Real-time monitoring and operations</p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {metrics.map((metric) => (
            <Card key={metric.label}>
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-gray-500">{metric.label}</p>
                  <p className="text-2xl font-bold mt-1">{metric.value}</p>
                </div>
                <div className={`flex items-center text-sm ${
                  metric.trend === 'up' ? 'text-green-600' : 
                  metric.trend === 'down' ? 'text-red-600' : 'text-gray-500'
                }`}>
                  {metric.change > 0 && '+'}{metric.change}%
                </div>
              </div>
            </Card>
          ))}
        </div>

        <Card title="Provider Health">
          <div className="space-y-4">
            {providers.map((provider) => (
              <div key={provider.id} className="flex items-center justify-between py-3 border-b border-gray-100 last:border-0">
                <div className="flex items-center gap-3">
                  <div className={`w-3 h-3 rounded-full ${
                    provider.status === 'healthy' ? 'bg-green-500' :
                    provider.status === 'degraded' ? 'bg-yellow-500' : 'bg-red-500'
                  }`}></div>
                  <div>
                    <p className="font-medium">{provider.name}</p>
                    <div className="flex items-center gap-2 text-sm text-gray-500">
                      <span>{provider.latency}ms latency</span>
                      <span>•</span>
                      <span>{provider.requestsPerMinute} req/min</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <StatusBadge status={provider.status} showPulse={provider.status === 'healthy'} />
                  <StatusBadge status={provider.circuitState} />
                </div>
              </div>
            ))}
          </div>
        </Card>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card title="Recent Alerts">
            <div className="space-y-3">
              {[
                { level: 'warning', message: 'Gemini latency above threshold', time: '2m ago' },
                { level: 'info', message: 'Circuit breaker opened for test-provider', time: '15m ago' },
                { level: 'success', message: 'OpenAI health check passed', time: '1h ago' },
              ].map((alert, i) => (
                <div key={i} className="flex items-start gap-3 p-3 bg-gray-50 rounded-lg">
                  <AlertCircle className={`w-5 h-5 ${
                    alert.level === 'warning' ? 'text-yellow-500' :
                    alert.level === 'success' ? 'text-green-500' : 'text-blue-500'
                  }`} />
                  <div className="flex-1">
                    <p className="text-sm font-medium">{alert.message}</p>
                    <p className="text-xs text-gray-500">{alert.time}</p>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          <Card title="System Events">
            <div className="space-y-3">
              {[
                { event: 'API request processed', count: '12,453', icon: Activity },
                { event: 'Provider health check', count: '3/3 healthy', icon: Server },
                { event: 'Tokens processed', count: '2.4M', icon: Zap },
              ].map((item, i) => (
                <div key={i} className="flex items-center justify-between py-2">
                  <div className="flex items-center gap-3">
                    <item.icon className="w-5 h-5 text-gray-400" />
                    <span className="text-sm text-gray-700">{item.event}</span>
                  </div>
                  <span className="text-sm font-medium text-gray-900">{item.count}</span>
                </div>
              ))}
            </div>
          </Card>
        </div>
      </div>
    </AppLayout>
  );
}
