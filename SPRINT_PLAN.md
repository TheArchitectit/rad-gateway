# RAD Gateway - 21-Day Sprint Plan (AI-Only Execution)

**Start Date**: 2026-02-19  
**End Date**: 2026-03-12  
**Total Effort**: 21 days  
**Execution Mode**: AI agents complete all work (no human intervention)

---

## Overview

This plan delivers:
1. **Complete Web UI** (Phases 1-3): Fully functional admin dashboard with all core features
2. **Real Provider Adapters** (Phase 4): OpenAI, Anthropic, Gemini integration
3. **Production Deployment** (Phase 5): Containerized, automated, monitored

**Assumption**: I execute all tasks via AI agents. Humans only approve/review.

---

## Sprint Structure

| Sprint | Focus | Days | Deliverable |
|--------|-------|------|-------------|
| **1** | Web UI Foundation | 1-3 | Component library + Layouts |
| **2** | Core Pages | 4-7 | 6 functional admin pages |
| **3** | Advanced Features | 8-10 | Real-time updates + Charts |
| **4** | Provider Adapters | 11-16 | Real AI provider integration |
| **5** | Deployment | 17-19 | Production-ready containers |
| **6** | Testing + Polish | 20-21 | E2E validation + Bug fixes |

---

## SPRINT 1: Web UI Foundation (Days 1-3)
**Goal**: Build atomic design system and application shell

### Day 1: Atomic Components (8 hours)
**Agent**: frontend-ui-ux + quick

**Morning (4h)**:
- [ ] Create `web/src/components/atoms/Button.tsx`
  - Variants: primary, secondary, danger, ghost
  - Sizes: sm, md, lg
  - States: default, hover, active, disabled, loading
- [ ] Create `web/src/components/atoms/Input.tsx`
  - Types: text, password, email, number
  - States: default, focus, error, disabled
  - Labels + Error messages
- [ ] Create `web/src/components/atoms/Select.tsx`
  - Single select with search
  - Options grouping
  - Custom rendering

**Afternoon (4h)**:
- [ ] Create `web/src/components/atoms/Card.tsx`
  - Header, body, footer sections
  - Shadow variants
  - Hover effects
- [ ] Create `web/src/components/atoms/Badge.tsx`
  - Colors: success, warning, error, info
  - Sizes: sm, md
- [ ] Create `web/src/components/atoms/Avatar.tsx`
  - Fallback initials
  - Sizes: xs, sm, md, lg, xl

**Output**: 6 atomic components with Storybook-style docs

---

### Day 2: Molecular Components (8 hours)
**Agent**: frontend-ui-ux + quick

**Morning (4h)**:
- [ ] Create `web/src/components/molecules/FormField.tsx`
  - Label + Input + Error message
  - Required indicator
  - Help text
- [ ] Create `web/src/components/molecules/SearchBar.tsx`
  - Input + Search icon + Clear button
  - Debounced search
  - Loading state
- [ ] Create `web/src/components/molecules/Pagination.tsx`
  - Previous/Next buttons
  - Page numbers with ellipsis
  - Items per page selector

**Afternoon (4h)**:
- [ ] Create `web/src/components/molecules/StatusBadge.tsx`
  - Status colors
  - Animated pulse for "in-progress"
  - Icons for each status
- [ ] Create `web/src/components/molecules/EmptyState.tsx`
  - Icon + Title + Description + CTA
  - Variants: no data, error, search empty
- [ ] Create `web/src/components/molecules/Dropdown.tsx`
  - Trigger button + Menu
  - Item groups
  - Keyboard navigation

**Output**: 6 molecular components

---

### Day 3: Layout System (8 hours)
**Agent**: frontend-ui-ux + quick

**Morning (4h)**:
- [ ] Create `web/src/components/organisms/Sidebar.tsx`
  - Logo + Navigation items
  - Collapsible sections
  - Active state highlighting
  - Icons from Lucide
- [ ] Create `web/src/components/organisms/TopNavigation.tsx`
  - Breadcrumb trail
  - User menu (avatar + dropdown)
  - Notifications bell
  - Workspace switcher

**Afternoon (4h)**:
- [ ] Create `web/src/components/templates/AppLayout.tsx`
  - Sidebar + TopNav + Content area
  - Responsive breakpoints
  - Mobile drawer
  - Loading states
- [ ] Create `web/src/components/templates/AuthLayout.tsx`
  - Centered card layout
  - Background gradient
  - Logo + Footer links
- [ ] Update `web/next.config.js` for static export

**Output**: Complete layout system ready for pages

**Sprint 1 Deliverable**: Storybook-ready component library with 15+ components

---

## SPRINT 2: Core Pages (Days 4-7)
**Goal**: Build functional admin pages consuming deployed API

### Day 4: Authentication & Dashboard (8 hours)
**Agent**: deep + playwright

**Morning (4h)**:
- [ ] Create `web/src/app/login/page.tsx`
  - Use existing LoginForm component
  - Connect to `/v1/auth/login` API
  - JWT token storage (httpOnly cookies)
  - Error handling + Loading states
  - Redirect on success

**Afternoon (4h)**:
- [ ] Create `web/src/app/page.tsx` (Dashboard)
  - Overview cards: Active providers, API calls today, Cost today
  - Quick actions: Create API key, Add provider
  - Recent activity feed
  - System health widget
  - Connect to SSE for real-time updates

**Output**: Working login + dashboard

---

### Day 5: Provider Management (8 hours)
**Agent**: deep + playwright

**Morning (4h)**:
- [ ] Create `web/src/app/providers/page.tsx` (List)
  - DataTable with providers
  - Columns: Name, Status, Health, Circuit, Cost
  - Filters: Status, Provider type
  - Sorting + Pagination
  - Add provider button

**Afternoon (4h)**:
- [ ] Create `web/src/app/providers/[id]/page.tsx` (Detail)
  - Provider info card
  - Health history chart
  - Circuit breaker controls
  - Recent requests table
  - Edit/Delete actions

**Output**: Full provider management

---

### Day 6: API Keys & Projects (8 hours)
**Agent**: deep + playwright

**Morning (4h)**:
- [ ] Create `web/src/app/api-keys/page.tsx`
  - DataTable with API keys (masked)
  - Create key modal
  - Revoke/Rotate actions
  - Copy to clipboard
  - Usage per key chart

**Afternoon (4h)**:
- [ ] Create `web/src/app/projects/page.tsx`
  - Workspace list with cards
  - Create project modal
  - Project selector
  - Settings per project

**Output**: API key + project management

---

### Day 7: Usage Analytics (8 hours)
**Agent**: deep + playwright

**Morning (4h)**:
- [ ] Create `web/src/app/usage/page.tsx`
  - Time range selector (24h, 7d, 30d)
  - Requests chart (line chart)
  - Tokens used chart (bar chart)
  - Cost breakdown pie chart
  - Export to CSV button

**Afternoon (4h)**:
- [ ] Create `web/src/app/costs/page.tsx`
  - Cost trends chart
  - Budget widget
  - Forecast display
  - Alert configuration
  - Provider cost comparison

**Output**: Analytics pages with Recharts

**Sprint 2 Deliverable**: 6 functional admin pages with real API integration

---

## SPRINT 3: Advanced Features (Days 8-10)
**Goal**: Real-time updates, advanced visualization, polish

### Day 8: Real-Time Dashboard (8 hours)
**Agent**: deep + dev-browser

**Morning (4h)**:
- [ ] Create `web/src/app/control-rooms/page.tsx`
  - Live metrics cards
  - WebSocket/SSE integration
  - Provider health grid
  - Circuit breaker status
  - Auto-refresh toggle

**Afternoon (4h)**:
- [ ] Create `web/src/components/organisms/LiveMetrics.tsx`
  - Rolling counters
  - Sparkline charts
  - Color-coded thresholds
  - Alert banners

**Output**: Real-time control room

---

### Day 9: Trace Explorer (8 hours)
**Agent**: deep + playwright

**Morning (4h)**:
- [ ] Create `web/src/app/traces/page.tsx` (List)
  - Filter by: Status, Provider, Model, Time
  - Trace table with expandable rows
  - Quick view drawer
  - Export to JSON

**Afternoon (4h)**:
- [ ] Create `web/src/components/organisms/TraceWaterfall.tsx`
  - Visual timeline of request phases
  - Provider routing visualization
  - Retry indicators
  - Error highlighting
  - Collapsible sections

**Output**: Full trace explorer

---

### Day 10: Polish & RBAC (8 hours)
**Agent**: deep + playwright

**Morning (4h)**:
- [ ] Create `web/src/app/admin/users/page.tsx`
  - User list with roles
  - Invite user modal
  - Role assignment
  - User detail view

**Afternoon (4h)**:
- [ ] Add loading skeletons
- [ ] Error boundaries
- [ ] Empty states
- [ ] Responsive fixes
- [ ] Dark mode support

**Output**: Polished UI with RBAC

**Sprint 3 Deliverable**: Production-quality UI with real-time features

---

## SPRINT 4: Real Provider Adapters (Days 11-16)
**Goal**: Replace MockAdapter with real HTTP clients

### Day 11: OpenAI Adapter - Foundation (8 hours)
**Agent**: deep + ultrabrain

**Morning (4h)**:
- [ ] Create `internal/provider/openai/client.go`
  - HTTP client with retry logic
  - Request builder
  - Response parser
  - Error handling

**Afternoon (4h)**:
- [ ] Create `internal/provider/openai/auth.go`
  - API key management
  - Organization headers
  - Rate limit handling

**Output**: OpenAI client foundation

---

### Day 12: OpenAI Adapter - Chat Completions (8 hours)
**Agent**: deep + ultrabrain

**Morning (4h)**:
- [ ] Implement chat completions
  - Message format conversion
  - Model mapping
  - Token counting
  - Error classification

**Afternoon (4h)**:
- [ ] Implement streaming (SSE)
  - Chunk parsing
  - Event streaming
  - Connection management
  - Abort handling

**Output**: Working chat completions

---

### Day 13: OpenAI Adapter - Embeddings & Images (8 hours)
**Agent**: deep + ultrabrain

**Morning (4h)**:
- [ ] Implement embeddings endpoint
  - Input format handling
  - Model selection
  - Batch processing

**Afternoon (4h)**:
- [ ] Implement image generation
  - Prompt handling
  - Size/format options
  - Response parsing

**Output**: OpenAI adapter complete

---

### Day 14: Anthropic Adapter (8 hours)
**Agent**: deep + ultrabrain

**Morning (4h)**:
- [ ] Create `internal/provider/anthropic/client.go`
  - Claude API integration
  - Message format conversion
  - Version header handling

**Afternoon (4h)**:
- [ ] Implement streaming
  - SSE handling
  - Anthropic-specific events
  - Tool use support

**Output**: Anthropic adapter complete

---

### Day 15: Gemini Adapter (8 hours)
**Agent**: deep + ultrabrain

**Morning (4h)**:
- [ ] Create `internal/provider/gemini/client.go`
  - Google AI API integration
  - Content format conversion
  - Safety settings

**Afternoon (4h)**:
- [ ] Implement all endpoints
  - Chat completions
  - Embeddings
  - Streaming support

**Output**: Gemini adapter complete

---

### Day 16: Adapter Integration (8 hours)
**Agent**: deep + ultrabrain

**Morning (4h)**:
- [ ] Update provider registry
  - Register real adapters
  - Health check integration
  - Circuit breaker wiring

**Afternoon (4h)**:
- [ ] Configuration
  - Provider credentials (Infisical)
  - Model routing
  - Fallback chains
  - Testing with real API keys

**Output**: Real provider adapters integrated

**Sprint 4 Deliverable**: Backend processes real AI requests

---

## SPRINT 5: Deployment & Integration (Days 17-19)
**Goal**: Production deployment of complete stack

### Day 17: Web UI Containerization (8 hours)
**Agent**: deep + git-master

**Morning (4h)**:
- [ ] Create `web/Dockerfile`
  - Multi-stage build (Node.js → Nginx)
  - npm ci for dependencies
  - npm run build
  - Nginx static serving

**Afternoon (4h)**:
- [ ] Create nginx configuration
  - Static file serving
  - API proxy to backend
  - Gzip compression
  - Security headers

**Output**: Web UI container image

---

### Day 18: Production Deployment (8 hours)
**Agent**: deep + git-master

**Morning (4h)**:
- [ ] Create deployment manifests
  - `deploy/docker-compose.yml`
  - Backend service definition
  - Web UI service definition
  - Shared network configuration

**Afternoon (4h)**:
- [ ] Create systemd services
  - `radgateway01-web.service`
  - Nginx reverse proxy service
  - Update `install.sh`
  - SSL certificate handling (certbot)

**Output**: Production deployment scripts

---

### Day 19: Monitoring & Integration (8 hours)
**Agent**: deep + git-master

**Morning (4h)**:
- [ ] Add monitoring
  - Prometheus metrics endpoint
  - Grafana dashboard
  - Health checks
  - Alert rules

**Afternoon (4h)**:
- [ ] Integrate services
  - Web UI → Backend API calls
  - CORS configuration
  - JWT token flow
  - Real-time SSE

**Output**: Fully integrated stack

**Sprint 5 Deliverable**: Production-ready deployment

---

## SPRINT 6: Testing & Polish (Days 20-21)
**Goal**: End-to-end validation and bug fixes

### Day 20: E2E Testing (8 hours)
**Agent**: deep + playwright + dev-browser

**Morning (4h)**:
- [ ] Playwright E2E tests
  - Login flow
  - Dashboard navigation
  - Provider CRUD
  - API key lifecycle
  - Usage analytics

**Afternoon (4h)**:
- [ ] API integration tests
  - Real provider calls (with test keys)
  - Error scenarios
  - Rate limiting
  - Circuit breaker

**Output**: Test suite passing

---

### Day 21: Bug Fixes & Documentation (8 hours)
**Agent**: deep + writing

**Morning (4h)**:
- [ ] Bug fixes
  - UI polish
  - Performance optimization
  - Error handling improvements
  - Mobile responsiveness

**Afternoon (4h)**:
- [ ] Documentation
  - Update README
  - Deployment guide
  - API documentation
  - Changelog
  - Commit all changes

**Output**: Production release

**Sprint 6 Deliverable**: Validated, documented, production-ready system

---

## Daily Execution Checklist (Per Agent)

### Morning Routine (Start of Day)
1. [ ] Pull latest code
2. [ ] Run tests
3. [ ] Check dependencies
4. [ ] Review yesterday's work

### End of Day
1. [ ] Run linting
2. [ ] Run type checking
3. [ ] Run tests
4. [ ] Commit changes
5. [ ] Push to remote
6. [ ] Update todo list

---

## Resource Requirements

### For Web UI Development:
- Node.js 18+
- Next.js 14
- Tailwind CSS
- Zustand
- TanStack Query
- Recharts
- Lucide React icons

### For Backend Development:
- Go 1.24
- PostgreSQL
- Redis (optional)
- Test API keys for OpenAI/Anthropic/Gemini

### For Deployment:
- Podman/Docker
- Nginx
- systemd
- SSL certificates

---

## Success Criteria

### Sprint 1 ✅
- [ ] 15+ reusable components
- [ ] Component library documented
- [ ] Layout system functional

### Sprint 2 ✅
- [ ] 6 admin pages working
- [ ] Real API integration
- [ ] Authentication flow complete

### Sprint 3 ✅
- [ ] Real-time updates
- [ ] Charts and visualizations
- [ ] RBAC implemented

### Sprint 4 ✅
- [ ] OpenAI adapter working
- [ ] Anthropic adapter working
- [ ] Gemini adapter working
- [ ] Real AI requests processing

### Sprint 5 ✅
- [ ] Web UI containerized
- [ ] Production deployment scripts
- [ ] Monitoring in place

### Sprint 6 ✅
- [ ] E2E tests passing
- [ ] Documentation complete
- [ ] Production ready

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| **Web UI takes longer** | Parallel development of backend adapters |
| **Provider API changes** | Use official SDKs, add version pinning |
| **Test API keys expire** | Use environment variables, rotate regularly |
| **SSE connection issues** | Fallback to polling, add reconnection logic |
| **Mobile responsiveness** | Use Tailwind responsive classes, test early |

---

## Daily Standup Questions (Self-Check)

1. What did I complete yesterday?
2. What will I work on today?
3. Are there any blockers?

---

## Final Deliverable

By Day 21, you will have:

1. ✅ **Complete Web UI** deployed at `http://172.16.30.45`
2. ✅ **Real AI Provider Integration** (OpenAI, Anthropic, Gemini)
3. ✅ **Production Backend** with monitoring
4. ✅ **End-to-end tested** system
5. ✅ **Full documentation**

**Total Lines of Code**: ~15,000 (Web UI) + ~8,000 (Adapters)  
**Total Commits**: ~150  
**Test Coverage**: 80%+

---

**Plan Ready for Execution**  
**Start Date**: Upon approval  
**Estimated Completion**: 21 days from start
