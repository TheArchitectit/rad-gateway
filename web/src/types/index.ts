/**
 * RAD Gateway Admin UI - Type Definitions
 * State Management Engineer - Phase 2 Implementation
 */

// ============================================================================
// Authentication Types
// ============================================================================

export interface User {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  avatar?: string;
  createdAt: string;
  lastLoginAt?: string;
}

export type UserRole = 'admin' | 'developer' | 'viewer';

export interface Permission {
  resource: string;
  actions: Action[];
}

export type Action = 'read' | 'write' | 'delete' | 'admin';

export interface AuthState {
  user: User | null;
  token: string | null;
  permissions: Permission[];
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}

// ============================================================================
// Workspace Types
// ============================================================================

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  description?: string;
  logo?: string;
  createdAt: string;
  updatedAt: string;
  ownerId: string;
  memberCount: number;
  settings: WorkspaceSettings;
}

export interface WorkspaceSettings {
  theme: 'light' | 'dark' | 'system';
  timezone: string;
  currency: string;
  dateFormat: string;
}

export interface WorkspaceState {
  current: Workspace | null;
  list: Workspace[];
  recent: Workspace[];
  favorites: string[];
  isLoading: boolean;
  error: string | null;
}

// ============================================================================
// Provider Types
// ============================================================================

export type ProviderStatus = 'healthy' | 'degraded' | 'unhealthy' | 'disabled';
export type CircuitBreakerState = 'closed' | 'open' | 'half-open';

export interface Provider {
  id: string;
  name: string;
  displayName: string;
  status: ProviderStatus;
  circuitBreaker: CircuitBreakerState;
  lastCheck?: string;
  requestCount24h: number;
  errorRate24h: number;
  latencyMs?: number;
  models: string[];
}

export interface ProviderHealth {
  status: 'healthy' | 'unhealthy';
  latencyMs: number;
  checkedAt: string;
}

// ============================================================================
// API Key Types
// ============================================================================

export interface APIKey {
  id: string;
  name: string;
  keyPreview: string;
  permissions: string[];
  rateLimit: number;
  usageLimit?: number;
  currentUsage: number;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
  isActive: boolean;
  workspaceId: string;
}

export interface CreateAPIKeyDTO {
  name: string;
  permissions: string[];
  rateLimit?: number;
  usageLimit?: number;
  expiresAt?: string;
}

// ============================================================================
// Usage & Analytics Types
// ============================================================================

export interface UsageFilters {
  startTime?: string;
  endTime?: string;
  apiKeyName?: string;
  provider?: string;
  status?: 'success' | 'error' | 'timeout';
}

export interface UsageRecord {
  timestamp: string;
  requestId: string;
  traceId: string;
  apiKeyName: string;
  incomingApiType: string;
  incomingModel: string;
  selectedModel: string;
  provider: string;
  responseStatus: string;
  durationMs: number;
  usage: TokenUsage;
}

export interface TokenUsage {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  costTotal: number;
}

export interface UsageMetrics {
  totalRequests: number;
  totalTokens: number;
  totalCost: number;
  averageLatency: number;
  errorRate: number;
  requestsPerSecond: number;
}

export interface TimeSeriesData {
  timestamp: string;
  value: number;
}

// ============================================================================
// Trace Types
// ============================================================================

export interface Trace {
  id: string;
  requestId: string;
  timestamp: string;
  duration: number;
  status: 'success' | 'error' | 'timeout';
  provider: string;
  model: string;
  phases: TracePhase[];
}

export interface TracePhase {
  name: string;
  startTime: string;
  duration: number;
  status: 'pending' | 'running' | 'completed' | 'error';
}

export interface TraceFilters {
  status?: string;
  model?: string;
  provider?: string;
  startTime?: string;
  endTime?: string;
}

// ============================================================================
// Model Route Types
// ============================================================================

export interface ModelRoute {
  modelId: string;
  candidates: RouteCandidate[];
}

export interface RouteCandidate {
  provider: 'mock' | 'openai' | 'anthropic' | 'gemini';
  model: string;
  weight: number;
}

// ============================================================================
// Control Room Types
// ============================================================================

export interface ControlRoom {
  id: string;
  name: string;
  description?: string;
  tags: Tag[];
  widgets: Widget[];
  layout: LayoutConfig;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Tag {
  category: string;
  value: string;
  color?: string;
}

export interface Widget {
  id: string;
  type: WidgetType;
  title: string;
  config: Record<string, unknown>;
  position: Position;
  size: Size;
}

export type WidgetType =
  | 'metric'
  | 'chart'
  | 'table'
  | 'provider-status'
  | 'request-stream'
  | 'custom';

export interface Position {
  x: number;
  y: number;
}

export interface Size {
  w: number;
  h: number;
}

export interface LayoutConfig {
  columns: number;
  rowHeight: number;
}

// ============================================================================
// UI State Types
// ============================================================================

export interface Notification {
  id: string;
  type: 'info' | 'success' | 'warning' | 'error';
  title: string;
  message?: string;
  timestamp: string;
  read: boolean;
  action?: {
    label: string;
    href: string;
  };
}

export interface UIState {
  sidebarCollapsed: boolean;
  theme: 'light' | 'dark' | 'system';
  notifications: Notification[];
  activeModal: string | null;
  globalSearchOpen: boolean;
  isLoading: boolean;
}

// ============================================================================
// Real-time Types
// ============================================================================

export interface RealtimeState {
  connected: boolean;
  subscriptions: string[];
  lastEvent: WebSocketEvent | null;
  reconnectAttempts: number;
}

export type WebSocketEvent =
  | UsageRealtimeEvent
  | ProviderHealthEvent
  | CircuitBreakerEvent
  | SystemAlertEvent;

export interface UsageRealtimeEvent {
  type: 'usage:realtime';
  data: RealtimeUsageMetrics;
}

export interface ProviderHealthEvent {
  type: 'provider:health';
  data: ProviderHealthUpdate;
}

export interface CircuitBreakerEvent {
  type: 'provider:circuit';
  data: CircuitBreakerUpdate;
}

export interface SystemAlertEvent {
  type: 'system:alert';
  data: SystemAlert;
}

export interface RealtimeUsageMetrics {
  requestsPerSecond: number;
  latencyMs: number;
  activeConnections: number;
  timestamp: string;
}

export interface ProviderHealthUpdate {
  provider: string;
  status: ProviderStatus;
  latencyMs: number;
  checkedAt: string;
}

export interface CircuitBreakerUpdate {
  provider: string;
  state: CircuitBreakerState;
  reason?: string;
  timestamp: string;
}

export interface SystemAlert {
  id: string;
  severity: 'info' | 'warning' | 'critical';
  title: string;
  message: string;
  timestamp: string;
}

// ============================================================================
// API Response Types
// ============================================================================

export interface ApiResponse<T> {
  data: T;
  meta?: {
    cursor?: string;
    hasMore?: boolean;
    totalCount?: number;
  };
}

export interface ApiError {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
    requestId?: string;
  };
}

// ============================================================================
// Pagination Types
// ============================================================================

export interface PaginationParams {
  cursor?: string;
  limit?: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: {
    cursor?: string;
    hasMore: boolean;
    totalCount?: number;
  };
}

// ============================================================================
// Configuration Types
// ============================================================================

export interface GatewayConfig {
  listenAddr: string;
  retryBudget: number;
  keysConfigured: number;
  models: Record<string, ModelCandidate[]>;
}

export interface ConfigUpdateRequest {
  retryBudget?: number;
  maintenanceMode?: boolean;
}

// ============================================================================
// Health Types
// ============================================================================

export interface HealthResponse {
  status: 'ok' | 'degraded' | 'error';
  version: string;
  timestamp: string;
}

export interface DetailedHealthResponse extends HealthResponse {
  components: {
    database: ComponentHealth;
    redis: ComponentHealth;
    providers: Record<string, ComponentHealth>;
  };
}

export interface ComponentHealth {
  status: 'healthy' | 'unhealthy' | 'unknown';
  latencyMs?: number;
  lastCheck?: string;
  error?: string;
}

export interface GatewayStatus {
  uptimeSeconds: number;
  requestsPerSecond: number;
  activeConnections: number;
  requestQueueDepth: number;
  circuitBreakerStatus: Record<string, CircuitBreakerState>;
}
