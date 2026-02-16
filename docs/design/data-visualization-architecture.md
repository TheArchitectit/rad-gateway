# Brass Relay Data Visualization Architecture

## World-Class Steampunk Monitoring Dashboard

**Document Version**: 1.0
**Date**: 2026-02-16
**Status**: Technical Architecture Specification
**Author**: Frontend Architecture Team

---

## 1. Executive Summary

### Recommended Tech Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| **Rendering Engine** | SVG + CSS Animations | Optimal balance of performance, stylability, and accessibility for steampunk gauges |
| **Animation Library** | Framer Motion + Custom Spring Physics | Declarative React animations with realistic needle physics |
| **Real-Time Transport** | Server-Sent Events (SSE) | Unidirectional streaming perfect for metrics; auto-reconnect; lower overhead than WebSockets |
| **State Management** | Zustand + TanStack Query | Lightweight global state for UI; robust server-state caching |
| **Canvas Fallback** | HTML5 Canvas (Offscreen) | Particle effects (steam, smoke) via OffscreenCanvas for worker thread rendering |
| **Styling** | CSS Custom Properties + Tailwind | Design token system supporting dynamic theming |
| **Type Safety** | TypeScript (Strict Mode) | Component contracts and metric schemas |

### Architecture Philosophy

The Brass Relay dashboard employs a **hybrid rendering strategy**: SVG for crisp, scalable gauge components that demand precise styling control; Canvas (in workers) for performance-intensive particle effects like steam and smoke. This approach delivers:

- **60fps animations** across 20+ concurrent gauges
- **Full accessibility** support (screen readers, keyboard navigation)
- **Responsive design** from mobile (360px) to ultrawide (3440px)
- **Sub-100ms latency** from metric emission to visual update

---

## 2. Gauge Component Architecture

### 2.1 Component Hierarchy

```
BrassDashboard
├── ThemeProvider (CSS custom properties)
├── MetricsStreamProvider (SSE connection)
│
├── GaugeCluster
│   ├── PressureGauge (Provider Health)
│   │   ├── GaugeFace (SVG radial ticks)
│   │   ├── NeedleAssembly (Animated needle)
│   │   ├── SteamVent (Canvas particle layer)
│   │   └── ValueDisplay (Digital readout)
│   │
│   ├── ChronometerDial (Latency)
│   │   ├── ArcScale (Concentric rings)
│   │   ├── SweepHand (Rotating indicator)
│   │   └── DigitalWindow (Precise value)
│   │
│   ├── FuelMeter (Token Usage)
│   │   ├── VerticalScale (Linear gauge)
│   │   ├── PistonIndicator (Animated fill)
│   │   └── FlowRateDisplay (Rate of change)
│   │
│   └── BoilerPressure (System Health)
│       ├── CompoundGauge (Multi-needle)
│       ├── WarningLamp (Status indicator)
│       └── PressureHistory (Mini sparkline)
│
├── TelegraphTape (Live Request Stream)
│   ├── TapeWindow (Scrolling container)
│   ├── RequestChit (Individual request card)
│   └── FadeOverlay (Edge gradients)
│
└── GearAssembly (Decorative/Functional)
    ├── SpurGear (Rotating decoration)
    ├── WormGear (Counter-rotation)
    └── LinkageRod (Connecting animation)
```

### 2.2 Core Gauge Props Interface

```typescript
// types/gauges.ts

/**
 * Base properties shared by all gauge types
 */
interface BaseGaugeProps {
  /** Unique identifier for metric correlation */
  id: string;
  /** Display label - supports themed and plain text */
  label: string;
  labelTechnical?: string; // Plain-technical fallback
  /** Current value with unit */
  value: number;
  unit: string;
  /** Value range for scaling */
  min: number;
  max: number;
  /** Alert thresholds */
  warningThreshold?: number;
  criticalThreshold?: number;
  /** Animation configuration */
  animationConfig?: GaugeAnimationConfig;
  /** Accessibility */
  ariaLabel?: string;
  /** Event handlers */
  onThresholdCross?: (level: 'warning' | 'critical', value: number) => void;
  onClick?: () => void;
}

interface GaugeAnimationConfig {
  /** Spring tension (higher = snappier) */
  tension: number; // default: 120
  /** Spring friction (higher = more damping) */
  friction: number; // default: 14
  /** Mass of needle (affects momentum) */
  mass: number; // default: 1
  /** Maximum velocity cap for realism */
  maxVelocity: number; // default: 50
  /** Settle threshold (when animation considers complete) */
  precision: number; // default: 0.01
}

/**
 * Pressure Gauge: Provider health and availability
 * Metaphor: Steam pressure in industrial boiler
 */
interface PressureGaugeProps extends BaseGaugeProps {
  type: 'pressure';
  /** Number of major tick divisions */
  majorTicks: number;
  /** Number of minor ticks per major */
  minorTicks: number;
  /** Color zones (percentages of range) */
  zones: Array<{
    start: number;
    end: number;
    color: 'green' | 'yellow' | 'red';
    label: string;
  }>;
  /** Steam effect intensity (0-1) based on pressure */
  steamIntensity?: number;
}

/**
 * Chronometer: Latency measurement
 * Metaphor: Nautical chronometer dial
 */
interface ChronometerProps extends BaseGaugeProps {
  type: 'chronometer';
  /** Target latency (optimal operating point) */
  targetValue: number;
  /** Historical data for trend arc */
  history: number[];
  /** Show trend indicator */
  showTrend: boolean;
}

/**
 * Fuel Meter: Token consumption tracking
 * Metaphor: Steam locomotive fuel gauge
 */
interface FuelMeterProps extends BaseGaugeProps {
  type: 'fuel';
  /** Current consumption rate */
  flowRate: number;
  /** Peak capacity indicator */
  capacity: number;
  /** Remaining estimate */
  timeRemaining?: number;
  /** Fill direction */
  orientation: 'vertical' | 'horizontal';
}

/**
 * Compound Gauge: Multi-metric system health
 * Metaphor: Boiler room pressure manifold
 */
interface CompoundGaugeProps extends BaseGaugeProps {
  type: 'compound';
  /** Multiple needles for different metrics */
  needles: Array<{
    id: string;
    value: number;
    color: string;
    label: string;
  }>;
  /** Master warning lamp state */
  warningState: 'off' | 'warning' | 'critical';
}
```

### 2.3 Animation Strategy: Spring Physics vs Tweens

#### Why Spring Physics?

Traditional CSS transitions and tweens produce robotic, linear movement. Real mechanical gauges exhibit:

1. **Momentum** - Needles overshoot slightly before settling
2. **Damping** - Oscillations decay based on friction
3. **Inertia** - Heavier needles accelerate/decelerate more slowly

#### Implementation: Custom Spring Hook

```typescript
// hooks/useGaugeSpring.ts

import { useSpring, useMotionValue, useTransform } from 'framer-motion';
import { useEffect, useRef } from 'react';

interface SpringConfig {
  stiffness: number;
  damping: number;
  mass: number;
}

/**
 * Simulates mechanical gauge needle physics
 * Models Hooke's law with damping for realistic movement
 */
export function useGaugeSpring(
  targetValue: number,
  config: SpringConfig = { stiffness: 120, damping: 14, mass: 1 }
) {
  const motionValue = useMotionValue(0);

  const spring = useSpring(motionValue, {
    stiffness: config.stiffness,
    damping: config.damping,
    mass: config.mass,
    restDelta: 0.001,
  });

  // Add "mechanical wobble" for realism at high velocities
  const wobble = useTransform(spring, (latest) => {
    const velocity = spring.getVelocity();
    const wobbleAmount = Math.min(Math.abs(velocity) * 0.002, 0.5);
    return Math.sin(Date.now() * 0.05) * wobbleAmount;
  });

  useEffect(() => {
    motionValue.set(targetValue);
  }, [targetValue, motionValue]);

  return { value: spring, wobble };
}

/**
 * Specialized hook for pressure gauges with "steam hiss" vibration
 */
export function usePressureSpring(
  pressureValue: number,
  isCritical: boolean
) {
  const baseSpring = useGaugeSpring(pressureValue, {
    stiffness: 100,
    damping: 10,
    mass: 1.5, // Heavier needle for pressure gauges
  });

  // Add high-frequency vibration when in critical zone
  const vibration = useTransform(baseSpring.value, () => {
    if (!isCritical) return 0;
    // Simulate mechanical vibration at 30Hz
    return Math.sin(Date.now() * 0.03) * 0.8;
  });

  const finalValue = useTransform(
    [baseSpring.value, vibration],
    ([base, vib]) => (base as number) + (vib as number)
  );

  return { value: finalValue, isSettled: baseSpring.value.isAnimating };
}
```

#### Animation Timing Reference

| Gauge Type | Stiffness | Damping | Mass | Settle Time | Character |
|------------|-----------|---------|------|-------------|-----------|
| Pressure | 100 | 10 | 1.5 | ~800ms | Heavy industrial feel |
| Chronometer | 150 | 20 | 0.8 | ~400ms | Precise, responsive |
| Fuel Meter | 80 | 12 | 1.2 | ~1000ms | Smooth, flowing |
| Compound | 120 | 15 | 1.0 | ~600ms | Balanced |

### 2.4 Render Engine Comparison: Canvas vs SVG vs WebGL

#### Decision Matrix

| Criteria | SVG | Canvas 2D | WebGL | Winner |
|----------|-----|-----------|-------|--------|
| **Styling** | CSS + DOM | Programmatic | Shaders | SVG |
| **Accessibility** | Native ARIA | Manual | Difficult | SVG |
| **Resolution Independence** | Perfect | Requires scaling | Perfect | Tie |
| **Animation Performance** | Good (20-30 gauges) | Better (50+ gauges) | Best (100+) | Canvas |
| **Particle Effects** | Poor | Good | Excellent | Canvas/WebGL |
| **Development Velocity** | Fast | Medium | Slow | SVG |
| **Mobile Battery** | Good | Moderate | Poor | SVG |
| **Steampunk Detail** | Excellent (filters) | Good | Overkill | SVG |

#### Hybrid Rendering Strategy

```
┌─────────────────────────────────────────────────────────────┐
│                    Render Layer Architecture                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Z-Index 10: Particle Layer (OffscreenCanvas in Worker)     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Steam vents, smoke, spark effects                  │   │
│  │  Rendered at 30fps (decoupled from main thread)     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Z-Index 5: Gear Layer (SVG with CSS transforms)            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Decorative gears, linkages                         │   │
│  │  CSS animation for rotation                         │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Z-Index 1: Gauge Layer (SVG)                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Faces, ticks, needles, glass reflections           │   │
│  │  Framer Motion for smooth animations                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
│  Z-Index 0: Background Layer (CSS)                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  Parchment texture, brass patina, shadows           │   │
│  │  CSS gradients and filters                          │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### SVG Gauge Component Structure

```typescript
// components/gauges/PressureGauge.tsx

import { motion, useTransform } from 'framer-motion';
import { useGaugeSpring } from '@/hooks/useGaugeSpring';
import { useTheme } from '@/hooks/useTheme';

interface PressureGaugeProps {
  value: number;
  min: number;
  max: number;
  size?: number;
  zones: ZoneConfig[];
}

export function PressureGauge({
  value,
  min,
  max,
  size = 200,
  zones,
}: PressureGaugeProps) {
  const theme = useTheme();
  const { value: animatedValue } = useGaugeSpring(value, {
    stiffness: 100,
    damping: 10,
    mass: 1.5,
  });

  // Transform value to rotation angle (-135deg to +135deg)
  const rotation = useTransform(
    animatedValue,
    [min, max],
    [-135, 135]
  );

  const center = size / 2;
  const radius = (size - 40) / 2;

  return (
    <div
      className="pressure-gauge"
      style={{ width: size, height: size }}
      role="meter"
      aria-valuemin={min}
      aria-valuemax={max}
      aria-valuenow={value}
      aria-label={`Steam pressure: ${value} PSI`}
    >
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        className="gauge-svg"
      >
        <defs>
          {/* Metallic gradient for needle */}
          <linearGradient id="needleGradient" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor={theme.colors.brass.dark} />
            <stop offset="50%" stopColor={theme.colors.brass.light} />
            <stop offset="100%" stopColor={theme.colors.brass.dark} />
          </linearGradient>

          {/* Glass reflection overlay */}
          <radialGradient id="glassGlare" cx="30%" cy="30%" r="50%">
            <stop offset="0%" stopColor="white" stopOpacity="0.3" />
            <stop offset="100%" stopColor="white" stopOpacity="0" />
          </radialGradient>

          {/* Drop shadow for depth */}
          <filter id="dropShadow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur in="SourceAlpha" stdDeviation="3" />
            <feOffset dx="2" dy="4" result="offsetblur" />
            <feComponentTransfer>
              <feFuncA type="linear" slope="0.5" />
            </feComponentTransfer>
            <feMerge>
              <feMergeNode />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        {/* Outer brass bezel */}
        <circle
          cx={center}
          cy={center}
          r={radius + 15}
          fill="url(#brassGradient)"
          filter="url(#dropShadow)"
        />

        {/* Face background (parchment) */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill={theme.colors.parchment.base}
          stroke={theme.colors.brass.base}
          strokeWidth={2}
        />

        {/* Color zones */}
        {zones.map((zone, i) => (
          <ZoneArc
            key={i}
            center={center}
            radius={radius - 10}
            startAngle={valueToAngle(zone.start, min, max)}
            endAngle={valueToAngle(zone.end, min, max)}
            color={theme.colors.zones[zone.color]}
          />
        ))}

        {/* Tick marks */}
        <TickMarks
          center={center}
          radius={radius - 20}
          min={min}
          max={max}
          majorStep={10}
          minorStep={2}
        />

        {/* Needle rotation group */}
        <motion.g
          style={{
            rotate: rotation,
            originX: center,
            originY: center,
          }}
        >
          {/* Needle */}
          <path
            d={`
              M ${center} ${center + 5}
              L ${center} ${center - radius + 20}
              L ${center - 4} ${center - radius + 30}
              L ${center} ${center - radius + 35}
              L ${center + 4} ${center - radius + 30}
              L ${center} ${center - radius + 20}
              Z
            `}
            fill="url(#needleGradient)"
            filter="url(#dropShadow)"
          />
        </motion.g>

        {/* Center cap */}
        <circle
          cx={center}
          cy={center}
          r={8}
          fill={theme.colors.brass.base}
          stroke={theme.colors.brass.dark}
          strokeWidth={2}
        />

        {/* Glass reflection */}
        <circle
          cx={center}
          cy={center}
          r={radius - 5}
          fill="url(#glassGlare)"
          pointerEvents="none"
        />
      </svg>

      {/* Value display */}
      <div className="gauge-value">
        <span className="value-number">{Math.round(value)}</span>
        <span className="value-unit">PSI</span>
      </div>
    </div>
  );
}
```

---

## 3. Real-Time Data Pipeline

### 3.1 Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Real-Time Data Pipeline                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌──────────────┐                                                   │
│   │   Prometheus │  (Metrics Store)                                   │
│   │   /metrics   │                                                   │
│   └──────┬───────┘                                                   │
│          │                                                           │
│          ▼                                                           │
│   ┌──────────────┐     ┌──────────────┐     ┌──────────────┐       │
│   │  Go Backend  │────▶│   SSE Hub    │────▶│  Dashboard   │       │
│   │  (rad-gateway)│     │  /events     │     │   Client     │       │
│   │              │     │              │     │              │       │
│   │ - Collects   │     │ - Buffers    │     │ - Subscribes │       │
│   │   metrics    │     │ - Throttles  │     │ - Renders    │       │
│   │ - Aggregates │     │ - Broadcasts │     │ - Animates   │       │
│   └──────────────┘     └──────────────┘     └──────┬───────┘       │
│                                                     │                │
│                                                     ▼                │
│   ┌─────────────────────────────────────────────────────────┐      │
│   │                  Client State Management                  │      │
│   ├─────────────────────────────────────────────────────────┤      │
│   │                                                          │      │
│   │   Zustand Store (Global UI State)                        │      │
│   │   ├── activeGauges: Map<string, GaugeConfig>             │      │
│   │   ├── theme: ThemeConfig                                 │      │
│   │   └── viewState: ViewState                               │      │
│   │                                                          │      │
│   │   TanStack Query (Server State Cache)                    │      │
│   │   ├── metrics: MetricPoint[] (time-series)               │      │
│   │   ├── alerts: Alert[]                                    │      │
│   │   └── providerHealth: ProviderStatus[]                   │      │
│   │                                                          │      │
│   │   Custom Hooks                                           │      │
│   │   ├── useMetricsStream() - SSE connection                │      │
│   │   ├── useSmoothValue() - Data smoothing                  │      │
│   │   └── useThrottledUpdate() - Render throttling           │      │
│   │                                                          │      │
│   └─────────────────────────────────────────────────────────┘      │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 Server-Sent Events Architecture

#### Why SSE over WebSocket?

| Factor | SSE | WebSocket | Recommendation |
|--------|-----|-----------|----------------|
| Direction | Server -> Client | Bidirectional | SSE adequate (metrics are push-only) |
| Reconnection | Automatic | Manual | SSE wins for reliability |
| Protocol | HTTP/1.1 or HTTP/2 | TCP | SSE simpler infrastructure |
| Browser Support | Excellent (IE requires polyfill) | Excellent | Tie |
| Binary Data | Base64 encoded | Native | Not needed for JSON metrics |
| Connection Overhead | Lower (one request) | Higher (handshake) | SSE more efficient |
| Firewall Friendly | Yes (HTTP port) | Sometimes blocked | SSE wins |

#### SSE Implementation

```typescript
// hooks/useMetricsStream.ts

import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useGaugeStore } from '@/stores/gaugeStore';

interface MetricEvent {
  type: 'metric' | 'alert' | 'heartbeat';
  timestamp: number;
  payload: MetricPayload | AlertPayload;
}

interface MetricPayload {
  metricId: string;
  value: number;
  labels: Record<string, string>;
}

interface StreamConfig {
  endpoint: string;
  retryInterval?: number;
  maxRetries?: number;
  bufferSize?: number;
}

/**
 * Manages SSE connection for real-time metrics
 * Implements exponential backoff, buffering, and automatic reconnection
 */
export function useMetricsStream(config: StreamConfig) {
  const queryClient = useQueryClient();
  const eventSourceRef = useRef<EventSource | null>(null);
  const bufferRef = useRef<MetricEvent[]>([]);
  const retryCountRef = useRef(0);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  const { updateMetric, setConnectionStatus } = useGaugeStore();

  const connect = useCallback(() => {
    if (eventSourceRef.current?.readyState === EventSource.OPEN) {
      return;
    }

    setConnectionStatus('connecting');

    const es = new EventSource(config.endpoint, {
      withCredentials: true,
    });

    es.onopen = () => {
      console.info('[SSE] Connected to metrics stream');
      setConnectionStatus('connected');
      retryCountRef.current = 0;

      // Flush any buffered events
      if (bufferRef.current.length > 0) {
        processBatch(bufferRef.current);
        bufferRef.current = [];
      }
    };

    es.onmessage = (event) => {
      try {
        const metricEvent: MetricEvent = JSON.parse(event.data);

        if (metricEvent.type === 'heartbeat') {
          // Reset connection timeout
          return;
        }

        // Buffer events during high-frequency bursts
        bufferRef.current.push(metricEvent);

        if (bufferRef.current.length >= (config.bufferSize || 10)) {
          processBatch(bufferRef.current);
          bufferRef.current = [];
        }
      } catch (err) {
        console.error('[SSE] Failed to parse metric event:', err);
      }
    };

    es.onerror = (error) => {
      console.error('[SSE] Connection error:', error);
      setConnectionStatus('error');
      es.close();

      // Exponential backoff retry
      const retryDelay = Math.min(
        1000 * Math.pow(2, retryCountRef.current),
        30000 // Max 30 seconds
      );

      if (retryCountRef.current < (config.maxRetries || Infinity)) {
        retryCountRef.current++;
        timeoutRef.current = setTimeout(connect, retryDelay);
      }
    };

    eventSourceRef.current = es;
  }, [config]);

  const processBatch = useCallback((events: MetricEvent[]) => {
    // Update React Query cache
    events.forEach((event) => {
      if (event.type === 'metric') {
        const payload = event.payload as MetricPayload;

        // Update time-series data
        queryClient.setQueryData(
          ['metrics', payload.metricId],
          (old: number[] = []) => {
            const updated = [...old, payload.value];
            // Keep last 100 points for sparklines
            return updated.slice(-100);
          }
        );

        // Update gauge store for immediate animation
        updateMetric(payload.metricId, {
          value: payload.value,
          timestamp: event.timestamp,
          labels: payload.labels,
        });
      }
    });
  }, [queryClient, updateMetric]);

  // Throttled flush for remaining buffered events
  useEffect(() => {
    const flushInterval = setInterval(() => {
      if (bufferRef.current.length > 0) {
        processBatch(bufferRef.current);
        bufferRef.current = [];
      }
    }, 100); // 100ms max latency

    return () => clearInterval(flushInterval);
  }, [processBatch]);

  useEffect(() => {
    connect();

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      eventSourceRef.current?.close();
      setConnectionStatus('disconnected');
    };
  }, [connect]);

  return {
    connectionStatus: useGaugeStore((state) => state.connectionStatus),
    reconnect: connect,
  };
}
```

### 3.3 Data Smoothing and Throttling

#### The Problem: Jitter in Real-Time Metrics

Raw metric data often contains:
- **Noise** - Random fluctuations from measurement variance
- **Spikes** - Brief anomalies that distort visualization
- **Burst patterns** - Sudden floods of updates causing UI jank

#### Solution: Multi-Stage Smoothing Pipeline

```typescript
// lib/dataSmoothing.ts

/**
 * Exponential Moving Average (EMA) for noise reduction
 * Weighted average giving more importance to recent values
 */
export class ExponentialMovingAverage {
  private alpha: number;
  private smoothedValue: number | null = null;

  constructor(alpha: number = 0.3) {
    this.alpha = alpha; // 0.3 = 30% weight on new value
  }

  update(value: number): number {
    if (this.smoothedValue === null) {
      this.smoothedValue = value;
    } else {
      this.smoothedValue =
        this.alpha * value + (1 - this.alpha) * this.smoothedValue;
    }
    return this.smoothedValue;
  }

  reset(): void {
    this.smoothedValue = null;
  }
}

/**
 * Kalman-inspired filter for gauge needle stability
 * Estimates true value while filtering out measurement noise
 */
export class GaugeFilter {
  private estimate: number;
  private errorCovariance: number;
  private readonly processNoise: number;
  private readonly measurementNoise: number;

  constructor(
    initialValue: number = 0,
    processNoise: number = 0.01,
    measurementNoise: number = 0.1
  ) {
    this.estimate = initialValue;
    this.errorCovariance = 1;
    this.processNoise = processNoise;
    this.measurementNoise = measurementNoise;
  }

  update(measurement: number): number {
    // Prediction step
    this.errorCovariance += this.processNoise;

    // Update step
    const kalmanGain =
      this.errorCovariance / (this.errorCovariance + this.measurementNoise);
    this.estimate += kalmanGain * (measurement - this.estimate);
    this.errorCovariance *= (1 - kalmanGain);

    return this.estimate;
  }
}

/**
 * Rate limiter for render updates
 * Ensures 60fps target by coalescing rapid updates
 */
export class ThrottledUpdater<T> {
  private lastUpdate: number = 0;
  private pendingValue: T | null = null;
  private rafId: number | null = null;
  private readonly minInterval: number;
  private readonly callback: (value: T) => void;

  constructor(callback: (value: T) => void, targetFps: number = 60) {
    this.callback = callback;
    this.minInterval = 1000 / targetFps;
  }

  update(value: T): void {
    const now = performance.now();
    this.pendingValue = value;

    if (now - this.lastUpdate >= this.minInterval) {
      this.flush();
    } else if (!this.rafId) {
      this.rafId = requestAnimationFrame(() => this.flush());
    }
  }

  private flush(): void {
    if (this.pendingValue !== null) {
      this.callback(this.pendingValue);
      this.lastUpdate = performance.now();
      this.pendingValue = null;
    }
    this.rafId = null;
  }

  dispose(): void {
    if (this.rafId) {
      cancelAnimationFrame(this.rafId);
    }
  }
}
```

#### Hook Integration

```typescript
// hooks/useSmoothMetric.ts

import { useState, useEffect, useRef } from 'react';
import { GaugeFilter, ThrottledUpdater } from '@/lib/dataSmoothing';

interface UseSmoothMetricOptions {
  smoothingFactor?: number;
  targetFps?: number;
  onUpdate?: (value: number) => void;
}

/**
 * Combines Kalman filtering with render throttling
 * for optimal gauge animation performance
 */
export function useSmoothMetric(
  rawValue: number,
  options: UseSmoothMetricOptions = {}
) {
  const { targetFps = 60, onUpdate } = options;

  const filterRef = useRef(new GaugeFilter(rawValue));
  const [smoothedValue, setSmoothedValue] = useState(rawValue);
  const throttlerRef = useRef<ThrottledUpdater<number> | null>(null);

  useEffect(() => {
    throttlerRef.current = new ThrottledUpdater((value) => {
      setSmoothedValue(value);
      onUpdate?.(value);
    }, targetFps);

    return () => throttlerRef.current?.dispose();
  }, [targetFps, onUpdate]);

  useEffect(() => {
    const filtered = filterRef.current.update(rawValue);
    throttlerRef.current?.update(filtered);
  }, [rawValue]);

  return smoothedValue;
}
```

### 3.4 State Management Architecture

```typescript
// stores/gaugeStore.ts

import { create } from 'zustand';
import { immer } from 'zustand/middleware/immer';
import { subscribeWithSelector } from 'zustand/middleware';

interface MetricPoint {
  value: number;
  timestamp: number;
  labels: Record<string, string>;
}

interface GaugeState {
  // Connection state
  connectionStatus: 'connected' | 'connecting' | 'error' | 'disconnected';
  lastHeartbeat: number;

  // Metric data
  metrics: Map<string, MetricPoint>;
  timeSeries: Map<string, number[]>;

  // Alert state
  activeAlerts: Alert[];
  acknowledgedAlerts: Set<string>;

  // UI state
  selectedTimeRange: TimeRange;
  focusedGauge: string | null;
}

interface GaugeActions {
  updateMetric: (metricId: string, point: MetricPoint) => void;
  setConnectionStatus: (status: GaugeState['connectionStatus']) => void;
  acknowledgeAlert: (alertId: string) => void;
  setTimeRange: (range: TimeRange) => void;
  focusGauge: (gaugeId: string | null) => void;
}

export const useGaugeStore = create<GaugeState & GaugeActions>()(
  subscribeWithSelector(
    immer((set) => ({
      // Initial state
      connectionStatus: 'disconnected',
      lastHeartbeat: 0,
      metrics: new Map(),
      timeSeries: new Map(),
      activeAlerts: [],
      acknowledgedAlerts: new Set(),
      selectedTimeRange: '5m',
      focusedGauge: null,

      // Actions
      updateMetric: (metricId, point) =>
        set((state) => {
          state.metrics.set(metricId, point);

          // Maintain time-series for sparklines
          const series = state.timeSeries.get(metricId) || [];
          series.push(point.value);
          if (series.length > 100) series.shift();
          state.timeSeries.set(metricId, series);
        }),

      setConnectionStatus: (status) =>
        set((state) => {
          state.connectionStatus = status;
          if (status === 'connected') {
            state.lastHeartbeat = Date.now();
          }
        }),

      acknowledgeAlert: (alertId) =>
        set((state) => {
          state.acknowledgedAlerts.add(alertId);
        }),

      setTimeRange: (range) =>
        set((state) => {
          state.selectedTimeRange = range;
        }),

      focusGauge: (gaugeId) =>
        set((state) => {
          state.focusedGauge = gaugeId;
        }),
    }))
  )
);

// Selector hooks for optimized re-renders
export const useMetricValue = (metricId: string) =>
  useGaugeStore((state) => state.metrics.get(metricId)?.value ?? 0);

export const useMetricHistory = (metricId: string) =>
  useGaugeStore((state) => state.timeSeries.get(metricId) ?? []);
```

---

## 4. Performance Budget

### 4.1 Targets

| Metric | Target | Maximum | Measurement |
|--------|--------|---------|-------------|
| **Animation FPS** | 60 | 45 | Chrome DevTools FPS meter |
| **First Contentful Paint** | < 1.5s | 2.5s | Lighthouse |
| **Time to Interactive** | < 3s | 5s | Lighthouse |
| **Input Latency** | < 16ms | 50ms | Chrome DevTools |
| **Memory Usage** | < 100MB | 200MB | Chrome Task Manager |
| **Bundle Size (gzipped)** | < 150KB | 250KB | webpack-bundle-analyzer |
| **Metric Latency (backend to paint)** | < 100ms | 500ms | Custom timing API |

### 4.2 Gauge Capacity Limits

| View Type | Max Gauges | Render Strategy |
|-----------|------------|-----------------|
| Mobile Portrait | 4 | Single column, lazy-loaded |
| Mobile Landscape | 6 | 2-column grid |
| Tablet | 12 | 3-column grid |
| Desktop | 20 | 4-5 column grid |
| Ultrawide | 24 | 6-column grid + overflow |

### 4.3 Optimization Strategies

```typescript
// components/performance/VirtualizedGaugeGrid.tsx

import { useRef, useCallback } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { PressureGauge } from '../gauges/PressureGauge';

interface VirtualizedGaugeGridProps {
  gauges: GaugeConfig[];
  rowHeight: number;
  overscan?: number;
}

/**
 * Virtualized grid for handling 50+ gauges efficiently
 * Only renders visible gauges + overscan buffer
 */
export function VirtualizedGaugeGrid({
  gauges,
  rowHeight,
  overscan = 5,
}: VirtualizedGaugeGridProps) {
  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: gauges.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => rowHeight,
    overscan,
  });

  const virtualItems = virtualizer.getVirtualItems();

  return (
    <div
      ref={parentRef}
      className="gauge-grid-container"
      style={{ height: '100%', overflow: 'auto' }}
    >
      <div
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {virtualItems.map((virtualItem) => {
          const gauge = gauges[virtualItem.index];
          return (
            <div
              key={gauge.id}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                height: `${virtualItem.size}px`,
                transform: `translateY(${virtualItem.start}px)`,
              }}
            >
              <PressureGauge
                {...gauge}
                // Pause animations when not visible
                isActive={virtualItem.index >= virtualItems[0].index &&
                         virtualItem.index <= virtualItems[virtualItems.length - 1].index}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
}
```

### 4.4 Memory Management

```typescript
// hooks/useMemoryManagement.ts

import { useEffect, useRef } from 'react';

/**
 * Automatic cleanup for long-running dashboards
 * Prevents memory leaks from accumulated event listeners and caches
 */
export function useMemoryManagement() {
  const cleanupIntervalRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    // Periodic garbage collection hints
    cleanupIntervalRef.current = setInterval(() => {
      // Clear old time-series data
      if ('gc' in window) {
        (window as any).gc();
      }

      // Log memory usage for monitoring
      if (performance.memory) {
        const mem = performance.memory as any;
        const usedMB = mem.usedJSHeapSize / 1048576;
        const totalMB = mem.totalJSHeapSize / 1048576;

        if (usedMB > 150) {
          console.warn(`[Memory] High usage: ${usedMB.toFixed(1)}MB / ${totalMB.toFixed(1)}MB`);
        }
      }
    }, 30000); // Every 30 seconds

    return () => {
      if (cleanupIntervalRef.current) {
        clearInterval(cleanupIntervalRef.current);
      }
    };
  }, []);
}

/**
 * Limit time-series data growth
 */
export function useBoundedHistory<T>(
  data: T[],
  maxPoints: number = 100
): T[] {
  return data.slice(-maxPoints);
}
```

---

## 5. Theming System

### 5.1 Design Token Structure

```typescript
// theme/tokens.ts

/**
 * Steampunk-themed design tokens for Brass Relay
 * All colors reference physical materials (brass, copper, parchment)
 */
export const steampunkTokens = {
  colors: {
    // Metallic palette
    brass: {
      base: '#B5A642',
      light: '#D4C574',
      dark: '#8B7D2E',
      shadow: '#5C5218',
      highlight: '#E8DCA8',
    },
    copper: {
      base: '#B87333',
      light: '#D4915A',
      dark: '#8A5320',
      patina: '#4A7C59',
    },
    bronze: {
      base: '#CD7F32',
      light: '#E09B54',
      dark: '#A06020',
    },
    steel: {
      base: '#71797E',
      light: '#9BA3A8',
      dark: '#4A5054',
    },

    // Material palette
    parchment: {
      base: '#F5F0E1',
      aged: '#E8E0C5',
      dark: '#D4C9A8',
      ink: '#3D3B30',
    },
    leather: {
      base: '#8B4513',
      dark: '#5C2E0C',
      worn: '#A65E2F',
    },

    // Status colors (steampunk-appropriate)
    zones: {
      green: '#4A7C59', // Patinated copper
      yellow: '#D4A017', // Old gold
      red: '#8B2635', // Deep crimson
    },

    // Functional
    steam: {
      light: 'rgba(255, 255, 255, 0.8)',
      medium: 'rgba(200, 200, 200, 0.6)',
      dark: 'rgba(150, 150, 150, 0.4)',
    },
  },

  typography: {
    fontFamily: {
      display: '"Playfair Display", serif', // Victorian headers
      body: '"Crimson Text", serif', // Readable body
      mono: '"Fira Code", monospace', // Technical readouts
    },
    sizes: {
      xs: '0.75rem',
      sm: '0.875rem',
      base: '1rem',
      lg: '1.125rem',
      xl: '1.25rem',
      '2xl': '1.5rem',
      '3xl': '2rem',
      '4xl': '2.5rem',
    },
  },

  animation: {
    timing: {
      instant: '0ms',
      fast: '150ms',
      normal: '300ms',
      slow: '500ms',
      stately: '800ms',
    },
    easing: {
      mechanical: 'cubic-bezier(0.34, 1.56, 0.64, 1)', // Overshoot
      smooth: 'cubic-bezier(0.4, 0, 0.2, 1)',
      gear: 'cubic-bezier(0.68, -0.55, 0.265, 1.55)', // Bounce
    },
  },

  shadows: {
    bezel: 'inset 2px 2px 5px rgba(0,0,0,0.3), inset -2px -2px 5px rgba(255,255,255,0.1)',
    raised: '4px 4px 10px rgba(0,0,0,0.4), -2px -2px 5px rgba(255,255,255,0.1)',
    glass: 'inset 0 0 20px rgba(255,255,255,0.1)',
  },

  textures: {
    parchment: 'url(/textures/parchment.png)',
    brass: 'url(/textures/brushed-brass.png)',
    noise: 'url(/textures/subtle-noise.png)',
  },
} as const;

export type ThemeTokens = typeof steampunkTokens;
```

### 5.2 CSS Custom Properties Integration

```css
/* styles/theme.css */

:root {
  /* Brass palette */
  --brass-base: #B5A642;
  --brass-light: #D4C574;
  --brass-dark: #8B7D2E;
  --brass-shadow: #5C5218;
  --brass-highlight: #E8DCA8;

  /* Copper palette */
  --copper-base: #B87333;
  --copper-light: #D4915A;
  --copper-dark: #8A5320;
  --copper-patina: #4A7C59;

  /* Parchment palette */
  --parchment-base: #F5F0E1;
  --parchment-aged: #E8E0C5;
  --parchment-dark: #D4C9A8;
  --parchment-ink: #3D3B30;

  /* Status zones */
  --zone-steady: #4A7C59;      /* healthy -> steady pressure */
  --zone-caution: #D4A017;     /* degraded -> pressure drop */
  --zone-critical: #8B2635;    /* failing -> line rupture */

  /* Typography */
  --font-display: 'Playfair Display', serif;
  --font-body: 'Crimson Text', serif;
  --font-mono: 'Fira Code', monospace;

  /* Animation */
  --ease-mechanical: cubic-bezier(0.34, 1.56, 0.64, 1);
  --ease-gear: cubic-bezier(0.68, -0.55, 0.265, 1.55);

  /* Shadows */
  --shadow-bezel: inset 2px 2px 5px rgba(0,0,0,0.3), inset -2px -2px 5px rgba(255,255,255,0.1);
  --shadow-raised: 4px 4px 10px rgba(0,0,0,0.4);
}

/* High contrast mode for accessibility */
@media (prefers-contrast: high) {
  :root {
    --zone-steady: #006400;
    --zone-caution: #B8860B;
    --zone-critical: #8B0000;
    --parchment-ink: #000000;
  }
}

/* Dark mode variant (night watch theme) */
[data-theme='night-watch'] {
  --parchment-base: #1A1814;
  --parchment-aged: #252218;
  --parchment-ink: #C4B9A0;
  --brass-base: #8B7D2E;
  --brass-light: #A6963A;
}

/* Reduced motion for accessibility */
@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

### 5.3 Theme Provider Component

```typescript
// components/theme/ThemeProvider.tsx

import { createContext, useContext, useEffect, useState } from 'react';
import { steampunkTokens, ThemeTokens } from '@/theme/tokens';

theme ThemeMode = 'day' | 'night-watch' | 'high-contrast';

interface ThemeContextValue {
  tokens: ThemeTokens;
  mode: ThemeMode;
  setMode: (mode: ThemeMode) => void;
  toggleMode: () => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [mode, setMode] = useState<ThemeMode>('day');

  useEffect(() => {
    // Apply theme mode to document
    document.documentElement.setAttribute('data-theme', mode);

    // Sync with system preference
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (e: MediaQueryListEvent) => {
      if (mode === 'day' && e.matches) {
        setMode('night-watch');
      }
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [mode]);

  const toggleMode = () => {
    setMode((prev) =>
      prev === 'day' ? 'night-watch' : 'day'
    );
  };

  return (
    <ThemeContext.Provider
      value={{
        tokens: steampunkTokens,
        mode,
        setMode,
        toggleMode,
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
}

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return context;
};
```

---

## 6. Implementation Plan

### Phase 1: Static Gauge Foundation (Sprint 1-2)

**Deliverables:**
- [ ] Base SVG gauge component with props interface
- [ ] Pressure gauge (provider health visualization)
- [ ] CSS theme system with brass/parchment tokens
- [ ] Responsive grid layout
- [ ] Storybook documentation

**Technical Focus:**
- Component architecture
- SVG rendering optimization
- Theme token implementation
- Accessibility foundations

### Phase 2: Real-Time Data Integration (Sprint 3-4)

**Deliverables:**
- [ ] SSE connection hook with auto-reconnect
- [ ] Zustand store for metric state
- [ ] Chronometer dial (latency visualization)
- [ ] Data smoothing pipeline
- [ ] Connection status indicators

**Technical Focus:**
- Streaming data architecture
- State management optimization
- Error handling
- Performance monitoring

### Phase 3: Advanced Animations & Polish (Sprint 5-6)

**Deliverables:**
- [ ] Spring physics animation system
- [ ] Steam/smoke particle effects (Canvas)
- [ ] Fuel meter (token usage)
- [ ] Telegraph tape (live request stream)
- [ ] Gear decorations with CSS animations
- [ ] Mobile optimization

**Technical Focus:**
- Animation performance
- Particle systems
- Visual polish
- Mobile responsiveness

### Phase 4: Production Hardening (Sprint 7-8)

**Deliverables:**
- [ ] Virtualized rendering for 50+ gauges
- [ ] Memory leak prevention
- [ ] Error boundaries
- [ ] Comprehensive testing (>85% coverage)
- [ ] Performance benchmarks met
- [ ] Accessibility audit (WCAG 2.1 AA)

**Technical Focus:**
- Scalability
- Reliability
- Testing
- Documentation

---

## 7. Proof of Concept Code

### 7.1 Complete Pressure Gauge Component

```typescript
// components/gauges/PressureGauge/index.tsx

import { useMemo } from 'react';
import { motion, useTransform, useMotionValue } from 'framer-motion';
import { useSmoothMetric } from '@/hooks/useSmoothMetric';
import { useGaugeSpring } from '@/hooks/useGaugeSpring';
import { useTheme } from '@/hooks/useTheme';
import { formatValue, valueToAngle } from '@/lib/gaugeMath';

interface PressureGaugeProps {
  id: string;
  value: number;
  unit?: string;
  min?: number;
  max?: number;
  size?: number;
  label?: string;
  labelTechnical?: string;
  warningThreshold?: number;
  criticalThreshold?: number;
  zones?: ZoneConfig[];
}

export function PressureGauge({
  id,
  value,
  unit = 'PSI',
  min = 0,
  max = 100,
  size = 240,
  label = 'Boiler Pressure',
  labelTechnical = 'Provider Health',
  warningThreshold = 70,
  criticalThreshold = 90,
  zones = defaultZones,
}: PressureGaugeProps) {
  const theme = useTheme();

  // Data smoothing
  const smoothedValue = useSmoothMetric(value, {
    smoothingFactor: 0.3,
    targetFps: 60,
  });

  // Animation physics
  const { value: animatedValue } = useGaugeSpring(smoothedValue, {
    stiffness: 100,
    damping: 10,
    mass: 1.5,
  });

  // Transform value to rotation
  const rotation = useTransform(animatedValue, [min, max], [-135, 135]);

  // Determine alert state
  const alertState = useMemo(() => {
    if (value >= criticalThreshold) return 'critical';
    if (value >= warningThreshold) return 'warning';
    return 'normal';
  }, [value, warningThreshold, criticalThreshold]);

  const center = size / 2;
  const radius = (size - 60) / 2;

  return (
    <div
      className={`pressure-gauge pressure-gauge--${alertState}`}
      style={{ width: size, height: size + 40 }}
      role="meter"
      aria-valuemin={min}
      aria-valuemax={max}
      aria-valuenow={Math.round(value)}
      aria-label={`${labelTechnical}: ${formatValue(value)} ${unit}`}
    >
      {/* SVG Gauge */}
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        className="pressure-gauge__svg"
      >
        <defs>
          <BrassGradient id="brassGradient" theme={theme} />
          <GlassGlare id="glassGlare" />
          <DropShadow id="dropShadow" />
        </defs>

        {/* Bezel */}
        <Bezel
          center={center}
          radius={radius}
          theme={theme}
        />

        {/* Face */}
        <Face
          center={center}
          radius={radius}
          theme={theme}
        />

        {/* Color zones */}
        <Zones
          center={center}
          radius={radius - 15}
          min={min}
          max={max}
          zones={zones}
          theme={theme}
        />

        {/* Ticks */}
        <TickMarks
          center={center}
          radius={radius - 25}
          min={min}
          max={max}
          majorStep={(max - min) / 10}
          theme={theme}
        />

        {/* Animated needle */}
        <motion.g
          style={{
            rotate: rotation,
            originX: `${center}px`,
            originY: `${center}px`,
          }}
        >
          <Needle
            center={center}
            radius={radius}
            theme={theme}
            alertState={alertState}
          />
        </motion.g>

        {/* Center cap */}
        <CenterCap center={center} theme={theme} />

        {/* Glass reflection */}
        <GlassReflection center={center} radius={radius} />
      </svg>

      {/* Label and value */}
      <div className="pressure-gauge__display">
        <div className="pressure-gauge__label">{label}</div>
        <div className="pressure-gauge__value">
          <span className="value-number">{formatValue(smoothedValue)}</span>
          <span className="value-unit">{unit}</span>
        </div>
      </div>

      {/* Status lamp */}
      <StatusLamp state={alertState} />
    </div>
  );
}

// Sub-components
function BrassGradient({ id, theme }: { id: string; theme: ThemeTokens }) {
  return (
    <linearGradient id={id} x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stopColor={theme.colors.brass.highlight} />
      <stop offset="50%" stopColor={theme.colors.brass.base} />
      <stop offset="100%" stopColor={theme.colors.brass.dark} />
    </linearGradient>
  );
}

function Bezel({ center, radius, theme }: BezelProps) {
  return (
    <circle
      cx={center}
      cy={center}
      r={radius + 12}
      fill="none"
      stroke="url(#brassGradient)"
      strokeWidth={16}
      filter="url(#dropShadow)"
    />
  );
}

function Face({ center, radius, theme }: FaceProps) {
  return (
    <circle
      cx={center}
      cy={center}
      r={radius}
      fill={theme.colors.parchment.base}
      stroke={theme.colors.brass.dark}
      strokeWidth={2}
    />
  );
}

function Needle({ center, radius, theme, alertState }: NeedleProps) {
  const needleColor = alertState === 'critical'
    ? theme.colors.zones.red
    : 'url(#brassGradient)';

  return (
    <path
      d={`
        M ${center} ${center + 6}
        L ${center} ${center - radius + 25}
        L ${center - 5} ${center - radius + 35}
        L ${center} ${center - radius + 40}
        L ${center + 5} ${center - radius + 35}
        L ${center} ${center - radius + 25}
        Z
      `}
      fill={needleColor}
      filter="url(#dropShadow)"
    />
  );
}
```

### 7.2 Data Streaming Hook

```typescript
// hooks/useBrassRelayMetrics.ts

import { useEffect, useRef, useCallback } from 'react';
import { useMetricsStream } from './useMetricsStream';
import { useGaugeStore } from '@/stores/gaugeStore';
import { MetricMapping } from '@/types/metrics';

/**
 * Maps RAD Gateway Prometheus metrics to themed gauge representations
 */
const METRIC_MAP: Record<string, MetricMapping> = {
  'rad_gateway_requests_total': {
    gaugeType: 'pressure',
    label: 'Boiler Load',
    labelTechnical: 'Request Rate',
    unit: 'RPS',
    min: 0,
    max: 1000,
    warningThreshold: 700,
    criticalThreshold: 900,
  },
  'rad_gateway_request_duration_seconds': {
    gaugeType: 'chronometer',
    label: 'Chronometer',
    labelTechnical: 'P95 Latency',
    unit: 'ms',
    min: 0,
    max: 2000,
    warningThreshold: 1000,
    criticalThreshold: 1500,
  },
  'rad_gateway_provider_requests_total': {
    gaugeType: 'pressure',
    label: 'Boiler Array',
    labelTechnical: 'Provider Load',
    unit: 'req/s',
    min: 0,
    max: 500,
  },
  'rad_gateway_tokens_consumed_total': {
    gaugeType: 'fuel',
    label: 'Fuel Consumption',
    labelTechnical: 'Token Usage',
    unit: 'tokens/s',
    min: 0,
    max: 10000,
  },
  'rad_gateway_active_connections': {
    gaugeType: 'compound',
    label: 'Pressure Manifold',
    labelTechnical: 'Active Connections',
    unit: 'conns',
    min: 0,
    max: 1000,
  },
};

/**
 * Comprehensive hook for Brass Relay dashboard metrics
 * Handles streaming, mapping, and gauge state management
 */
export function useBrassRelayMetrics() {
  const { connectionStatus } = useMetricsStream({
    endpoint: '/api/v1/events/metrics',
    bufferSize: 20,
    maxRetries: 10,
  });

  const updateMetric = useGaugeStore((state) => state.updateMetric);
  const setConnectionStatus = useGaugeStore((state) => state.setConnectionStatus);

  // Sync connection status
  useEffect(() => {
    setConnectionStatus(connectionStatus);
  }, [connectionStatus, setConnectionStatus]);

  // Get current metric values with their gauge mappings
  const getMetricForGauge = useCallback((metricId: string) => {
    const mapping = METRIC_MAP[metricId];
    const currentValue = useGaugeStore.getState().metrics.get(metricId);

    return {
      mapping,
      currentValue: currentValue?.value ?? 0,
      history: useGaugeStore.getState().timeSeries.get(metricId) ?? [],
    };
  }, []);

  return {
    connectionStatus,
    getMetricForGauge,
    metricIds: Object.keys(METRIC_MAP),
  };
}
```

### 7.3 Telegraph Tape Component

```typescript
// components/TelegraphTape/index.tsx

import { useRef, useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useGaugeStore } from '@/stores/gaugeStore';

interface RequestEvent {
  id: string;
  timestamp: number;
  provider: string;
  model: string;
  status: 'success' | 'warning' | 'error';
  duration: number;
  tokens: number;
}

/**
 * Scrolling telegraph-style display for live requests
 * Visual metaphor: Ticker tape from old telegraph systems
 */
export function TelegraphTape({ maxItems = 50 }: { maxItems?: number }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isPaused, setIsPaused] = useState(false);

  // Subscribe to request stream from store
  const requests = useGaugeStore((state) =>
    Array.from(state.metrics.entries())
      .filter(([key]) => key.startsWith('request_'))
      .map(([_, value]) => value)
      .slice(-maxItems)
  );

  // Auto-scroll to bottom
  useEffect(() => {
    if (!isPaused && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [requests, isPaused]);

  return (
    <div
      className="telegraph-tape"
      onMouseEnter={() => setIsPaused(true)}
      onMouseLeave={() => setIsPaused(false)}
      role="log"
      aria-label="Live request stream"
      aria-live="polite"
    >
      {/* Tape header */}
      <div className="telegraph-tape__header">
        <span className="tape-title">Telegraph Feed</span>
        <span className="tape-status">
          {isPaused ? 'PAUSED' : 'LIVE'}
        </span>
      </div>

      {/* Scrolling tape window */}
      <div
        ref={containerRef}
        className="telegraph-tape__window"
        style={{
          height: 300,
          overflowY: 'auto',
          scrollBehavior: isPaused ? 'auto' : 'smooth',
        }}
      >
        <AnimatePresence initial={false}>
          {requests.map((request) => (
            <RequestChit
              key={request.id}
              request={request as RequestEvent}
            />
          ))}
        </AnimatePresence>
      </div>

      {/* Edge gradients for depth effect */}
      <div className="telegraph-tape__fade telegraph-tape__fade--top" />
      <div className="telegraph-tape__fade telegraph-tape__fade--bottom" />
    </div>
  );
}

function RequestChit({ request }: { request: RequestEvent }) {
  const statusClass = `request-chit--${request.status}`;

  return (
    <motion.div
      className={`request-chit ${statusClass}`}
      initial={{ opacity: 0, x: -50, scale: 0.9 }}
      animate={{ opacity: 1, x: 0, scale: 1 }}
      exit={{ opacity: 0, height: 0 }}
      transition={{ duration: 0.3, ease: 'easeOut' }}
    >
      <div className="request-chit__timestamp">
        {new Date(request.timestamp).toLocaleTimeString()}
      </div>
      <div className="request-chit__route">
        <span className="provider">{request.provider}</span>
        <span className="separator">→</span>
        <span className="model">{request.model}</span>
      </div>
      <div className="request-chit__metrics">
        <span className="metric duration">{request.duration}ms</span>
        <span className="metric tokens">{request.tokens} tok</span>
      </div>
      <div className={`request-chit__status-indicator status-${request.status}`} />
    </motion.div>
  );
}
```

---

## 8. Risk Assessment

### 8.1 Browser Compatibility

| Feature | Chrome | Firefox | Safari | Edge | IE11 |
|---------|--------|---------|--------|------|------|
| SVG Filters | Full | Full | Full | Full | Partial |
| CSS Custom Properties | Full | Full | Full | Full | No |
| Server-Sent Events | Full | Full | Full | Full | Polyfill |
| OffscreenCanvas | Full | Full | No | Full | No |
| Framer Motion | Full | Full | Full | Full | No |

**Mitigation Strategies:**
- CSS custom properties: PostCSS fallback plugin
- OffscreenCanvas: Main-thread fallback
- Framer Motion: Reduced motion + CSS transitions fallback
- IE11: Graceful degradation (static gauges only)

### 8.2 Performance Pitfalls

| Risk | Impact | Mitigation |
|------|--------|------------|
| Too many gauges on screen | Frame drops | Virtualization, lazy loading |
| Unthrottled data updates | UI jank | 60fps throttling, batch updates |
| Memory leaks from closures | OOM crashes | useEffect cleanup, weak refs |
| Heavy particle effects | Battery drain | Canvas in workers, quality scaling |
| Large bundle size | Slow initial load | Code splitting, tree shaking |

### 8.3 Accessibility Concerns

| Concern | Level | Solution |
|---------|-------|----------|
| Screen reader gauge interpretation | Critical | `aria-valuenow`, `aria-label` with technical terms |
| Colorblind status indication | Critical | Pattern overlays, shape differences |
| Motion sensitivity | High | `prefers-reduced-motion` support |
| Keyboard navigation | High | Tab order, Enter/Space interaction |
| High contrast mode | Medium | `prefers-contrast` media query |

### 8.4 Maintenance Complexity

| Component | Complexity | Maintenance Burden |
|-----------|------------|-------------------|
| Spring physics engine | Medium | Low (Framer Motion handles this) |
| Particle system | High | Medium (Canvas worker complexity) |
| Theme system | Low | Low (CSS custom properties) |
| SSE connection | Medium | Medium (reconnection logic) |
| Gauge component library | Medium | High (visual consistency needs) |

**Mitigation:**
- Comprehensive Storybook documentation
- Visual regression testing (Chromatic)
- Strict TypeScript interfaces
- Component API stability commitment

---

## Appendices

### A. Metric Naming Mapping

| Technical Metric | Themed Display | Gauge Type |
|------------------|----------------|------------|
| `rad_gateway_requests_total` | Boiler Load | Pressure |
| `rad_gateway_request_duration_seconds` | Chronometer | Dial |
| `rad_gateway_error_rate` | Leak Indicator | Warning Lamp |
| `rad_gateway_provider_requests_total` | Boiler Array | Multi-Pressure |
| `rad_gateway_failover_attempts_total` | Track Switches | Counter |
| `rad_gateway_tokens_consumed_total` | Fuel Consumption | Meter |
| `rad_gateway_active_connections` | Pressure Manifold | Compound |
| `rad_gateway_queue_depth` | Backpressure | Vertical |

### B. Performance Testing Checklist

- [ ] 60fps maintained with 20 gauges
- [ ] Memory stays below 100MB after 1 hour
- [ ] Metric latency < 100ms end-to-end
- [ ] Mobile battery usage < 5% per hour
- [ ] First paint < 1.5s on 3G
- [ ] Lighthouse score > 90
- [ ] Screen reader announces gauge changes
- [ ] Keyboard navigation works throughout

### C. Reference Implementations

- **Tesla Energy**: Real-time animation smoothness
- **Grafana**: Gauge configuration patterns
- **Datadog**: Information density strategies
- **Flight Instruments**: Realistic mechanical feel
- **Jules Verne UI**: Victorian aesthetic reference

---

**Document Ownership**: Frontend Architecture Team
**Review Cycle**: Bi-weekly during implementation
**Next Review Date**: 2026-03-02
