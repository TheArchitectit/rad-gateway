# RAD Gateway Admin UI - Frontend Feature Specification

**Version:** 1.0.0
**Date:** 2026-02-17
**Status:** Draft for Review
**Author:** Frontend Feature Designer

---

## 1. Executive Summary

This document defines the frontend architecture and feature set for the RAD Gateway Admin Dashboard. The UI will provide a modern, real-time interface for managing AI API infrastructure with exceptional user experience through live updates, drag-and-drop customization, and rich data visualizations.

### Key Differentiators (Super Power Features)

1. **Live Control Rooms** - Real-time operational dashboards with WebSocket updates
2. **Visual Provider Mesh** - Interactive topology visualization of provider routing
3. **Intelligent Cost Forecasting** - ML-powered cost prediction with budget alerts
4. **Drag-and-Drop Dashboard Builder** - Customizable widget-based layouts
5. **Trace Timeline Explorer** - Visual request tracing with waterfall analysis

---

## 2. Feature List (FL-001 to FL-015)

### FL-001: Multi-Workspace Navigation
**Priority:** P0
**Description:** Workspace switcher with cross-workspace visibility for admins
**Acceptance Criteria:**
- Workspace dropdown with search and favorites
- Visual indicators for active workspace
- Recent workspaces quick access
- Workspace creation wizard for admins

### FL-002: Role-Based Access Control (RBAC) UI
**Priority:** P0
**Description:** Visual permission matrix and role management
**Acceptance Criteria:**
- Role assignment interface (Admin/Developer/Viewer)
- Permission grid showing access levels
- User management with invite workflows
- Role preview mode (see UI as different role)

### FL-003: Real-Time Control Room Dashboard
**Priority:** P0
**Super Power:** WebSocket-powered live updates
**Description:** Configurable operational views with tag-based filtering
**Acceptance Criteria:**
- Create/edit/delete control rooms
- Tag-based resource filtering with autocomplete
- Real-time metrics streaming (latency, throughput, errors)
- Multiple layout presets (grid, list, compact)
- Drag-and-drop widget repositioning

### FL-004: Interactive Provider Management
**Priority:** P0
**Super Power:** Visual provider mesh topology
**Description:** Manage AI providers with health monitoring and circuit breaker status
**Acceptance Criteria:**
- Provider cards with live health indicators
- Circuit breaker state visualization (closed/open/half-open)
- Provider comparison matrix (cost, latency, reliability)
- Visual routing topology showing provider relationships
- One-click provider failover simulation

### FL-005: API Key Lifecycle Management
**Priority:** P0
**Description:** Complete API key CRUD with usage tracking
**Acceptance Criteria:**
- Key generation with customizable permissions
- Visual usage charts per key
- Rate limit configuration with burst visualization
- Key rotation workflow with expiration reminders
- Copy-to-clipboard with secure reveal toggle

### FL-006: Usage Analytics & Reporting
**Priority:** P0
**Super Power:** ML-powered cost forecasting
**Description:** Comprehensive usage dashboards with exportable reports
**Acceptance Criteria:**
- Time-series charts (hourly, daily, monthly)
- Token usage breakdown by model/provider
- Cost analysis with currency conversion
- Predictive cost forecasting
- Scheduled report generation with email delivery
- Custom date range selection with presets

### FL-007: Request Trace Explorer
**Priority:** P0
**Super Power:** Visual waterfall timeline
**Description:** Detailed request tracing with provider routing visualization
**Acceptance Criteria:**
- Trace list with filtering (status, model, provider, time)
- Waterfall timeline view showing request phases
- Provider routing visualization with retry indicators
- Payload inspection (request/response)
- Trace comparison side-by-side
- Export to JSON/Curl commands

### FL-008: Cost Budget Management
**Priority:** P1
**Description:** Budget setup with alerting and enforcement
**Acceptance Criteria:**
- Budget creation per workspace/project/tag
- Visual budget burn-down charts
- Alert configuration (email, webhook, Slack)
- Hard/soft limit enforcement options
- Budget forecasting with anomaly detection

### FL-009: Model Performance Comparison
**Priority:** P1
**Super Power:** A/B test results visualization
**Description:** Compare model performance across providers
**Acceptance Criteria:**
- Side-by-side model comparison
- Latency distribution histograms
- Cost per token analysis
- Quality metrics integration (coming from evaluation API)
- Recommended model suggestions

### FL-010: Tag Management System
**Priority:** P1
**Description:** Hierarchical tag creation and assignment
**Acceptance Criteria:**
- Tag creation with category:value format validation
- Bulk tag assignment to resources
- Tag-based filtering across all views
- Tag usage analytics
- Tag color coding and icons

### FL-011: Real-Time Notifications Center
**Priority:** P1
**Super Power:** WebSocket event streaming
**Description:** Centralized notification hub for system events
**Acceptance Criteria:**
- Notification bell with unread count
- Event categories (alerts, warnings, info)
- Notification preferences per channel
- Historical event log with search
- Webhook integration for external notifications

### FL-012: System Health Overview
**Priority:** P1
**Description:** High-level system status with component health
**Acceptance Criteria:**
- Service status grid (gateway, providers, database)
- Historical uptime graphs
- Incident timeline
- Health check details modal
- RSS feed for status page integration

### FL-013: Advanced Query Builder
**Priority:** P2
**Super Power:** Natural language query interface
**Description:** Build complex queries for usage/traces without SQL
**Acceptance Criteria:**
- Visual query builder with drag-and-drop conditions
- Natural language to query conversion
- Saved queries with sharing
- Query result visualization options
- Export to CSV/JSON/Excel

### FL-014: Custom Dashboard Builder
**Priority:** P2
**Super Power:** Widget marketplace
**Description:** User-created dashboards with custom widgets
**Acceptance Criteria:**
- Blank canvas or template-based creation
- Widget library (charts, metrics, tables, text)
- Custom widget creation (JSON/YAML config)
- Dashboard sharing and embedding
- Scheduled dashboard snapshots (PDF/PNG)

### FL-015: Settings & Configuration
**Priority:** P1
**Description:** Workspace and personal settings management
**Acceptance Criteria:**
- Workspace configuration (name, branding, defaults)
- User profile with theme preferences
- API endpoint configuration
- Webhook management
- Data retention policies
- Import/export settings

---

## 3. Page Structure & Navigation

### Primary Navigation Structure

```
┌─────────────────────────────────────────────────────────────┐
│  RAD Gateway Admin UI                                        │
├──────────┬──────────────────────────────────────────────────┤
│          │                                                  │
│  LOGO    │  [Workspace Selector ▼]     [Notifications] [User ▼]
│          │                                                  │
├──────────┼──────────────────────────────────────────────────┤
│          │                                                  │
│  DASHBOARD                                                  │
│  ├── Overview                                               │
│  ├── Control Rooms                                          │
│  └── Custom Dashboards                                      │
│                                                             │
│  OPERATIONS                                                 │
│  ├── Live Traffic                                           │
│  ├── Request Traces                                         │
│  └── System Health                                          │
│                                                             │
│  RESOURCES                                                  │
│  ├── Providers                                              │
│  ├── API Keys                                               │
│  └── Tags                                                   │
│                                                             │
│  ANALYTICS                                                  │
│  ├── Usage Reports                                          │
│  ├── Cost Analysis                                          │
│  ├── Budgets                                                │
│  └── Model Comparison                                       │
│                                                             │
│  ADMINISTRATION                                             │
│  ├── Users & Roles                                          │
│  ├── Workspaces                                             │
│  ├── Webhooks                                               │
│  └── Settings                                               │
│                                                             │
│  [Help & Support]                                           │
│                                                             │
└──────────┴──────────────────────────────────────────────────┘
```

### Page Details

#### 3.1 Dashboard Section

**Overview (/)**
- Default landing page with summary widgets
- Quick stats cards (requests today, cost, active providers)
- Recent activity feed
- Favorite control rooms preview
- Alerts requiring attention

**Control Rooms (/control-rooms)**
- Grid of available control rooms
- Create new button with template selection
- Quick filter by tags
- Real-time connection status indicator

**Custom Dashboards (/dashboards)**
- List of user-created dashboards
- Create from template or blank
- Shared dashboards from team

#### 3.2 Operations Section

**Live Traffic (/operations/live)**
- Real-time request stream
- Request volume sparklines
- Active connection counter
- Geographic request distribution map

**Request Traces (/operations/traces)**
- Advanced filtering sidebar
- Trace list with key metrics
- Waterfall detail view
- Export options

**System Health (/operations/health)**
- Component status grid
- Latency heatmap by provider
- Error rate trends
- Incident history

#### 3.3 Resources Section

**Providers (/resources/providers)**
- Provider cards/grid view toggle
- Health status indicators
- Circuit breaker controls
- Routing configuration

**API Keys (/resources/api-keys)**
- Key table with usage stats
- Create key wizard
- Permission matrix
- Rotation schedule

**Tags (/resources/tags)**
- Tag hierarchy browser
- Bulk assignment tools
- Usage statistics

#### 3.4 Analytics Section

**Usage Reports (/analytics/usage)**
- Time range selector
- Multi-metric charts
- Provider breakdown
- Export scheduler

**Cost Analysis (/analytics/costs)**
- Cost trends with forecasting
- Budget comparison
- Cost by tag/project breakdown
- Invoice generation preview

**Budgets (/analytics/budgets)**
- Budget list with status
- Burn-down charts
- Alert history
- Creation wizard

**Model Comparison (/analytics/models)**
- Performance comparison charts
- Cost-effectiveness matrix
- Recommendation engine results

#### 3.5 Administration Section

**Users & Roles (/admin/users)**
- User directory with search
- Role assignment modal
- Invite user workflow
- Permission audit log

**Workspaces (/admin/workspaces)**
- Workspace list (admin view)
- Creation/deletion controls
- Resource usage by workspace
- Billing aggregation

**Webhooks (/admin/webhooks)**
- Endpoint configuration
- Event type selection
- Delivery history and retry
- Testing interface

**Settings (/admin/settings)**
- Workspace configuration
- Security settings
- Data retention
- API configuration

---

## 4. Component Hierarchy

### 4.1 Layout Components

```
Layout
├── AppShell
│   ├── TopNavigation
│   │   ├── WorkspaceSelector
│   │   ├── GlobalSearch
│   │   ├── NotificationCenter
│   │   └── UserMenu
│   ├── Sidebar
│   │   ├── NavigationGroup
│   │   │   └── NavigationItem
│   │   └── SidebarToggle
│   └── MainContent
│       └── PageContainer
│           ├── PageHeader
│           │   ├── Breadcrumbs
│           │   ├── Title
│           │   └── ActionButtons
│           └── PageContent
└── ToastContainer
```

### 4.2 Dashboard Widgets

```
Dashboard
├── DashboardGrid (react-grid-layout)
│   └── WidgetContainer
│       ├── WidgetHeader
│       │   ├── Title
│       │   ├── Actions (refresh, expand, remove)
│       │   └── DragHandle
│       └── WidgetContent
│           ├── MetricCard
│           ├── TimeSeriesChart
│           ├── PieChart
│           ├── BarChart
│           ├── DataTable
│           ├── ProviderStatusGrid
│           ├── RequestStream
│           └── CustomHTML
└── AddWidgetPanel
    └── WidgetPreviewCard
```

### 4.3 Form Components

```
Forms
├── FormContainer
│   ├── FormSection
│   │   ├── SectionTitle
│   │   ├── SectionDescription
│   │   └── FormField
│   │       ├── Label
│   │       ├── Input (text, number, email, password)
│   │       ├── Select (single, multi, searchable)
│   │       ├── DatePicker (single, range)
│   │       ├── Toggle
│   │       ├── Checkbox
│   │       ├── RadioGroup
│   │       ├── TextArea
│   │       ├── TagInput
│   │       ├── JSONEditor
│   │       └── FieldError
│   └── FormActions
│       ├── SubmitButton
│       ├── CancelButton
│       └── SecondaryActions
└── FormValidationSummary
```

### 4.4 Data Display Components

```
DataDisplay
├── DataTable
│   ├── TableToolbar
│   │   ├── SearchInput
│   │   ├── FilterChips
│   │   ├── ColumnVisibility
│   │   └── ExportButton
│   ├── TableHeader
│   │   └── SortableHeaderCell
│   ├── TableBody
│   │   └── TableRow
│   │       └── TableCell
│   │           ├── TextCell
│   │           ├── StatusBadge
│   │           ├── ProgressBar
│   │           ├── Sparkline
│   │           ├── Avatar
│   │           └── ActionsMenu
│   └── TablePagination
├── CardGrid
│   └── ResourceCard
│       ├── CardHeader
│       ├── CardBody
│       └── CardFooter
├── DetailView
│   ├── DetailHeader
│   ├── DetailTabs
│   └── DetailSections
└── EmptyState
```

### 4.5 Visualization Components

```
Visualizations
├── TimeSeriesChart (Recharts/D3)
├── PieChart
├── BarChart
├── GaugeChart
├── Heatmap
├── TopologyGraph (D3/Cytoscape)
│   ├── Node
│   └── Edge
├── WaterfallChart
├── RequestTimeline
├── GeoMap
└── Sparkline
```

### 4.6 Feedback Components

```
Feedback
├── Alert
│   ├── AlertTitle
│   └── AlertDescription
├── Modal
│   ├── ModalHeader
│   ├── ModalBody
│   └── ModalFooter
├── Drawer
├── Toast
├── Loading
│   ├── Spinner
│   ├── Skeleton
│   └── ProgressBar
├── EmptyState
├── ErrorBoundary
└── ConfirmationDialog
```

---

## 5. State Management Requirements

### 5.1 Global State (Zustand)

```typescript
// Store Structure
interface RootState {
  // Authentication & User
  auth: {
    user: User | null;
    token: string | null;
    permissions: Permission[];
    isAuthenticated: boolean;
  };

  // Workspace Context
  workspace: {
    current: Workspace | null;
    recent: Workspace[];
    favorites: string[];
    list: Workspace[];
  };

  // UI State
  ui: {
    sidebarCollapsed: boolean;
    theme: 'light' | 'dark' | 'system';
    notifications: Notification[];
    activeModal: string | null;
    globalSearchOpen: boolean;
  };

  // Real-time Connections
  realtime: {
    connected: boolean;
    subscriptions: string[];
    lastEvent: WebSocketEvent | null;
  };
}
```

### 5.2 Server State (TanStack Query)

```typescript
// Query Keys Pattern
const queryKeys = {
  workspaces: ['workspaces'] as const,
  workspace: (id: string) => ['workspaces', id] as const,
  providers: ['providers'] as const,
  provider: (id: string) => ['providers', id] as const,
  apiKeys: ['api-keys'] as const,
  usage: (filters: UsageFilters) => ['usage', filters] as const,
  traces: (filters: TraceFilters) => ['traces', filters] as const,
  controlRooms: ['control-rooms'] as const,
  budgets: ['budgets'] as const,
};
```

### 5.3 Local State Patterns

- **Form State:** React Hook Form with Zod validation
- **Table State:** URL-persisted filters, sorting, pagination
- **Dashboard Layout:** react-grid-layout with localStorage persistence
- **Widget Configuration:** Component-level state with save/load

---

## 6. API Integration Layer

### 6.1 REST API Client

```typescript
// API Client Structure
class AdminAPI {
  // Dashboard
  getOverview(): Promise<DashboardOverview>;
  getControlRooms(): Promise<ControlRoom[]>;

  // Resources
  getProviders(): Promise<Provider[]>;
  getAPIKeys(): Promise<APIKey[]>;
  createAPIKey(data: CreateAPIKeyDTO): Promise<APIKey>;

  // Analytics
  getUsage(filters: UsageFilters): Promise<UsageData>;
  getTraces(filters: TraceFilters): Promise<Trace[]>;

  // Administration
  getUsers(): Promise<User[]>;
  updateUserRole(userId: string, role: Role): Promise<User>;
}
```

### 6.2 WebSocket Integration

```typescript
// Real-time Event Types
interface WebSocketEvents {
  // Usage events
  'usage:realtime': RealtimeUsageMetrics;
  'usage:threshold': ThresholdAlert;

  // Provider events
  'provider:health': ProviderHealthUpdate;
  'provider:circuit': CircuitBreakerEvent;

  // System events
  'system:alert': SystemAlert;
  'system:notification': Notification;

  // Request events
  'request:completed': RequestCompletedEvent;
  'request:failed': RequestFailedEvent;
}
```

---

## 7. Priority Matrix

| Feature | Priority | Effort | Business Value | Technical Complexity |
|---------|----------|--------|----------------|---------------------|
| FL-001: Multi-Workspace Navigation | P0 | Low | High | Low |
| FL-002: RBAC UI | P0 | Medium | High | Medium |
| FL-003: Control Room Dashboard | P0 | High | High | High |
| FL-004: Provider Management | P0 | Medium | High | Medium |
| FL-005: API Key Management | P0 | Medium | High | Medium |
| FL-006: Usage Analytics | P0 | High | High | Medium |
| FL-007: Trace Explorer | P0 | High | High | High |
| FL-008: Budget Management | P1 | Medium | Medium | Medium |
| FL-009: Model Comparison | P1 | Medium | Medium | Medium |
| FL-010: Tag Management | P1 | Low | Medium | Low |
| FL-011: Notifications | P1 | Medium | Medium | High |
| FL-012: System Health | P1 | Low | Medium | Low |
| FL-013: Query Builder | P2 | High | Medium | High |
| FL-014: Custom Dashboards | P2 | High | Medium | High |
| FL-015: Settings | P1 | Medium | Medium | Medium |

---

## 8. Technical Requirements

### 8.1 Browser Support
- Chrome 90+, Firefox 88+, Safari 14+, Edge 90+
- Responsive: Desktop (1280px+), Tablet (768px+), Mobile (360px+)

### 8.2 Performance Targets
- First Contentful Paint: < 1.5s
- Time to Interactive: < 3s
- Lighthouse Score: > 90
- Bundle size: < 500KB initial

### 8.3 Accessibility
- WCAG 2.1 Level AA compliance
- Keyboard navigation support
- Screen reader optimized
- Color contrast ratios met

### 8.4 Security
- CSP headers
- XSS protection
- CSRF tokens
- Secure token storage

---

## 9. Implementation Phases

### Phase 1: Foundation (Weeks 1-3)
- Project setup with Next.js 14, TypeScript, Tailwind
- Core layout and navigation
- Authentication integration
- Basic API client setup

### Phase 2: Core Features (Weeks 4-7)
- Dashboard overview
- Provider management
- API key management
- Usage analytics basic

### Phase 3: Advanced Features (Weeks 8-11)
- Control rooms with real-time
- Trace explorer
- Budget management
- User/Role management

### Phase 4: Polish & Super Powers (Weeks 12-14)
- Custom dashboards
- Query builder
- Advanced visualizations
- Performance optimization

---

## 10. Appendix

### A. Design System Reference
- Component library: Radix UI primitives
- Styling: Tailwind CSS with custom design tokens
- Icons: Lucide React
- Charts: Recharts + D3 for custom visualizations

### B. File Naming Conventions
- Components: PascalCase (`ControlRoomCard.tsx`)
- Hooks: camelCase with `use` prefix (`useWebSocket.ts`)
- Utils: camelCase (`formatDate.ts`)
- Types: PascalCase with suffix (`types.ts` or `User.types.ts`)

### C. Route Conventions
- Private routes: `/app/*`
- Public routes: `/*`
- Admin-only: `/app/admin/*`

---

**Document Status:** Ready for review by Team Alpha (Architecture) and Team Golf (Documentation)
**Next Steps:** API specification alignment, Component library selection, Development environment setup
