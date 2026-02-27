/**
 * Token Bar Chart Component
 * Sprint 6.1: Recharts Integration
 *
 * Bar chart for token usage breakdown (input/output/reasoning)
 */

'use client';

import React from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { Card } from '@/components/atoms/Card';

interface TokenDataPoint {
  timestamp: string;
  input: number;
  output: number;
  reasoning?: number;
  cached?: number;
}

interface TokenBarChartProps {
  data: TokenDataPoint[];
  title?: string;
  className?: string;
}

export const TokenBarChart: React.FC<TokenBarChartProps> = ({
  data,
  title = 'Token Usage',
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
            <BarChart data={data} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
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
                tickFormatter={(value) => `${(value / 1000).toFixed(0)}k`}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--panel)',
                  border: '1px solid var(--line-soft)',
                  borderRadius: '8px',
                }}
                labelFormatter={(label) => formatTime(label as string)}
              />
              <Legend />
              <Bar
                dataKey="input"
                stackId="tokens"
                fill="#3B82F6"
                name="Input Tokens"
                radius={[0, 0, 4, 4]}
              />
              <Bar
                dataKey="output"
                stackId="tokens"
                fill="#10B981"
                name="Output Tokens"
              />
              <Bar
                dataKey="reasoning"
                stackId="tokens"
                fill="#8B5CF6"
                name="Reasoning Tokens"
              />
              <Bar
                dataKey="cached"
                stackId="tokens"
                fill="#F59E0B"
                name="Cached Tokens"
                radius={[4, 4, 0, 0]}
              />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </Card>
  );
};

export default TokenBarChart;
