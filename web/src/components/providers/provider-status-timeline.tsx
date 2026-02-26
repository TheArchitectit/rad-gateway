/**
 * Provider Status Timeline Component
 * Task 1.6: Provider Health Dashboard
 * Mini sparkline chart showing 24h uptime with Art Deco styling
 */

'use client';

import React, { useState } from 'react';
import { cn } from '@/lib/utils';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import type { ProviderStatus } from '@/types';

interface TimelineDataPoint {
  timestamp: string;
  value: number; // 0-100 representing uptime percentage
}

interface ProviderStatusTimelineProps {
  data: TimelineDataPoint[];
  status: ProviderStatus;
  width?: number;
  height?: number;
  className?: string;
}

// Generate smooth path for sparkline using cubic bezier curves
const generateSmoothPath = (points: { x: number; y: number }[]): string => {
  if (points.length === 0) return '';
  const firstPoint = points[0]!;
  if (points.length === 1) return 'M ' + firstPoint.x + ' ' + firstPoint.y;

  let path = 'M ' + firstPoint.x + ' ' + firstPoint.y;

  for (let i = 0; i < points.length - 1; i++) {
    const current = points[i]!;
    const next = points[i + 1]!;
    
    // Control points for smooth curve
    const cp1x = current.x + (next.x - current.x) * 0.3;
    const cp1y = current.y;
    const cp2x = current.x + (next.x - current.x) * 0.7;
    const cp2y = next.y;
    
    path += ' C ' + cp1x + ' ' + cp1y + ', ' + cp2x + ' ' + cp2y + ', ' + next.x + ' ' + next.y;
  }

  return path;
};

// Generate area path for gradient fill
const generateAreaPath = (points: { x: number; y: number }[], height: number): string => {
  const linePath = generateSmoothPath(points);
  if (!linePath) return '';
  
  const lastPoint = points[points.length - 1]!;
  const firstPoint = points[0]!;
  
  return linePath + ' L ' + lastPoint.x + ' ' + height + ' L ' + firstPoint.x + ' ' + height + ' Z';
};

export const ProviderStatusTimeline: React.FC<ProviderStatusTimelineProps> = ({
  data,
  status,
  width = 280,
  height = 40,
  className,
}) => {
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);

  // Get color based on status
  const getStatusColor = (s: ProviderStatus): string => {
    switch (s) {
      case 'healthy':
        return '#10B981'; // emerald-500
      case 'degraded':
        return '#F59E0B'; // amber-500
      case 'unhealthy':
        return '#EF4444'; // red-500
      case 'disabled':
        return '#94A3B8'; // slate-400
      default:
        return '#B57D41'; // brass
    }
  };

  // Get gradient colors
  const getGradientColors = (s: ProviderStatus): { start: string; end: string } => {
    switch (s) {
      case 'healthy':
        return { start: 'rgba(16, 185, 129, 0.3)', end: 'rgba(16, 185, 129, 0.05)' };
      case 'degraded':
        return { start: 'rgba(245, 158, 11, 0.3)', end: 'rgba(245, 158, 11, 0.05)' };
      case 'unhealthy':
        return { start: 'rgba(239, 68, 68, 0.3)', end: 'rgba(239, 68, 68, 0.05)' };
      case 'disabled':
        return { start: 'rgba(148, 163, 184, 0.2)', end: 'rgba(148, 163, 184, 0.02)' };
      default:
        return { start: 'rgba(181, 125, 65, 0.3)', end: 'rgba(181, 125, 65, 0.05)' };
    }
  };

  const strokeColor = getStatusColor(status);
  const gradientColors = getGradientColors(status);

  // Calculate points
  const padding = { top: 4, right: 4, bottom: 4, left: 4 };
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  const points = data.map((point, index) => {
    const x = padding.left + (index / (data.length - 1)) * chartWidth;
    const y = padding.top + chartHeight - (point.value / 100) * chartHeight;
    return { x, y, data: point };
  });

  const pathD = generateSmoothPath(points);
  const areaD = generateAreaPath(points, height - padding.bottom);

  // Calculate average for reference line
  const average = data.reduce((sum, d) => sum + d.value, 0) / data.length;
  const avgY = padding.top + chartHeight - (average / 100) * chartHeight;

  return (
    <TooltipProvider>
      <div className={cn('relative', className)}>
        <svg
          width={width}
          height={height}
          className="overflow-visible"
          onMouseLeave={() => setHoveredIndex(null)}
        >
          <defs>
            {/* Gradient for area fill */}
            <linearGradient id={`area-gradient-${status}`} x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" stopColor={gradientColors.start} />
              <stop offset="100%" stopColor={gradientColors.end} />
            </linearGradient>

            {/* Glow filter for Art Deco effect */}
            <filter id={`glow-${status}`} x="-20%" y="-20%" width="140%" height="140%">
              <feGaussianBlur stdDeviation="1.5" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
          </defs>

          {/* Background grid lines (Art Deco style) */}
          {[0, 25, 50, 75, 100].map((pct) => {
            const y = padding.top + chartHeight - (pct / 100) * chartHeight;
            return (
              <line
                key={pct}
                x1={padding.left}
                y1={y}
                x2={width - padding.right}
                y2={y}
                stroke="#E2E8F0"
                strokeWidth={0.5}
                strokeDasharray={pct === 50 ? '0' : '2,2'}
                opacity={pct === 50 ? 0.5 : 0.3}
              />
            );
          })}

          {/* Average reference line */}
          <line
            x1={padding.left}
            y1={avgY}
            x2={width - padding.right}
            y2={avgY}
            stroke={strokeColor}
            strokeWidth={1}
            strokeDasharray="4,4"
            opacity={0.4}
          />

          {/* Area fill */}
          {areaD && (
            <path
              d={areaD}
              fill={`url(#area-gradient-${status})`}
              className="transition-all duration-300"
            />
          )}

          {/* Sparkline path */}
          {pathD && (
            <path
              d={pathD}
              fill="none"
              stroke={strokeColor}
              strokeWidth={2}
              strokeLinecap="round"
              strokeLinejoin="round"
              filter={`url(#glow-${status})`}
              className="transition-all duration-300"
            />
          )}

          {/* Data points */}
          {points.map((point, index) => (
            <g key={index}>
              <circle
                cx={point.x}
                cy={point.y}
                r={hoveredIndex === index ? 4 : 2}
                fill={status === 'healthy' ? '#10B981' : strokeColor}
                stroke="#fff"
                strokeWidth={1.5}
                className="transition-all duration-200 cursor-pointer"
                onMouseEnter={() => setHoveredIndex(index)}
              />
              
              {/* Invisible hit area for easier hovering */}
              <rect
                x={point.x - (chartWidth / data.length / 2)}
                y={0}
                width={chartWidth / data.length}
                height={height}
                fill="transparent"
                className="cursor-pointer"
                onMouseEnter={() => setHoveredIndex(index)}
              />
            </g>
          ))}

          {/* Hover indicator line */}
          {hoveredIndex !== null && (
            <line
              x1={points[hoveredIndex]!.x}
              y1={padding.top}
              x2={points[hoveredIndex]!.x}
              y2={height - padding.bottom}
              stroke={strokeColor}
              strokeWidth={1}
              strokeDasharray="2,2"
              opacity={0.5}
            />
          )}
        </svg>

        {/* Tooltip for hovered point */}
        {hoveredIndex !== null && (
          <div
            className="absolute top-0 pointer-events-none z-10"
            style={{
              left: ((points[hoveredIndex]!.x / width) * 100) + '%',
              transform: 'translateX(-50%)',
            }}
          >
            <Tooltip open={true}>
              <TooltipTrigger asChild>
                <div className="w-1 h-1" />
              </TooltipTrigger>
              <TooltipContent
                side="top"
                className="bg-[#1E293B] text-white border-[#B57D41]/30"
              >
                <div className="text-xs">
                  <div className="font-medium">
                    {new Date(points[hoveredIndex]!.data.timestamp).toLocaleTimeString([], {
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </div>
                  <div className="text-emerald-400">
                    {points[hoveredIndex]!.data.value.toFixed(1)}% uptime
                  </div>
                </div>
              </TooltipContent>
            </Tooltip>
          </div>
        )}

        {/* Legend */}
        <div className="flex items-center justify-between mt-1.5 text-[10px] text-[#7A7F99]">
          <span>24h ago</span>
          <span className="flex items-center gap-1">
            <span
              className="w-2 h-0.5 rounded-full"
              style={{ backgroundColor: strokeColor, opacity: 0.6 }}
            />
            avg: {average.toFixed(1)}%
          </span>
          <span>Now</span>
        </div>
      </div>
    </TooltipProvider>
  );
};

// Compact version for alerts section
export const ProviderStatusTimelineCompact: React.FC<ProviderStatusTimelineProps> = ({
  data,
  status,
  width = 120,
  height = 24,
  className,
}) => {
  const strokeColor = status === 'healthy' ? '#10B981' : 
                     status === 'degraded' ? '#F59E0B' : 
                     status === 'unhealthy' ? '#EF4444' : '#94A3B8';

  const padding = { top: 2, right: 2, bottom: 2, left: 2 };
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;

  const points = data.map((point, index) => {
    const x = padding.left + (index / (data.length - 1)) * chartWidth;
    const y = padding.top + chartHeight - (point.value / 100) * chartHeight;
    return { x, y };
  });

  const pathD = generateSmoothPath(points);

  return (
    <svg
      width={width}
      height={height}
      className={cn('overflow-visible', className)}
    >
      {pathD && (
        <path
          d={pathD}
          fill="none"
          stroke={strokeColor}
          strokeWidth={1.5}
          strokeLinecap="round"
          strokeLinejoin="round"
          opacity={0.8}
        />
      )}
    </svg>
  );
};

export default ProviderStatusTimeline;
