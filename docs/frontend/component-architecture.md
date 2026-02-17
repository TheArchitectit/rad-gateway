# RAD Gateway Admin UI - Component Architecture

**Version:** 1.0.0
**Date:** 2026-02-17
**Author:** Component Architect (The Pessimist)
**Status:** Phase 2 Skeleton Architecture

---

## 1. Executive Summary

This document defines the component architecture for the RAD Gateway Admin UI. This is a **skeleton architecture** for Phase 2 - functional but without CSS polish. The design prioritizes maintainability, testability, and error resilience over visual perfection.

**Architecture Philosophy:**
- Start simple, add complexity only when necessary
- Error boundaries at every level
- Lazy loading for code splitting
- Strong typing throughout
- Testable component design

---

## 2. Architecture Decisions & Warnings

### 2.1 Framework Choice: React 18 + Vite

**Decision:** React 18 with Vite for build tooling

**Warnings:**
- **Concurrent Mode Risks:** React 18's concurrent features can expose race conditions. Start with `React.StrictMode` disabled in dev to avoid double-rendering issues during initial development.
- **Vite Plugin Hell:** Many Vite plugins have version conflicts. Lock versions early.
- **SSR Complexity:** If SSR is considered later, the current SPA structure will require significant refactoring.

### 2.2 State Management: Zustand + TanStack Query

**Decision:** Zustand for client state, TanStack Query for server state

**Warnings:**
- **Cache Invalidation Trap:** TanStack Query's cache invalidation is powerful but error-prone. Define query keys as constants, never inline.
- **Zustand DevTools:** Without Redux DevTools, debugging complex state flows becomes painful. Include Zustand's devtools middleware from day one.
- **Over-Storing:** Resist putting everything in global state. Form state stays local. UI state (sidebar, modals) can be global.

**Alternative Rejected:** Redux Toolkit - too boilerplate-heavy for Phase 2

### 2.3 Routing: React Router v6

**Decision:** React Router v6 with nested route definitions

**Warnings:**
- **Outlet Confusion:** Nested routes with `Outlet` can become hard to trace. Document the route tree visually.
- **Loader/Action Pattern:** RR6's loaders/actions look clean but add complexity. Consider using TanStack Query instead for data fetching to keep concerns separated.

---

## 3. Component Hierarchy (Atomic Design)

### 3.1 Component Tiers

```
┌─────────────────────────────────────────────────────────────────┐
│                         PAGES (Routes)                          │
│  LoginPage | DashboardPage | ProvidersPage | TracesPage | ...   │
├─────────────────────────────────────────────────────────────────┤
│                      TEMPLATES (Layouts)                        │
│  AppLayout | AuthLayout | ModalLayout | EmptyLayout             │
├─────────────────────────────────────────────────────────────────┤
│                      ORGANISMS (Complex UI)                       │
│  Sidebar | TopNavigation | DataTable | ControlRoomGrid          │
│  ProviderCard | TraceTimeline | MetricChart | FilterPanel       │
├─────────────────────────────────────────────────────────────────┤
│                      MOLECULES (Composed)                         │
│  FormField | SearchBar | Pagination | StatusBadge                 │
│  MetricCard | ChartContainer | ListItem | ActionMenu              │
├─────────────────────────────────────────────────────────────────┤
│                       ATOMS (Primitives)                          │
│  Button | Input | Select | Card | Badge | Spinner                 │
│  Icon | Text | Heading | Divider | Skeleton                       │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 File Structure

```
web/
├── src/
│   ├── main.tsx                 # Entry point
│   ├── App.tsx                  # Root with providers
│   ├── routes.tsx               # Route definitions
│   │
│   ├── components/              # All components
│   │   ├── atoms/               # Primitive components
│   │   │   ├── Button/
│   │   │   │   ├── Button.tsx
│   │   │   │   ├── Button.types.ts
│   │   │   │   └── index.ts
│   │   │   ├── Input/
│   │   │   ├── Select/
│   │   │   ├── Card/
│   │   │   ├── Badge/
│   │   │   ├── Spinner/
│   │   │   ├── Skeleton/
│   │   │   └── index.ts         # Barrel export
│   │   │
│   │   ├── molecules/           # Composed components
│   │   │   ├── FormField/
│   │   │   ├── SearchBar/
│   │   │   ├── Pagination/
│   │   │   ├── StatusBadge/
│   │   │   ├── MetricCard/
│   │   │   ├── EmptyState/
│   │   │   └── index.ts
│   │   │
│   │   ├── organisms/           # Complex UI sections
│   │   │   ├── Sidebar/
│   │   │   ├── TopNavigation/
│   │   │   ├── DataTable/
│   │   │   ├── ControlRoomGrid/
│   │   │   ├── ProviderList/
│   │   │   ├── TraceExplorer/
│   │   │   ├── FilterPanel/
│   │   │   └── index.ts
│   │   │
│   │   └── templates/             # Layout components
│   │       ├── AppLayout/
│   │       ├── AuthLayout/
│   │       └── index.ts
│   │
│   ├── pages/                     # Route components
│   │   ├── LoginPage/
│   │   ├── DashboardPage/
│   │   ├── ControlRoomsPage/
│   │   ├── ProvidersPage/
│   │   ├── TracesPage/
│   │   ├── UsagePage/
│   │   ├── SettingsPage/
│   │   └── index.ts
│   │
│   ├── hooks/                     # Custom hooks
│   │   ├── useAuth.ts
│   │   ├── useWebSocket.ts
│   │   ├── useWorkspace.ts
│   │   └── index.ts
│   │
│   ├── stores/                    # Zustand stores
│   │   ├── authStore.ts
│   │   ├── uiStore.ts
│   │   ├── workspaceStore.ts
│   │   └── index.ts
│   │
│   ├── api/                       # API clients
│   │   ├── client.ts              # Axios/fetch instance
│   │   ├── adminApi.ts            # Admin API methods
│   │   ├── websocket.ts           # WebSocket client
│   │   └── index.ts
│   │
│   ├── types/                     # Global TypeScript types
│   │   ├── api.ts
│   │   ├── models.ts
│   │   └── index.ts
│   │
│   ├── utils/                     # Utility functions
│   │   ├── formatters.ts
│   │   ├── validators.ts
│   │   └── index.ts
│   │
│   └── styles/                    # Global styles
│       └── index.css
│
├── index.html
├── vite.config.ts
├── tsconfig.json
└── package.json
```

### 3.3 Component Naming Conventions

| Tier | Naming | Example | File |
|------|--------|---------|------|
| Atoms | PascalCase | `Button` | `Button.tsx` |
| Molecules | PascalCase + Domain | `FormField`, `SearchBar` | `FormField.tsx` |
| Organisms | PascalCase + Domain | `DataTable`, `Sidebar` | `DataTable.tsx` |
| Templates | PascalCase + Layout | `AppLayout` | `AppLayout.tsx` |
| Pages | PascalCase + Page | `DashboardPage` | `DashboardPage.tsx` |
| Hooks | camelCase + use | `useAuth` | `useAuth.ts` |
| Stores | camelCase + Store | `authStore` | `authStore.ts` |

**Warning:** Inconsistent naming leads to import confusion. Enforce with ESLint.

---

## 4. Routing Structure

### 4.1 Route Table

```typescript
// routes.tsx - Complete route definition

import { createBrowserRouter, Navigate } from 'react-router-dom';
import { AppLayout } from './components/templates';
import { AuthLayout } from './components/templates';

// Lazy-loaded pages (code splitting)
const LoginPage = lazy(() => import('./pages/LoginPage'));
const DashboardPage = lazy(() => import('./pages/DashboardPage'));
const ControlRoomsPage = lazy(() => import('./pages/ControlRoomsPage'));
const ProvidersPage = lazy(() => import('./pages/ProvidersPage'));
const ProviderDetailPage = lazy(() => import('./pages/ProviderDetailPage'));
const TracesPage = lazy(() => import('./pages/TracesPage'));
const TraceDetailPage = lazy(() => import('./pages/TraceDetailPage'));
const UsagePage = lazy(() => import('./pages/UsagePage'));
const APIKeysPage = lazy(() => import('./pages/APIKeysPage'));
const TagsPage = lazy(() => import('./pages/TagsPage'));
const UsersPage = lazy(() => import('./pages/UsersPage'));
const WorkspacesPage = lazy(() => import('./pages/WorkspacesPage'));
const SettingsPage = lazy(() => import('./pages/SettingsPage'));
const NotFoundPage = lazy(() => import('./pages/NotFoundPage'));

export const router = createBrowserRouter([
  // Public routes
  {
    path: '/auth',
    element: <AuthLayout />,
    children: [
      { path: 'login', element: <LoginPage /> },
      { path: '', element: <Navigate to="login" replace /> },
    ],
  },

  // Protected app routes
  {
    path: '/',
    element: <ProtectedRoute><AppLayout /></ProtectedRoute>,
    errorElement: <ErrorBoundaryFallback />,
    children: [
      // Dashboard section
      { index: true, element: <DashboardPage /> },
      { path: 'control-rooms', element: <ControlRoomsPage /> },
      { path: 'control-rooms/:id', element: <ControlRoomDetailPage /> },

      // Operations section
      { path: 'operations/live', element: <LiveTrafficPage /> },
      { path: 'operations/traces', element: <TracesPage /> },
      { path: 'operations/traces/:traceId', element: <TraceDetailPage /> },
      { path: 'operations/health', element: <SystemHealthPage /> },

      // Resources section
      { path: 'resources/providers', element: <ProvidersPage /> },
      { path: 'resources/providers/:name', element: <ProviderDetailPage /> },
      { path: 'resources/api-keys', element: <APIKeysPage /> },
      { path: 'resources/tags', element: <TagsPage /> },

      // Analytics section
      { path: 'analytics/usage', element: <UsagePage /> },
      { path: 'analytics/costs', element: <CostAnalysisPage /> },
      { path: 'analytics/budgets', element: <BudgetsPage /> },
      { path: 'analytics/models', element: <ModelComparisonPage /> },

      // Administration section
      { path: 'admin/users', element: <UsersPage /> },
      { path: 'admin/workspaces', element: <WorkspacesPage /> },
      { path: 'admin/webhooks', element: <WebhooksPage /> },
      { path: 'admin/settings', element: <SettingsPage /> },
    ],
  },

  // Catch-all
  { path: '*', element: <NotFoundPage /> },
]);
```

### 4.2 Route Categories

| Category | Path Pattern | Access Level | Lazy Load |
|----------|-------------|--------------|-----------|
| Public | `/auth/*` | Anonymous | Yes |
| Dashboard | `/`, `/control-rooms/*` | Authenticated | Yes |
| Operations | `/operations/*` | Authenticated | Yes |
| Resources | `/resources/*` | Authenticated | Yes |
| Analytics | `/analytics/*` | Authenticated | Yes |
| Admin | `/admin/*` | Admin only | Yes |

**Warning:** Deep nesting (`/a/b/c/d`) creates navigation complexity. Prefer flatter structures.

---

## 5. State Management Architecture

### 5.1 Store Separation

```typescript
// Three separate stores, not one monolith

// 1. Auth Store - minimal, persistent
interface AuthStore {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  login: (credentials: Credentials) => Promise<void>;
  logout: () => void;
}

// 2. UI Store - transient UI state
interface UIStore {
  sidebarCollapsed: boolean;
  theme: 'light' | 'dark' | 'system';
  activeModal: string | null;
  globalSearchOpen: boolean;
  toggleSidebar: () => void;
  setTheme: (theme: Theme) => void;
}

// 3. Workspace Store - current context
interface WorkspaceStore {
  current: Workspace | null;
  recent: Workspace[];
  favorites: Set<string>;
  setCurrent: (workspace: Workspace) => void;
  addToFavorites: (id: string) => void;
}
```

**Warning Against Monolith Store:**
```typescript
// DON'T DO THIS - causes unnecessary re-renders
interface BadRootStore {
  auth: AuthState;      // Changes on every token refresh
  ui: UIState;          // Changes on every sidebar toggle
  workspace: WorkspaceState; // Changes frequently
  // Any change triggers re-render in all subscribers
}
```

### 5.2 TanStack Query Configuration

```typescript
// Query key factory - CRITICAL for cache management
export const queryKeys = {
  all: ['rad'] as const,
  workspaces: () => [...queryKeys.all, 'workspaces'] as const,
  workspace: (id: string) => [...queryKeys.workspaces(), id] as const,
  providers: () => [...queryKeys.all, 'providers'] as const,
  provider: (name: string) => [...queryKeys.providers(), name] as const,
  apiKeys: () => [...queryKeys.all, 'api-keys'] as const,
  usage: (filters: UsageFilters) => [...queryKeys.all, 'usage', filters] as const,
  traces: (filters: TraceFilters) => [...queryKeys.all, 'traces', filters] as const,
  controlRooms: () => [...queryKeys.all, 'control-rooms'] as const,
};

// Query client configuration
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000,        // 30 seconds
      gcTime: 5 * 60 * 1000,       // 5 minutes (formerly cacheTime)
      retry: 2,
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 30000),
      refetchOnWindowFocus: false, // Annoying in dev, consider for prod
    },
  },
});
```

### 5.3 WebSocket State Integration

```typescript
// WebSocket events update TanStack Query cache directly
// No separate WebSocket store needed

const useRealtimeUpdates = () => {
  const queryClient = useQueryClient();

  useEffect(() => {
    const ws = new WebSocket(WS_URL);

    ws.onmessage = (event) => {
      const update = JSON.parse(event.data);

      // Update query cache directly
      switch (update.type) {
        case 'provider:health':
          queryClient.invalidateQueries({ queryKey: queryKeys.providers() });
          break;
        case 'usage:realtime':
          queryClient.setQueryData(
            queryKeys.usage({ realtime: true }),
            (old) => mergeRealtimeData(old, update.payload)
          );
          break;
        // ... other cases
      }
    };

    return () => ws.close();
  }, [queryClient]);
};
```

**Warning:** WebSocket reconnection logic is complex. Implement exponential backoff and message buffering during disconnections.

---

## 6. Error Boundaries Strategy

### 6.1 Hierarchy of Error Boundaries

```
App (ErrorBoundary - catches all)
├── AuthLayout (no boundary - failures show App boundary)
└── AppLayout (ErrorBoundary - catches layout errors)
    ├── Sidebar (ErrorBoundary - isolated failure)
    ├── TopNavigation (ErrorBoundary - isolated failure)
    └── MainContent (ErrorBoundary - page-level)
        ├── DashboardPage (ErrorBoundary - page-specific)
        ├── ProvidersPage (ErrorBoundary - page-specific)
        └── etc.
```

### 6.2 Error Boundary Implementation

```typescript
// components/atoms/ErrorBoundary/ErrorBoundary.tsx

import { Component, type ReactNode, type ErrorInfo } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  resetKeys?: Array<string | number>;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // Log to error tracking service
    console.error('ErrorBoundary caught:', error, errorInfo);
    this.props.onError?.(error, errorInfo);
  }

  componentDidUpdate(prevProps: Props) {
    // Reset error state when resetKeys change
    if (this.state.hasError && this.props.resetKeys) {
      const hasResetKeyChanged = this.props.resetKeys.some(
        (key, index) => key !== prevProps.resetKeys?.[index]
      );
      if (hasResetKeyChanged) {
        this.setState({ hasError: false, error: null });
      }
    }
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }
      return <DefaultErrorFallback error={this.state.error} />;
    }

    return this.props.children;
  }
}

// Default fallback UI
function DefaultErrorFallback({ error }: { error: Error | null }) {
  return (
    <div role="alert" className="error-fallback">
      <h2>Something went wrong</h2>
      <details>
        <summary>Error details</summary>
        <pre>{error?.message}</pre>
      </details>
      <button onClick={() => window.location.reload()}>
        Reload page
      </button>
    </div>
  );
}
```

### 6.3 Usage Pattern

```typescript
// In route definitions
<Route
  path="control-rooms"
  element={
    <ErrorBoundary
      fallback={<ControlRoomsError />}
      resetKeys={[location.pathname]}
    >
      <ControlRoomsPage />
    </ErrorBoundary>
  }
/>

// In complex organisms
<ErrorBoundary fallback={<SidebarError />}
  <Sidebar />
</ErrorBoundary>
```

---

## 7. Component Specifications

### 7.1 Atoms (Phase 2 Skeleton)

| Component | Props | Edge Cases | Failure Mode |
|-----------|-------|------------|--------------|
| `Button` | `variant`, `size`, `loading`, `disabled` | Rapid clicks, async handlers | Debounce clicks, disable while loading |
| `Input` | `type`, `value`, `onChange`, `error` | Empty strings, null values | Controlled component only, never uncontrolled |
| `Select` | `options`, `value`, `onChange` | No options, duplicate values | Show empty state, dedupe options |
| `Card` | `children`, `title`, `actions` | Overflow content | Scroll or truncate, configurable |
| `Badge` | `variant`, `children` | Empty content | Hide when empty |
| `Spinner` | `size` | Multiple spinners cause visual noise | Centralize loading states |
| `Skeleton` | `width`, `height`, `count` | Layout shift on load | Reserve space with aspect ratio |

### 7.2 Molecules (Phase 2 Skeleton)

| Component | Props | Edge Cases | Failure Mode |
|-----------|-------|------------|--------------|
| `FormField` | `label`, `error`, `children` | Long labels, multiple errors | Truncate labels, show first error |
| `SearchBar` | `value`, `onChange`, `onSearch` | Empty search, special chars | Debounce input, escape regex |
| `Pagination` | `page`, `total`, `onChange` | Zero items, large totals | Hide when 1 page, abbreviate large numbers |
| `StatusBadge` | `status`, `text` | Unknown status values | Default to 'unknown' style |
| `MetricCard` | `title`, `value`, `trend` | Null values, overflow | Show '—' for null, abbreviate large numbers |
| `EmptyState` | `title`, `description`, `action` | Missing action | Hide action button when not provided |

### 7.3 Organisms (Phase 2 Skeleton)

| Component | Props | Edge Cases | Failure Mode |
|-----------|-------|------------|--------------|
| `Sidebar` | `items`, `collapsed` | Too many items | Scroll, collapse groups |
| `TopNavigation` | `workspace`, `user` | Missing user | Show loading state |
| `DataTable` | `columns`, `data`, `sorting` | Empty data, many columns | Empty state, horizontal scroll |
| `ProviderList` | `providers`, `onSelect` | Empty list | Empty state with CTA |
| `FilterPanel` | `filters`, `onChange` | Many filters | Collapsible groups |

---

## 8. Edge Cases & Failure Modes

### 8.1 Authentication Failures

| Scenario | Behavior | Recovery |
|----------|----------|----------|
| Token expired mid-session | Redirect to login, preserve route | Auto-redirect back after login |
| 401 on API call | Clear token, show auth error | Manual re-login required |
| 403 (forbidden) | Show permission error | Contact admin CTA |
| Token refresh fails | Logout user, show error | Re-authenticate |

### 8.2 API Failures

| Scenario | Behavior | Recovery |
|----------|----------|----------|
| Network offline | Show offline indicator | Auto-retry with backoff |
| Timeout | Show timeout error | Manual retry button |
| 5xx errors | Show service unavailable | Retry with exponential backoff |
| Rate limited (429) | Show rate limit message | Auto-retry after Retry-After |
| Partial data load | Show partial success | Continue with available data |

### 8.3 WebSocket Failures

| Scenario | Behavior | Recovery |
|----------|----------|----------|
| Connection lost | Show disconnected indicator | Auto-reconnect with backoff |
| Message parsing error | Log error, continue | Alert if pattern emerges |
| Reconnection storm | Backoff to max delay | Circuit breaker pattern |
| Auth failure on WS | Close connection, redirect | Re-authenticate |

### 8.4 UI Edge Cases

| Scenario | Behavior | Prevention |
|----------|----------|------------|
| Very long workspace names | Truncate with ellipsis | Max length validation |
| Thousands of providers | Virtual scroll | Implement windowing |
| Complex filter combinations | Show active filters | Clear all button |
| Dashboard with many widgets | Paginate or limit | Max widgets constraint |
| Mobile viewport | Responsive layout | Mobile-first CSS |
| Slow network | Skeleton screens | Perceived performance |

### 8.5 Data Edge Cases

| Scenario | Behavior | Prevention |
|----------|----------|------------|
| Null values in data | Show placeholder | Schema validation |
| Invalid date formats | Show 'Invalid date' | Centralized date parsing |
| Very large numbers | Abbreviate (1.2M) | Number formatting utility |
| Missing nested properties | Optional chaining | TypeScript strict mode |
| Circular references in JSON | Handle in serialization | Custom JSON replacer |

---

## 9. Testing Strategy

### 9.1 Component Testing Pyramid

```
        ┌─────────┐
        │   E2E   │  <- Full user flows (10%)
        │  (Cypress)│
       ┌┴─────────┴┐
       │ Integration│ <- Component interactions (20%)
       │ (RTL+MSW) │
      ┌┴───────────┴┐
      │    Unit      │ <- Component logic (70%)
      │    (Vitest)  │
      └──────────────┘
```

### 9.2 Critical Test Cases

| Component | Critical Test |
|-----------|---------------|
| `ErrorBoundary` | Renders fallback on error, resets on key change |
| `ProtectedRoute` | Redirects unauthenticated, allows authenticated |
| `useWebSocket` | Reconnects on disconnect, buffers messages |
| `DataTable` | Sorts, filters, paginates correctly |
| `LoginPage` | Submits credentials, handles errors |

### 9.3 Mock Service Worker Setup

```typescript
// mocks/handlers.ts
import { http, HttpResponse } from 'msw';

export const handlers = [
  http.get('/v0/admin/providers', () => {
    return HttpResponse.json({
      providers: [
        { name: 'openai', status: 'healthy', circuitBreaker: 'closed' },
        { name: 'anthropic', status: 'degraded', circuitBreaker: 'closed' },
      ],
    });
  }),

  http.get('/v0/admin/health', () => {
    return HttpResponse.json({
      status: 'ok',
      version: '0.1.0',
    });
  }),
];
```

---

## 10. Performance Considerations

### 10.1 Bundle Size Budgets

| Category | Budget | Warning At |
|----------|--------|------------|
| Initial bundle | < 200KB | 180KB |
| Async chunks | < 100KB each | - |
| Total lazy loaded | < 500KB | 450KB |

### 10.2 Optimization Strategies

1. **Code Splitting:** Each page is a separate chunk
2. **Tree Shaking:** Import only needed functions
3. **Lazy Loading:** Routes, heavy components
4. **Memoization:** React.memo for list items, useMemo for expensive calc
5. **Virtualization:** react-window for long lists

### 10.3 Over-Optimization Warnings

**Don't optimize prematurely:**
- Don't memoize everything (costs more than it saves)
- Don't split too granularly (HTTP overhead)
- Don't use useEffect for derived state

---

## 11. Accessibility Requirements

### 11.1 Phase 2 Minimums

- All interactive elements keyboard accessible
- Focus visible on all focusable elements
- Alt text on all images
- ARIA labels where visual label missing
- Color contrast 4.5:1 minimum

### 11.2 Component Checklist

```typescript
// Button component with accessibility
interface ButtonProps {
  children: React.ReactNode;
  variant?: 'primary' | 'secondary';
  disabled?: boolean;
  loading?: boolean;
  onClick?: () => void;
  'aria-label'?: string;  // Required if no text
}

function Button({
  children,
  variant = 'primary',
  disabled,
  loading,
  onClick,
  'aria-label': ariaLabel,
}: ButtonProps) {
  return (
    <button
      type="button"
      disabled={disabled || loading}
      onClick={onClick}
      aria-label={ariaLabel}
      aria-busy={loading}
      className={`button button--${variant}`}
    >
      {loading && <Spinner size="small" aria-hidden="true" />}
      {children}
    </button>
  );
}
```

---

## 12. Migration Path

### 12.1 Phase 2 -> Phase 3

Phase 2 (Skeleton) delivers:
- [ ] All pages with basic layout
- [ ] Routing and navigation working
- [ ] API integration with TanStack Query
- [ ] Basic forms with validation
- [ ] Error boundaries throughout
- [ ] Auth flow complete

Phase 3 (Core Features) adds:
- [ ] Real-time WebSocket integration
- [ ] Control room dashboards
- [ ] Provider management
- [ ] API key lifecycle
- [ ] Usage analytics (basic)
- [ ] Trace explorer (basic)

Phase 4 (Advanced) adds:
- [ ] Drag-and-drop dashboard builder
- [ ] Custom visualizations
- [ ] Query builder
- [ ] Advanced filtering

### 12.2 Breaking Changes to Anticipate

| Change | Impact | Mitigation |
|--------|--------|------------|
| Store restructuring | All components using store | Use selectors, not direct access |
| API version change | All API calls | Centralize in api/ layer |
| Route changes | All Links and navigations | Use route constants |
| Component API change | Parent components | Deprecate before removing |

---

## 13. Anti-Patterns to Avoid

### 13.1 State Management

```typescript
// DON'T: Prop drilling
<App>
  <Layout user={user}>
    <Sidebar user={user}>
      <Nav user={user}>
        <UserMenu user={user} />  // Too deep!

// DO: Use context or store
<App>
  <Layout>
    <Sidebar>
      <Nav>
        <UserMenu />  // Gets user from store
```

### 13.2 Effect Usage

```typescript
// DON'T: useEffect for derived state
useEffect(() => {
  setFullName(`${firstName} ${lastName}`);
}, [firstName, lastName]);

// DO: Compute directly
const fullName = `${firstName} ${lastName}`;
```

### 13.3 Async Handling

```typescript
// DON'T: Floating promises
useEffect(() => {
  fetchData();  // No error handling!
}, []);

// DO: Proper error handling
useEffect(() => {
  let cancelled = false;

  const load = async () => {
    try {
      const data = await fetchData();
      if (!cancelled) setData(data);
    } catch (err) {
      if (!cancelled) setError(err);
    }
  };

  load();
  return () => { cancelled = true; };
}, []);
```

### 13.4 Component Design

```typescript
// DON'T: God components
function DashboardPage() {
  // 500 lines of everything
}

// DO: Compose smaller components
function DashboardPage() {
  return (
    <DashboardLayout>
      <MetricsSection />
      <ChartsSection />
      <RecentActivity />
    </DashboardLayout>
  );
}
```

---

## 14. Appendix: Quick Reference

### 14.1 File Templates

**New Component:**
```typescript
// components/[tier]/ComponentName/ComponentName.tsx
import type { ComponentNameProps } from './ComponentName.types';

export function ComponentName({ prop1, prop2 }: ComponentNameProps) {
  return <div>{/* implementation */}</div>;
}
```

**New Hook:**
```typescript
// hooks/useHookName.ts
import { useState, useEffect } from 'react';

export function useHookName(param: string) {
  const [state, setState] = useState(null);
  // implementation
  return state;
}
```

**New Store:**
```typescript
// stores/storeName.ts
import { create } from 'zustand';

interface StoreNameState {
  value: string;
  setValue: (v: string) => void;
}

export const useStoreName = create<StoreNameState>((set) => ({
  value: '',
  setValue: (v) => set({ value: v }),
}));
```

### 14.2 Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-02-17 | Zustand over Redux | Less boilerplate, sufficient for Phase 2 |
| 2026-02-17 | TanStack Query for server state | Caching, error handling built-in |
| 2026-02-17 | React Router v6 | Industry standard, nested routes |
| 2026-02-17 | Separate stores vs monolith | Avoids unnecessary re-renders |
| 2026-02-17 | Atomic design structure | Clear organization, scalable |

---

**Document Status:** Ready for implementation
**Next Steps:**
1. Set up project with Vite + React + TypeScript
2. Install dependencies (Zustand, TanStack Query, React Router)
3. Create base components (atoms)
4. Implement routing structure
5. Add error boundaries
6. Implement auth flow

**Author's Warning:** This architecture will evolve. Don't treat it as immutable. Adapt as requirements change, but document changes in the decision log above.
