/**
 * RAD Gateway Admin UI - SSE Hook
 * Real-time Integration Developer - Phase 5 Implementation
 *
 * Provides robust Server-Sent Events (EventSource) connectivity with:
 * - Automatic reconnection with exponential backoff
 * - Connection state management
 * - Event type filtering
 * - Error handling and recovery
 * - Clean connection lifecycle management
 *
 * Pessimist's Note: EventSource can be flaky. We handle:
 * - Network drops (with reconnection)
 * - Server errors (with backoff)
 * - Memory leaks (with proper cleanup)
 * - Race conditions (with connection state guards)
 */

import { useCallback, useEffect, useRef, useState, useMemo } from 'react';
import { apiClient } from '../api/client';

type SSEState = 'connecting' | 'open' | 'closed' | 'error';

interface SSEOptions {
  /** Event types to subscribe to (comma-separated). Empty = all events. */
  events?: string[];

  /** Authentication token. Will be appended as query param for SSE. */
  token?: string | null;

  /** Auto-connect on mount. Default: true */
  autoConnect?: boolean;

  /** Reconnection options */
  reconnect?: {
    /** Enable auto-reconnection. Default: true */
    enabled?: boolean;
    /** Initial retry delay in ms. Default: 1000 */
    initialDelayMs?: number;
    /** Maximum retry delay in ms. Default: 30000 */
    maxDelayMs?: number;
    /** Backoff multiplier. Default: 2 */
    backoffMultiplier?: number;
    /** Maximum number of retry attempts. Default: Infinity */
    maxAttempts?: number;
  };

  /** Heartbeat timeout in ms. If no event received, connection considered dead. Default: 45000 */
  heartbeatTimeoutMs?: number;

  /** Callbacks */
  onOpen?: () => void;
  onMessage?: (event: MessageEvent) => void;
  onError?: (error: Event) => void;
  onClose?: () => void;
}

interface SSEReturn {
  /** Current connection state */
  state: SSEState;

  /** Last received message */
  lastMessage: MessageEvent | null;

  /** Connection error if any */
  error: Error | null;

  /** Number of reconnection attempts made */
  reconnectAttempts: number;

  /** Whether currently reconnecting */
  isReconnecting: boolean;

  /** Connect manually (or reconnect) */
  connect: () => void;

  /** Disconnect manually */
  disconnect: () => void;
}

/**
 * Hook for Server-Sent Events (EventSource) with robust error handling
 * and automatic reconnection.
 *
 * @param endpoint - SSE endpoint path (e.g., '/v0/admin/events')
 * @param options - SSE configuration options
 *
 * @example
 * ```tsx
 * const { state, lastMessage, connect, disconnect } = useSSE('/v0/admin/events', {
 *   events: ['usage:realtime', 'provider:health'],
 *   token: authToken,
 *   onMessage: (event) => {
 *     const data = JSON.parse(event.data);
 *     console.log('Received:', data);
 *   },
 * });
 * ```
 */
export function useSSE(endpoint: string, options: SSEOptions = {}): SSEReturn {
  const {
    events,
    token,
    autoConnect = true,
    reconnect = {},
    heartbeatTimeoutMs = 45000,
    onOpen,
    onMessage,
    onError,
    onClose,
  } = options;

  const {
    enabled: reconnectEnabled = true,
    initialDelayMs = 1000,
    maxDelayMs = 30000,
    backoffMultiplier = 2,
    maxAttempts = Infinity,
  } = reconnect;

  // State
  const [state, setState] = useState<SSEState>('closed');
  const [lastMessage, setLastMessage] = useState<MessageEvent | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const [isReconnecting, setIsReconnecting] = useState(false);

  // Refs for mutable values that shouldn't trigger re-renders
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const heartbeatTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectCountRef = useRef(0);
  const currentDelayRef = useRef(initialDelayMs);
  const isMountedRef = useRef(true);
  const manuallyClosedRef = useRef(false);

  // Memoize event types string
  const eventsParam = useMemo(() => (events?.length ? events.join(',') : ''), [events]);

  /**
   * Clear all pending timeouts to prevent memory leaks
   */
  const clearAllTimeouts = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (heartbeatTimeoutRef.current) {
      clearTimeout(heartbeatTimeoutRef.current);
      heartbeatTimeoutRef.current = null;
    }
  }, []);

  /**
   * Build SSE URL with query parameters
   */
  const buildSSEUrl = useCallback((): string => {
    const baseUrl = apiClient['baseUrl'] || '';
    const url = new URL(endpoint, baseUrl || window.location.origin);

    // Add event filter if specified
    if (eventsParam) {
      url.searchParams.set('events', eventsParam);
    }

    // Add auth token as query param (EventSource doesn't support headers)
    if (token) {
      url.searchParams.set('token', token);
    }

    return url.toString();
  }, [endpoint, eventsParam, token]);

  /**
   * Reset heartbeat timeout - called on any received message
   */
  const resetHeartbeatTimeout = useCallback(() => {
    if (heartbeatTimeoutRef.current) {
      clearTimeout(heartbeatTimeoutRef.current);
    }

    heartbeatTimeoutRef.current = setTimeout(() => {
      if (!isMountedRef.current) return;

      console.warn('[SSE] Heartbeat timeout - connection may be dead');
      // Force reconnection
      disconnect();
      if (reconnectEnabled && !manuallyClosedRef.current) {
        scheduleReconnect();
      }
    }, heartbeatTimeoutMs);
  }, [heartbeatTimeoutMs, reconnectEnabled]);

  /**
   * Schedule a reconnection attempt with exponential backoff
   */
  const scheduleReconnect = useCallback(() => {
    if (!isMountedRef.current) return;
    if (reconnectCountRef.current >= maxAttempts) {
      setError(new Error(`Max reconnection attempts (${maxAttempts}) reached`));
      setIsReconnecting(false);
      return;
    }

    setIsReconnecting(true);

    const delay = Math.min(currentDelayRef.current, maxDelayMs);
    console.log(`[SSE] Reconnecting in ${delay}ms (attempt ${reconnectCountRef.current + 1})`);

    reconnectTimeoutRef.current = setTimeout(() => {
      if (!isMountedRef.current) return;

      reconnectCountRef.current += 1;
      setReconnectAttempts(reconnectCountRef.current);
      currentDelayRef.current *= backoffMultiplier;

      // Attempt reconnection
      connect();
    }, delay);
  }, [maxAttempts, maxDelayMs, backoffMultiplier]);

  /**
   * Close the EventSource connection
   */
  const closeConnection = useCallback(() => {
    if (eventSourceRef.current) {
      // Remove listeners before closing to prevent callbacks during cleanup
      eventSourceRef.current.onopen = null;
      eventSourceRef.current.onmessage = null;
      eventSourceRef.current.onerror = null;
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    clearAllTimeouts();
  }, [clearAllTimeouts]);

  /**
   * Connect to the SSE endpoint
   */
  const connect = useCallback(() => {
    if (!isMountedRef.current) return;

    // Close existing connection if any
    closeConnection();

    // Reset manually closed flag
    manuallyClosedRef.current = false;

    setState('connecting');
    setError(null);

    const url = buildSSEUrl();

    try {
      // EventSource doesn't support custom headers, so we use query params for auth
      const es = new EventSource(url);
      eventSourceRef.current = es;

      es.onopen = () => {
        if (!isMountedRef.current) {
          es.close();
          return;
        }

        console.log('[SSE] Connection opened');
        setState('open');
        setError(null);
        setIsReconnecting(false);

        // Reset reconnection counters on successful connection
        reconnectCountRef.current = 0;
        currentDelayRef.current = initialDelayMs;
        setReconnectAttempts(0);

        // Start heartbeat monitoring
        resetHeartbeatTimeout();

        onOpen?.();
      };

      es.onmessage = (event: MessageEvent) => {
        if (!isMountedRef.current) return;

        // Reset heartbeat on any message
        resetHeartbeatTimeout();

        setLastMessage(event);
        onMessage?.(event);
      };

      es.onerror = (errorEvent: Event) => {
        if (!isMountedRef.current) return;

        console.error('[SSE] Connection error:', errorEvent);
        setState('error');

        const error = new Error('SSE connection error');
        setError(error);
        onError?.(errorEvent);

        // Close the failed connection
        closeConnection();

        // Schedule reconnection if enabled
        if (reconnectEnabled && !manuallyClosedRef.current) {
          scheduleReconnect();
        }
      };
    } catch (err) {
      console.error('[SSE] Failed to create EventSource:', err);
      setState('error');
      setError(err instanceof Error ? err : new Error(String(err)));

      if (reconnectEnabled && !manuallyClosedRef.current) {
        scheduleReconnect();
      }
    }
  }, [
    buildSSEUrl,
    closeConnection,
    onOpen,
    onMessage,
    onError,
    reconnectEnabled,
    resetHeartbeatTimeout,
    scheduleReconnect,
    initialDelayMs,
  ]);

  /**
   * Disconnect manually
   */
  const disconnect = useCallback(() => {
    manuallyClosedRef.current = true;
    closeConnection();
    setState('closed');
    setIsReconnecting(false);
    clearAllTimeouts();
    onClose?.();
  }, [closeConnection, clearAllTimeouts, onClose]);

  // Auto-connect on mount if enabled
  useEffect(() => {
    isMountedRef.current = true;

    if (autoConnect) {
      connect();
    }

    return () => {
      isMountedRef.current = false;
      disconnect();
    };
  }, [autoConnect, connect, disconnect]);

  // Reconnect when dependencies change (token, events)
  useEffect(() => {
    if (state === 'open' || state === 'connecting') {
      // If already connected, reconnect with new params
      connect();
    }
  }, [token, eventsParam, connect, state]);

  return {
    state,
    lastMessage,
    error,
    reconnectAttempts,
    isReconnecting,
    connect,
    disconnect,
  };
}

/**
 * Hook for subscribing to specific SSE event types with typed data.
 *
 * @param endpoint - SSE endpoint path
 * @param eventType - Specific event type to listen for
 * @param options - Additional SSE options
 *
 * @example
 * ```tsx
 * const { data, error, state } = useSSEEvent<UsageMetrics>(
 *   '/v0/admin/events',
 *   'usage:realtime',
 *   { token: authToken }
 * );
 * ```
 */
export function useSSEEvent<T>(
  endpoint: string,
  eventType: string,
  options: Omit<SSEOptions, 'events' | 'onMessage'> & {
    onData?: (data: T) => void;
  } = {}
): Omit<SSEReturn, 'lastMessage'> & {
  data: T | null;
  lastRawMessage: MessageEvent | null;
} {
  const { onData, ...restOptions } = options;
  const [data, setData] = useState<T | null>(null);
  const dataRef = useRef<T | null>(null);

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const parsed = JSON.parse(event.data);
        if (parsed.type === eventType || eventType === '') {
          dataRef.current = parsed.payload as T;
          setData(parsed.payload as T);
          onData?.(parsed.payload as T);
        }
      } catch (err) {
        console.error('[useSSEEvent] Failed to parse event data:', err);
      }
    },
    [eventType, onData]
  );

  const sseResult = useSSE(endpoint, {
    ...restOptions,
    events: [eventType],
    onMessage: handleMessage,
  });

  return {
    ...sseResult,
    data,
    lastRawMessage: sseResult.lastMessage,
  };
}

/**
 * Hook for multiple SSE event subscriptions.
 * Returns a map of event types to their last received data.
 *
 * @param endpoint - SSE endpoint path
 * @param eventTypes - Array of event types to subscribe to
 * @param options - Additional SSE options
 *
 * @example
 * ```tsx
 * const { events, state } = useSSEEvents(
 *   '/v0/admin/events',
 *   ['usage:realtime', 'provider:health'],
 *   { token: authToken }
 * );
 *
 * // Access specific event data
 * const usageData = events.get('usage:realtime');
 * ```
 */
export function useSSEEvents(
  endpoint: string,
  eventTypes: string[],
  options: Omit<SSEOptions, 'events' | 'onMessage'> & {
    onEvent?: (type: string, data: unknown) => void;
  } = {}
): Omit<SSEReturn, 'lastMessage'> & {
  events: Map<string, unknown>;
} {
  const { onEvent, ...restOptions } = options;
  const eventsMapRef = useRef<Map<string, unknown>>(new Map());
  const [, forceUpdate] = useState({});

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const parsed = JSON.parse(event.data);
        if (parsed.type && eventTypes.includes(parsed.type)) {
          eventsMapRef.current.set(parsed.type, parsed.payload);
          forceUpdate({}); // Trigger re-render
          onEvent?.(parsed.type, parsed.payload);
        }
      } catch (err) {
        console.error('[useSSEEvents] Failed to parse event data:', err);
      }
    },
    [eventTypes, onEvent]
  );

  const sseResult = useSSE(endpoint, {
    ...restOptions,
    events: eventTypes,
    onMessage: handleMessage,
  });

  return {
    ...sseResult,
    events: eventsMapRef.current,
  };
}
