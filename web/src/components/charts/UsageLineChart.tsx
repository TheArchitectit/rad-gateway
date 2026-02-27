/**
 * Usage Line Chart Component
 * Sprint 6.1: Recharts Integration
 *
 * Line chart for request volume over time
 */

'use client';

import React from 'react';
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Area,
  AreaChart,
} from 'recharts';
import { Card } from '@/components/atoms/Card';

interface DataPoint {
  timestamp: string;
  requests: number;
  errors?: number;
}

interface UsageLineChartProps {
  data: DataPoint[];
  title?: string;
  showErrors?: boolean;
  className?: string;
}

export const UsageLineChart: React.FC<UsageLineChartProps> = ({
  data,
  title = 'Requests Over Time',
  showErrors = true,
  className,
}) => {
  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };


  return (
    <Card className={className}>
      <div className="p-4">
        <h3 className="text-lg font-semibold text-[var(--ink-900)] mb-4">{title}</h3>
        <div className="h-[300px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
              <defs>
                <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#B18532" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#B18532" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="colorErrors" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#EF4444" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#EF4444" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--line-soft)" />
              <XAxis
                dataKey="timestamp"
                tickFormatter={formatTime}
                stroke="var(--ink-500)"
                tick={{ fill: 'var(--ink-500)', fontSize: 12 }}
              />
              <YAxis
                stroke="var(--ink-500)"
                tick={{ fill: 'var(--ink-500)', fontSize: 12 }}
                tickFormatter={(value) => value.toLocaleString()}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--panel)',
                  border: '1px solid var(--line-soft)',
                  borderRadius: '8px',
                }}
                labelFormatter={(label) => formatTime(label as string)}
              />
              <Area
                type="monotone"
                dataKey="requests"
                stroke="#B18532"
                strokeWidth={2}
                fillOpacity={1}
                fill="url(#colorRequests)"
                name="Requests"
              />
              {showErrors && (
                <Area
                  type="monotone"
                  dataKey="errors"
                  stroke="#EF4444"
                  strokeWidth={2}
                  fillOpacity={1}
                  fill="url(#colorErrors)"
                  name="Errors"
                />
              )}
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>
    </Card>
  );
};

export default UsageLineChart;
