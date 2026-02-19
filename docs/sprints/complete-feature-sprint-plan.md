# RAD Gateway - Complete Feature Sprint Plan
**Goal**: Achieve feature parity with Plexus/AxonHub plus A2A differentiation
**Duration**: 8 Weeks (40 days)
**Teams**: 4 parallel teams (Frontend, Backend A2A, OAuth/Audio, DevOps/QA)

---

## 📋 Executive Summary

### Current State Gaps
1. **Web UI**: Read-only dashboard, no CRUD forms (Critical)
2. **A2A Protocol**: Model cards only, missing task lifecycle (High)
3. **OAuth**: No OAuth provider support (Medium)
4. **Audio APIs**: No STT/TTS support (Medium)
5. **MCP**: No Model Context Protocol proxy (Medium)
6. **Reporting**: Basic analytics, missing advanced filters (Low)

### Success Criteria
- [ ] Web UI fully functional with all CRUD operations
- [ ] A2A protocol 100% spec compliant
- [ ] OAuth support for 5 major providers
- [ ] Audio APIs (STT/TTS) operational
- [ ] MCP proxy server functional
- [ ] Advanced reporting dashboard

---

## 🏃 Phase 1: Web UI CRUD Forms (Weeks 1-2)
**Team**: Frontend (2 engineers) + UI/UX Designer
**Priority**: CRITICAL - UI currently non-functional for management

### Day 1-2: Provider Management Forms

#### Day 1 Morning: Provider Create Form
**Task**: Build `/app/providers/new/page.tsx`
- [ ] Create provider form with fields:
  - Name (text, required)
  - Slug (text, auto-generate from name)
  - Provider Type (select: openai, anthropic, gemini)
  - Base URL (text with validation)
  - API Key (password input with reveal toggle)
  - Config JSON editor (codemirror or textarea)
  - Priority (number input, default 0)
  - Weight (number input, default 1)
- [ ] Form validation using React Hook Form + Zod
- [ ] Connect to `useCreateProvider` mutation
- [ ] Test connection button (call backend health check)
- [ ] Success toast + redirect to providers list

**Files to create**:
```
web/src/app/providers/new/page.tsx
web/src/components/forms/ProviderForm.tsx
web/src/hooks/useCreateProvider.ts (if not exists)
```

**API Integration**:
```typescript
// POST /v0/admin/providers
interface CreateProviderRequest {
  name: string;
  slug: string;
  providerType: 'openai' | 'anthropic' | 'gemini';
  baseUrl: string;
  apiKey: string;
  config?: Record<string, unknown>;
  priority?: number;
  weight?: number;
}
```

**Acceptance Criteria**:
- [ ] Form validates all required fields
- [ ] Slug auto-generates from name (kebab-case)
- [ ] API key is masked in input
- [ ] Test connection button works
- [ ] Shows loading state during submission
- [ ] Redirects to providers list on success
- [ ] Shows error toast on API failure

#### Day 1 Afternoon: Provider Edit Form
**Task**: Build `/app/providers/[id]/edit/page.tsx`
- [ ] Reuse ProviderForm component
- [ ] Load existing provider data with `useProvider(id)`
- [ ] Pre-populate all fields
- [ ] Update button calls `useUpdateProvider` mutation
- [ ] Add "Delete Provider" button with confirmation modal
- [ ] Add "Rotate API Key" button

**Files to create**:
```
web/src/app/providers/[id]/edit/page.tsx
web/src/components/modals/DeleteProviderModal.tsx
web/src/hooks/useUpdateProvider.ts
web/src/hooks/useDeleteProvider.ts
web/src/hooks/useRotateProviderKey.ts
```

**Acceptance Criteria**:
- [ ] Loads provider data correctly
- [ ] Shows loading skeleton while fetching
- [ ] Updates work without full page reload
- [ ] Delete shows confirmation modal
- [ ] Rotate key shows new key in modal with copy button

#### Day 2: Provider Detail View + Circuit Control
**Task**: Build `/app/providers/[id]/page.tsx`
- [ ] Provider detail card with all fields
- [ ] Health status badge with last check time
- [ ] Circuit breaker state visualization
- [ ] Control buttons:
  - Open Circuit
  - Close Circuit
  - Reset Circuit
  - Trigger Health Check
- [ ] Metrics mini-chart (last 24h requests/errors)
- [ ] Recent errors table (last 10)

**Files to create**:
```
web/src/app/providers/[id]/page.tsx
web/src/components/cards/ProviderDetailCard.tsx
web/src/components/cards/CircuitBreakerCard.tsx
web/src/components/charts/ProviderMetricsChart.tsx
web/src/hooks/useProviderMetrics.ts
web/src/hooks/useControlCircuit.ts
web/src/hooks/useTriggerHealthCheck.ts
```

**Acceptance Criteria**:
- [ ] Shows all provider fields
- [ ] Circuit state clearly visible
- [ ] Control buttons call correct APIs
- [ ] Health check triggers immediate refresh
- [ ] Metrics chart loads and displays data

---

### Day 3-4: API Key Management Forms

#### Day 3: API Key Create + Revoke
**Task**: Build `/app/api-keys/new/page.tsx`
- [ ] Create API key form:
  - Name (text, required)
  - Workspace (select from existing)
  - Expires At (datetime-local, optional)
  - Rate Limit (number, optional)
  - Allowed Models (multi-select, optional)
  - Allowed APIs (multi-select: chat, embeddings, images)
  - Metadata JSON editor
- [ ] On submit, shows modal with full key (copy to clipboard)
- [ ] Key is only shown once (security)

**Files to create**:
```
web/src/app/api-keys/new/page.tsx
web/src/components/forms/APIKeyForm.tsx
web/src/components/modals/ShowKeyModal.tsx
web/src/hooks/useCreateAPIKey.ts (verify exists)
```

**Acceptance Criteria**:
- [ ] Form validates required fields
- [ ] Shows full key in modal on creation
- [ ] Copy to clipboard works
- [ ] Warning that key won't be shown again
- [ ] Redirects to API keys list

#### Day 4: API Key Edit + Bulk Operations
**Task**: Build `/app/api-keys/[id]/edit/page.tsx`
- [ ] Edit form (reusable from create)
- [ ] Status toggle (active/revoked)
- [ ] Revoke button with confirmation
- [ ] Rotate key button
- [ ] Bulk operations on list page:
  - Select multiple keys with checkboxes
  - Bulk revoke action
  - Bulk delete action
  - Bulk export to CSV

**Files to create**:
```
web/src/app/api-keys/[id]/edit/page.tsx
web/src/components/tables/BulkOperationsToolbar.tsx
web/src/hooks/useBulkRevokeAPIKeys.ts
web/src/hooks/useBulkDeleteAPIKeys.ts
web/src/hooks/useExportAPIKeys.ts
```

**Acceptance Criteria**:
- [ ] Can revoke individual keys
- [ ] Can rotate keys
- [ ] Bulk operations work with selection
- [ ] CSV export includes all fields

---

### Day 5-6: Project/Workspace Forms

#### Day 5: Project Create + Edit
**Task**: Build `/app/projects/new/page.tsx` and `/app/projects/[id]/edit/page.tsx`
- [ ] Project form:
  - Name (text, required)
  - Slug (auto-generate)
  - Description (textarea)
  - Logo upload (optional, image preview)
  - Settings:
    - Theme (light/dark/system)
    - Timezone (select)
    - Currency (USD/EUR/GBP)
    - Date Format (select)
- [ ] Form validation

**Files to create**:
```
web/src/app/projects/new/page.tsx
web/src/app/projects/[id]/edit/page.tsx
web/src/components/forms/ProjectForm.tsx
web/src/components/inputs/ImageUpload.tsx
web/src/hooks/useCreateProject.ts (verify)
web/src/hooks/useUpdateProject.ts
```

**Acceptance Criteria**:
- [ ] Logo upload with preview
- [ ] Settings persist correctly
- [ ] Slug auto-generation works

#### Day 6: Project Detail + Member Management
**Task**: Build `/app/projects/[id]/page.tsx`
- [ ] Project detail view
- [ ] Member list with roles
- [ ] Add member button (email input + role select)
- [ ] Remove member button
- [ ] Project statistics cards:
  - Total API keys
  - Total requests (24h)
  - Total cost (month)
  - Active providers

**Files to create**:
```
web/src/app/projects/[id]/page.tsx
web/src/components/cards/ProjectStatsCard.tsx
web/src/components/tables/MemberTable.tsx
web/src/hooks/useProjectMembers.ts
web/src/hooks/useAddProjectMember.ts
web/src/hooks/useRemoveProjectMember.ts
```

**Acceptance Criteria**:
- [ ] Shows all project info
- [ ] Can add/remove members
- [ ] Statistics load correctly

---

### Day 7-8: Form Components + Validation

#### Day 7: Shared Form Components
**Task**: Build reusable form components
- [ ] FormField wrapper (label + input + error)
- [ ] SelectField with search
- [ ] MultiSelectField (for models, APIs)
- [ ] DateTimeField with picker
- [ ] JSONEditor (for config/metadata)
- [ ] ImageUpload with drag-drop
- [ ] FormActions (submit/cancel buttons)

**Files to create**:
```
web/src/components/forms/FormField.tsx
web/src/components/forms/SelectField.tsx
web/src/components/forms/MultiSelectField.tsx
web/src/components/forms/DateTimeField.tsx
web/src/components/forms/JSONEditor.tsx
web/src/components/forms/FormActions.tsx
```

#### Day 8: Form Validation + Error Handling
**Task**: Implement Zod schemas and error handling
- [ ] Validation schemas for all forms
- [ ] API error to form field mapping
- [ ] Global error boundary for forms
- [ ] Success/error toast notifications
- [ ] Form dirty state tracking (unsaved changes warning)

**Files to create**:
```
web/src/lib/validation/providerSchema.ts
web/src/lib/validation/apiKeySchema.ts
web/src/lib/validation/projectSchema.ts
web/src/components/error/FormErrorBoundary.tsx
web/src/hooks/useFormDirtyState.ts
```

---

### Day 9-10: Testing + Polish

#### Day 9: Testing
**Task**: Write tests for all forms
- [ ] Unit tests for form components
- [ ] Integration tests for form submissions
- [ ] E2E tests for full CRUD flows
- [ ] Accessibility audit (keyboard navigation)

**Files to create**:
```
web/src/components/forms/__tests__/ProviderForm.test.tsx
web/src/components/forms/__tests__/APIKeyForm.test.tsx
web/src/components/forms/__tests__/ProjectForm.test.tsx
web/tests/e2e/provider-crud.spec.ts
web/tests/e2e/apikey-crud.spec.ts
web/tests/e2e/project-crud.spec.ts
```

#### Day 10: Polish + Documentation
**Task**: Final polish
- [ ] Mobile responsiveness check
- [ ] Loading states consistency
- [ ] Empty states for all tables
- [ ] Documentation update
- [ ] Deploy to staging

**Deliverables**:
- [ ] All CRUD operations functional
- [ ] Tests passing
- [ ] Staging deployment verified

---

## 🤖 Phase 2: A2A Protocol Completion (Weeks 3-4)
**Team**: Backend A2A (2 engineers)
**Priority**: HIGH - Strategic differentiator

### Week 3: Core A2A Task Endpoints

#### Day 11-12: Task State Machine + Database
**Task**: Implement task persistence

**Files to create**:
```
internal/a2a/task.go                 # Task entity
internal/a2a/task_store.go           # Interface
internal/a2a/task_store_pg.go      # PostgreSQL impl
internal/db/migrations/009_create_a2a_tasks.sql
```

**Code requirements**:
```go
// Task state machine
type TaskState string
const (
    TaskStateSubmitted     TaskState = "submitted"
    TaskStateWorking       TaskState = "working"
    TaskStateInputRequired TaskState = "input-required"
    TaskStateCompleted     TaskState = "completed"
    TaskStateCanceled      TaskState = "canceled"
    TaskStateFailed        TaskState = "failed"
)

// Task transitions
func (t *Task) CanTransitionTo(state TaskState) bool {
    switch t.Status {
    case TaskStateSubmitted:
        return state == TaskStateWorking || state == TaskStateCanceled
    case TaskStateWorking:
        return state == TaskStateCompleted || state == TaskStateFailed || 
               state == TaskStateInputRequired || state == TaskStateCanceled
    // ... etc
    }
}
```

**Database Schema**:
```sql
CREATE TABLE a2a_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status TEXT NOT NULL,
    session_id TEXT NOT NULL,
    message JSONB NOT NULL,
    artifacts JSONB DEFAULT '[]',
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    parent_id UUID REFERENCES a2a_tasks(id),
    workspace_id TEXT REFERENCES workspaces(id),
    assigned_agent_id TEXT
);
```

#### Day 13: /.well-known/agent.json
**Task**: Implement Agent Card discovery

**Files to create**:
```
internal/a2a/agent_card.go           # Agent Card generation
internal/a2a/agent_card_handler.go   # HTTP handler
```

**Code requirements**:
```go
// GET /.well-known/agent.json
func (h *Handler) handleAgentCard(w http.ResponseWriter, r *http.Request) {
    card := AgentCard{
        Name: "RAD Gateway",
        Description: "AI API Gateway with A2A support",
        URL: h.config.BaseURL + "/a2a",
        Version: version.Version,
        Capabilities: Capabilities{
            Streaming: true,
            PushNotifications: false,
            StateTransitionHistory: true,
        },
        Skills: h.getSkills(), // From model cards
        Authentication: AuthenticationInfo{
            Schemes: []string{"Bearer", "OAuth2"},
        },
    }
    writeJSON(w, http.StatusOK, card)
}
```

#### Day 14-15: POST /a2a/tasks/send
**Task**: Sync task execution

**Files to create**:
```
internal/a2a/task_handlers.go        # Task handlers
internal/a2a/delegation.go           # Task delegation logic
internal/a2a/validation.go           # Request validation
```

**Code requirements**:
```go
// POST /a2a/tasks/send
func (h *Handler) handleSendTask(w http.ResponseWriter, r *http.Request) {
    var req SendTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request")
        return
    }
    
    // Validate
    if err := h.validate.SendTaskRequest(&req); err != nil {
        writeError(w, http.StatusBadRequest, err.Error())
        return
    }
    
    // Create task
    task := &Task{
        ID: uuid.New().String(),
        Status: TaskStateSubmitted,
        Message: req.Message,
        SessionID: req.SessionID,
        Metadata: req.Metadata,
    }
    
    // Persist
    if err := h.store.CreateTask(r.Context(), task); err != nil {
        writeError(w, http.StatusInternalServerError, "failed to create task")
        return
    }
    
    // Delegate to provider (blocking for sync)
    result, err := h.delegator.ExecuteSync(r.Context(), task)
    if err != nil {
        task.Status = TaskStateFailed
        task.Metadata = json.RawMessage(`{"error": "` + err.Error() + `"}`)
        h.store.UpdateTask(r.Context(), task)
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    // Update task
    task.Status = TaskStateCompleted
    task.Artifacts = result.Artifacts
    h.store.UpdateTask(r.Context(), task)
    
    writeJSON(w, http.StatusOK, SendTaskResponse{Task: task})
}
```

#### Day 16: GET /a2a/tasks/{taskId}
**Task**: Task retrieval endpoint

**Code requirements**:
```go
// GET /a2a/tasks/{taskId}
func (h *Handler) handleGetTask(w http.ResponseWriter, r *http.Request) {
    taskID := extractTaskID(r.URL.Path)
    
    task, err := h.store.GetTask(r.Context(), taskID)
    if err != nil {
        if errors.Is(err, ErrTaskNotFound) {
            writeError(w, http.StatusNotFound, "task not found")
            return
        }
        writeError(w, http.StatusInternalServerError, "failed to get task")
        return
    }
    
    writeJSON(w, http.StatusOK, GetTaskResponse{Task: task})
}
```

#### Day 17: POST /a2a/tasks/{taskId}/cancel
**Task**: Task cancellation

**Code requirements**:
```go
// POST /a2a/tasks/{taskId}/cancel
func (h *Handler) handleCancelTask(w http.ResponseWriter, r *http.Request) {
    taskID := extractTaskID(r.URL.Path)
    
    task, err := h.store.GetTask(r.Context(), taskID)
    if err != nil {
        writeError(w, http.StatusNotFound, "task not found")
        return
    }
    
    // Can only cancel non-terminal tasks
    if task.Status == TaskStateCompleted || task.Status == TaskStateFailed {
        writeError(w, http.StatusConflict, "task already terminal")
        return
    }
    
    // Cancel delegation
    h.delegator.Cancel(taskID)
    
    task.Status = TaskStateCanceled
    h.store.UpdateTask(r.Context(), task)
    
    writeJSON(w, http.StatusOK, CancelTaskResponse{Task: task})
}
```

---

### Week 4: A2A Streaming + Integration

#### Day 18-19: POST /a2a/tasks/sendSubscribe (SSE)
**Task**: Streaming task execution

**Files to create**:
```
internal/a2a/streaming.go            # SSE streaming support
internal/a2a/task_events.go          # Task event types
```

**Code requirements**:
```go
// POST /a2a/tasks/sendSubscribe
func (h *Handler) handleSendSubscribe(w http.ResponseWriter, r *http.Request) {
    var req SendTaskRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request")
        return
    }
    
    // Create task
    task := &Task{
        ID: uuid.New().String(),
        Status: TaskStateSubmitted,
        Message: req.Message,
        SessionID: req.SessionID,
    }
    h.store.CreateTask(r.Context(), task)
    
    // Set up SSE
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        writeError(w, http.StatusInternalServerError, "streaming not supported")
        return
    }
    
    // Event channel
    events := make(chan TaskEvent, 100)
    h.delegator.ExecuteAsync(r.Context(), task, events)
    
    // Stream events
    for event := range events {
        data, _ := json.Marshal(event)
        fmt.Fprintf(w, "data: %s\n\n", data)
        flusher.Flush()
        
        if event.Type == EventTypeCompleted || event.Type == EventTypeFailed {
            break
        }
    }
}
```

#### Day 20: A2A Web UI Console
**Task**: Build A2A management UI

**Files to create**:
```
web/src/app/a2a/page.tsx
web/src/app/a2a/tasks/page.tsx
web/src/components/a2a/TaskList.tsx
web/src/components/a2a/TaskDetail.tsx
web/src/components/a2a/AgentCardViewer.tsx
web/src/hooks/useA2ATasks.ts
web/src/hooks/useSendA2ATask.ts
web/src/hooks/useA2AStreaming.ts
```

**Features**:
- [ ] Task list with filters (status, date, agent)
- [ ] Task detail view with artifacts
- [ ] Send task form (sync + async options)
- [ ] Real-time streaming display
- [ ] Agent Card viewer

#### Day 21-22: Testing + Compliance
**Task**: A2A conformance tests

**Files to create**:
```
tests/integration/a2a_tasks_test.go
tests/integration/a2a_streaming_test.go
```

**Test cases**:
- [ ] Sync task execution
- [ ] Async task with SSE streaming
- [ ] Task cancellation
- [ ] State transitions
- [ ] Agent Card discovery
- [ ] Error handling

**Deliverables**:
- [ ] All A2A endpoints operational
- [ ] 100% spec compliance
- [ ] Web UI console functional
- [ ] Tests passing

---

## 🔐 Phase 3: OAuth Provider Support (Week 5)
**Team**: OAuth/Integration (1 engineer)
**Priority**: MEDIUM - Competitive parity

### Day 23-24: OAuth Framework
**Task**: OAuth provider abstraction

**Files to create**:
```
internal/oauth/provider.go           # Provider interface
internal/oauth/github.go             # GitHub Copilot
internal/oauth/anthropic.go          # Anthropic
internal/oauth/gemini.go             # Gemini CLI
internal/oauth/openai.go             # OpenAI Codex
internal/oauth/manager.go            # OAuth session manager
```

**Code requirements**:
```go
// OAuthProvider interface
type OAuthProvider interface {
    Name() string
    GetAuthURL(state string) string
    ExchangeCode(ctx context.Context, code string) (*Token, error)
    RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
    ValidateToken(ctx context.Context, token string) error
}

// Token response
type Token struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    TokenType    string
}
```

### Day 25-26: OAuth Endpoints
**Task**: OAuth HTTP handlers

**Files to create**:
```
internal/api/oauth.go                # OAuth handlers
```

**Endpoints**:
```
POST /v1/oauth/start                   # Start OAuth flow
GET  /v1/oauth/callback/{provider}    # OAuth callback
POST /v1/oauth/refresh               # Refresh token
POST /v1/oauth/validate              # Validate token
```

### Day 27-28: Provider Integration
**Task**: Integrate OAuth with providers

**Code changes**:
```go
// Provider config supports OAuth
 type ProviderConfig struct {
     Name         string
     Type         string
     AuthType     string // "api_key" or "oauth"
     OAuthProvider string // "github-copilot", "anthropic", etc.
     OAuthAccount string // User's account identifier
 }
 
 // Provider adapter uses OAuth token
 func (p *Provider) GetToken(ctx context.Context) (string, error) {
     if p.config.AuthType == "oauth" {
         return p.oauthManager.GetAccessToken(ctx, p.config.OAuthProvider, p.config.OAuthAccount)
     }
     return p.config.APIKey, nil
 }
```

### Day 29-30: OAuth UI + Testing
**Task**: OAuth management UI

**Files to create**:
```
web/src/app/settings/oauth/page.tsx
web/src/components/oauth/OAuthProviderList.tsx
web/src/components/oauth/OAuthConnectButton.tsx
web/src/hooks/useOAuthConnect.ts
```

**Features**:
- [ ] Connect OAuth provider buttons
- [ ] OAuth status dashboard
- [ ] Reconnect/Disconnect actions
- [ ] Token refresh indicator

---

## 🎵 Phase 4: Audio APIs & MCP (Weeks 6-7)
**Team**: Audio/MCP (1 engineer)
**Priority**: MEDIUM - Feature parity

### Week 6: Audio APIs (STT/TTS)

#### Day 31-32: Speech-to-Text (STT)
**Task**: Implement audio transcription

**Files to create**:
```
internal/provider/openai/stt.go      # OpenAI Whisper
internal/provider/gemini/stt.go      # Gemini STT
internal/api/stt.go                  # HTTP handler
```

**Endpoints**:
```
POST /v1/audio/transcriptions
POST /v1/audio/translations
```

**Code requirements**:
```go
func (h *Handler) handleTranscriptions(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form
    file, header, err := r.FormFile("file")
    if err != nil {
        writeError(w, http.StatusBadRequest, "no file provided")
        return
    }
    defer file.Close()
    
    model := r.FormValue("model") // whisper-1
    language := r.FormValue("language")
    prompt := r.FormValue("prompt")
    
    // Route to provider
    result, err := h.router.Transcribe(r.Context(), model, file, header.Size, language, prompt)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    writeJSON(w, http.StatusOK, result)
}
```

#### Day 33-34: Text-to-Speech (TTS)
**Task**: Implement speech generation

**Files to create**:
```
internal/provider/openai/tts.go      # OpenAI TTS
internal/api/tts.go                  # HTTP handler
```

**Endpoints**:
```
POST /v1/audio/speech
```

#### Day 35: Audio UI
**Task**: Audio management UI

**Files to create**:
```
web/src/app/audio/page.tsx
web/src/components/audio/TranscriptionForm.tsx
web/src/components/audio/SpeechForm.tsx
web/src/components/audio/AudioPlayer.tsx
```

### Week 7: MCP Proxy

#### Day 36-37: MCP Server
**Task**: Implement MCP proxy

**Files to create**:
```
internal/mcp/server.go               # MCP server
internal/mcp/proxy.go                # Proxy to stdio MCP
internal/mcp/handlers.go             # MCP HTTP handlers
```

**Code requirements**:
```go
// MCP over HTTP (streamable)
// POST /mcp/v1/stdio
func (h *Handler) handleMCP(w http.ResponseWriter, r *http.Request) {
    var req MCPRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request")
        return
    }
    
    // Proxy to stdio MCP server
    result, err := h.proxy.Execute(r.Context(), req)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err.Error())
        return
    }
    
    writeJSON(w, http.StatusOK, result)
}
```

#### Day 38-39: MCP UI + Testing
**Task**: MCP management UI + tests

**Files to create**:
```
web/src/app/mcp/page.tsx
web/src/components/mcp/MCPConnectionList.tsx
web/src/components/mcp/MCPConsole.tsx
```

#### Day 40: Audio/MCP Testing
**Task**: Final testing

**Test files**:
```
tests/integration/audio_test.go
tests/integration/mcp_test.go
```

---

## 📊 Phase 5: Advanced Reporting (Week 8)
**Team**: Reporting/Analytics (1 engineer)
**Priority**: LOW - Nice to have

### Day 41-42: Advanced Filters
**Task**: Enhanced usage filtering

**Files to modify**:
```
internal/admin/usage.go              # Add filters
web/src/app/usage/page.tsx           # Add filter UI
```

**New filters**:
- Date range picker
- API key multi-select
- Provider multi-select
- Model multi-select
- Status checkboxes
- Cost range slider

### Day 43-44: Performance Metrics
**Task**: TTFT and TPS metrics

**Files to create**:
```
internal/metrics/performance.go      # Performance tracking
web/src/components/metrics/PerformanceDashboard.tsx
```

**Metrics**:
- TTFT (Time to First Token) per provider
- TPS (Tokens Per Second) per model
- Latency percentiles (p50, p95, p99)
- Error rate trends

### Day 45-46: Attribution Tracking
**Task**: Usage attribution

**Code changes**:
```go
// Usage record supports attribution
 type UsageRecord struct {
     // ... existing fields
     AttributionKey string // "sk-key:attribution"
     Source         string // "api", "a2a", "web"
     TeamID         string
 }
 
 // Filter by attribution
 func (r *PostgresRepository) GetUsageByAttribution(ctx context.Context, key string) ([]UsageRecord, error)
```

### Day 47-48: Reporting UI
**Task**: Advanced reporting dashboard

**Files to create**:
```
web/src/app/reports/page.tsx
web/src/components/reports/UsageReportBuilder.tsx
web/src/components/reports/CostReportBuilder.tsx
web/src/components/reports/PerformanceReport.tsx
```

**Features**:
- [ ] Report builder with drag-drop
- [ ] Saved reports
- [ ] Scheduled reports (email)
- [ ] Export to CSV/PDF
- [ ] Charts and visualizations

### Day 49-50: Testing + Documentation
**Task**: Final testing and docs

**Deliverables**:
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Production deployment

---

## 📈 Dependencies & Critical Path

### Critical Path
```
Week 1-2 (Web UI) ──┐
                    ├──► Week 5 (OAuth) ──► Week 6-7 (Audio/MCP)
Week 3-4 (A2A) ─────┘
```

### Dependencies
1. **Web UI** depends on: Backend API (exists)
2. **A2A Tasks** depends on: Provider adapters (exists)
3. **OAuth** depends on: Authentication system (exists)
4. **Audio/MCP** depends on: Provider adapters (exists)
5. **Reporting** depends on: Usage tracking (exists)

### Parallel Work Streams
- Frontend team: Web UI (Weeks 1-2) → OAuth UI (Week 5) → Audio/MCP UI (Weeks 6-7)
- Backend A2A team: A2A implementation (Weeks 3-4)
- OAuth/Audio team: OAuth (Week 5) → Audio (Week 6) → MCP (Week 7)
- DevOps/QA: Testing throughout

---

## 🎯 Acceptance Criteria by Phase

### Phase 1: Web UI
- [ ] All CRUD forms functional
- [ ] Tests passing (>80% coverage)
- [ ] Mobile responsive
- [ ] Accessibility audit passed
- [ ] Deployed to staging

### Phase 2: A2A Protocol
- [ ] All A2A endpoints operational
- [ ] 100% spec compliance
- [ ] Conformance tests passing
- [ ] Web UI console functional
- [ ] Documentation complete

### Phase 3: OAuth
- [ ] 5 OAuth providers working
- [ ] Token refresh automatic
- [ ] UI for connect/disconnect
- [ ] Tests passing

### Phase 4: Audio/MCP
- [ ] STT/TTS working
- [ ] MCP proxy functional
- [ ] Tests passing

### Phase 5: Reporting
- [ ] Advanced filters working
- [ ] Performance metrics visible
- [ ] Report builder functional
- [ ] Tests passing

---

## ⚠️ Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Web UI complexity underestimated | Medium | High | Component library, reusable forms |
| A2A spec changes | Medium | Medium | Version negotiation, extensible design |
| OAuth provider API changes | Low | Medium | Provider abstraction layer |
| Audio file handling issues | Medium | Medium | Streaming uploads, size limits |
| MCP stdio limitations | Medium | Low | Clear documentation of limitations |

---

## 📅 Sprint Schedule Summary

| Week | Focus | Team | Deliverables |
|------|-------|------|--------------|
| 1 | Web UI Forms (Provider, API Key) | Frontend | CRUD forms for providers + API keys |
| 2 | Web UI Forms (Project) + Polish | Frontend | All forms functional, tests passing |
| 3 | A2A Core (Tasks, State Machine) | Backend A2A | Task persistence, sync endpoint |
| 4 | A2A Streaming + UI Console | Backend A2A | Full A2A protocol, console UI |
| 5 | OAuth Support | OAuth/Integration | 5 OAuth providers, UI |
| 6 | Audio APIs (STT/TTS) | Audio/MCP | Audio endpoints, UI |
| 7 | MCP Proxy | Audio/MCP | MCP server, console |
| 8 | Advanced Reporting | Reporting | Filters, dashboards, exports |

---

**Total Estimated Effort**: 8 weeks (4 engineers)
**Daily Standup**: 9:00 AM
**Sprint Reviews**: Fridays at 3:00 PM
**Retro**: Every 2 weeks
