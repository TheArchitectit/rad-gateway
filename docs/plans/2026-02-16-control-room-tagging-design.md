# Control Room Tagging System Design

**Date**: 2026-02-16
**Status**: Approved for Implementation
**Team**: Golf (Documentation & Design) + Alpha (Architecture)

---

## Executive Summary

The Control Room Tagging System enables customizable operational views that group providers, models, A2A agents, and usage tracking into themed dashboards. This supports multi-tenancy, environment separation, team-based views, and cost tracking while maintaining the steampunk operator aesthetic.

---

## Core Concepts

### 1. Control Room

A named operational view with its own "Steam Pressure Gauges" for monitoring.

**Properties**:
- `id`: Unique identifier (e.g., "prod-platform", "ml-research")
- `name`: Display name (e.g., "Production Platform", "ML Research")
- `description`: Human-readable description
- `tagFilter`: Query for included resources
- `dashboardLayout`: Preferred gauge arrangements
- `createdBy`, `createdAt`: Audit fields

**Example**:
```yaml
controlRoom:
  id: prod-platform
  name: "Production Platform"
  description: "Production environment for platform team"
  tagFilter: "env:production AND team:platform"
  dashboardLayout:
    gauges: [pressure, latency, tokens, cost]
    theme: brass-dark
```

### 2. Hierarchical Tags

Format: `category:value`

**Standard Categories**:

| Category | Purpose | Examples |
|----------|---------|----------|
| `env` | Environment separation | `production`, `staging`, `dev` |
| `team` | Team ownership | `platform`, `ml`, `data`, `research` |
| `project` | Project/customer grouping | `customer-a`, `internal`, `experiment-7` |
| `cost-center` | Billing/cost tracking | `engineering`, `research`, `marketing` |
| `region` | Geographic placement | `us-east`, `eu-west`, `ap-south` |
| `provider` | Provider-specific | `openai`, `anthropic`, `gemini` |
| `tier` | Service tier | `premium`, `standard`, `basic` |

**Custom Categories**: Organizations can define custom categories (e.g., `compliance:hipaa`, `priority:critical`)

### 3. Resource Tagging

Resources can have multiple tags:

```yaml
provider:
  id: openai-production
  name: "OpenAI Production"
  baseUrl: https://api.openai.com/v1
  tags:
    - env:production
    - team:platform
    - cost-center:engineering
    - project:customer-a
    - provider:openai
    - tier:premium
```

**Taggable Resources**:
- Providers (OpenAI, Anthropic, Gemini)
- Models (gpt-4, claude-3, etc.)
- Routes
- A2A Agents
- API Keys
- Quotas

---

## Phase 1: MVP (Tag-Based Filtering)

### Features

1. **Control Room Creation**
   - Create control rooms with tag filters
   - Support AND, OR, NOT operators
   - Wildcard support (`project:*`)

2. **Resource Tagging**
   - Assign tags during resource creation
   - Modify tags on existing resources
   - Bulk tag operations

3. **Dashboard Filtering**
   - View shows only matching resources
   - Gauge aggregation per control room
   - Alert scoping to control room

4. **Usage & Cost Aggregation**
   - Per-control-room usage tracking
   - Cost allocation by tags
   - Export billing reports by control room

### Tag Query Language

```
# Simple equality
env:production

# AND (both must match)
env:production AND team:platform

# OR (either matches)
team:platform OR team:ml

# NOT (exclusion)
env:production AND NOT team:research

# Wildcards
project:customer-*

# Complex
(env:production OR env:staging) AND team:platform AND NOT cost-center:research
```

### API Endpoints

```
POST   /v0/management/control-rooms
GET    /v0/management/control-rooms
GET    /v0/management/control-rooms/{id}
PUT    /v0/management/control-rooms/{id}
DELETE /v0/management/control-rooms/{id}

POST   /v0/management/control-rooms/{id}/tags     # Add tags
DELETE /v0/management/control-rooms/{id}/tags     # Remove tags

GET    /v0/management/usage?controlRoom={id}      # Usage for control room
GET    /v0/management/costs?controlRoom={id}      # Cost for control room

GET    /v0/management/resources?tag=env:production # Find resources by tag
```

### Data Model

```go
// Tag represents a hierarchical tag
type Tag struct {
    Category string `json:"category"` // env, team, project, etc.
    Value    string `json:"value"`    // production, platform, etc.
}

func (t Tag) String() string {
    return fmt.Sprintf("%s:%s", t.Category, t.Value)
}

// Taggable interface for resources that can be tagged
type Taggable interface {
    GetTags() []Tag
    SetTags([]Tag)
    HasTag(Tag) bool
    AddTag(Tag)
    RemoveTag(Tag)
}

// ControlRoom represents an operational view
type ControlRoom struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`
    Description   string    `json:"description"`
    TagFilter     string    `json:"tagFilter"`     // Query string
    DashboardLayout Layout  `json:"dashboardLayout"`
    CreatedBy     string    `json:"createdBy"`
    CreatedAt     time.Time `json:"createdAt"`
    UpdatedAt     time.Time `json:"updatedAt"`
}

// ControlRoomService manages control rooms
type ControlRoomService interface {
    Create(ctx context.Context, cr *ControlRoom) error
    Get(ctx context.Context, id string) (*ControlRoom, error)
    List(ctx context.Context, filter ListFilter) ([]*ControlRoom, error)
    Update(ctx context.Context, cr *ControlRoom) error
    Delete(ctx context.Context, id string) error

    // Resource matching
    MatchResources(ctx context.Context, cr *ControlRoom) ([]Resource, error)

    // Usage aggregation
    GetUsage(ctx context.Context, id string, range TimeRange) (*UsageReport, error)
    GetCosts(ctx context.Context, id string, range TimeRange) (*CostReport, error)
}
```

### Usage Tracking with Tags

```go
type TaggedUsageRecord struct {
    Timestamp   time.Time `json:"timestamp"`
    ControlRoom string    `json:"controlRoom,omitempty"`
    Tags        []Tag     `json:"tags"`

    // Usage metrics
    RequestID      string `json:"requestId"`
    Provider       string `json:"provider"`
    Model          string `json:"model"`
    PromptTokens   int64  `json:"promptTokens"`
    CompletionTokens int64 `json:"completionTokens"`
    TotalTokens    int64  `json:"totalTokens"`
    CostUSD        float64 `json:"costUsd"`
}
```

---

## Phase 2: RBAC (Enterprise Feature)

> **Note**: Phase 2 is planned as an enterprise feature requiring assisted setup or token funding.

### Features

1. **User Assignment**
   - Users assigned to one or more control rooms
   - Default control room per user

2. **Permission Levels**
   - `view`: Read-only access to dashboard
   - `operator`: Can trigger actions (pause routes, etc.)
   - `admin`: Can modify control room settings
   - `billing`: Access to cost reports and invoices

3. **Access Control**
   - API keys scoped to control rooms
   - Row-level security in database
   - Audit logging for all actions

### RBAC Model

```go
type ControlRoomAccess struct {
    ControlRoomID string    `json:"controlRoomId"`
    UserID        string    `json:"userId"`
    Role          Role      `json:"role"` // view, operator, admin, billing
    GrantedBy     string    `json:"grantedBy"`
    GrantedAt     time.Time `json:"grantedAt"`
    ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
}

type Role string

const (
    RoleView     Role = "view"
    RoleOperator Role = "operator"
    RoleAdmin    Role = "admin"
    RoleBilling  Role = "billing"
)

func (r Role) Permissions() []Permission {
    switch r {
    case RoleView:
        return []Permission{PermViewDashboard, PermViewUsage}
    case RoleOperator:
        return append(RoleView.Permissions(), PermPauseRoute, PermTriggerFailover)
    case RoleAdmin:
        return append(RoleOperator.Permissions(), PermModifyResources, PermManageUsers)
    case RoleBilling:
        return []Permission{PermViewUsage, PermViewCosts, PermExportReports}
    }
    return nil
}
```

---

## Use Cases

### Use Case 1: Multi-Tenancy (SaaS Provider)

**Scenario**: Brass Relay hosted as SaaS, multiple customers.

```yaml
# Customer A Control Room
id: customer-a
name: "Customer A Workspace"
tagFilter: "project:customer-a"
resources:
  - provider: openai (tagged: project:customer-a)
  - provider: anthropic (tagged: project:customer-a)
  - agent: customer-a-agent (tagged: project:customer-a)

# Customer B Control Room
id: customer-b
name: "Customer B Workspace"
tagFilter: "project:customer-b"
```

**Result**: Customer A sees only their providers and usage. Complete isolation.

### Use Case 2: Environment Separation

**Scenario**: Platform team manages dev/staging/prod.

```yaml
# Production Control Room
id: prod-platform
name: "Production Platform"
tagFilter: "env:production AND team:platform"

# Staging Control Room
id: staging-platform
name: "Staging Platform"
tagFilter: "env:staging AND team:platform"
```

**Result**: Separate dashboards for each environment with different alert thresholds.

### Use Case 3: Team-Based Views

**Scenario**: Multiple teams share infrastructure but want separate views.

```yaml
# ML Team Control Room
id: ml-team
name: "ML Research"
tagFilter: "team:ml"

# Data Team Control Room
id: data-team
name: "Data Engineering"
tagFilter: "team:data"
```

**Result**: ML team sees their models and experiments. Data team sees their pipelines.

### Use Case 4: Cost Tracking & Chargeback

**Scenario**: Finance needs cost allocation by department.

```yaml
# Engineering Costs
report: monthly
tagFilter: "cost-center:engineering"

# Research Costs
report: monthly
tagFilter: "cost-center:research"
```

**Result**: Automatic cost reports per cost center for chargeback.

### Use Case 5: Regional Operations

**Scenario**: Global deployment with regional teams.

```yaml
# US East Control Room
id: us-east-ops
name: "US East Operations"
tagFilter: "region:us-east"

# EU West Control Room
id: eu-west-ops
name: "EU West Operations"
tagFilter: "region:eu-west"
```

**Result**: Regional teams monitor local provider health and compliance.

---

## Steampunk UX Integration

### Control Room Switcher

```
┌─────────────────────────────────────────────────────────────┐
│  ⚙️ BRASS RELAY CONTROL ROOM    [Production Platform ▼]    │
├─────────────────────────────────────────────────────────────┤
│  Quick Switch:                                              │
│  ▓▓▓ Production Platform          env:production           │
│  ░░░ Staging Platform             env:staging              │
│  ░░░ ML Research                  team:ml                  │
│  ░░░ Customer A Workspace         project:customer-a       │
│                                                             │
│  [Manage Control Rooms...]                                  │
└─────────────────────────────────────────────────────────────┘
```

### Gauge Labels

| Technical Metric | Themed Display | Control Room Context |
|-----------------|----------------|---------------------|
| Request rate | Steam pressure | Boiler pressure for this room |
| Error rate | Safety valve | Line rupture risk |
| Latency | Chronometer | Telegraph transmission time |
| Token usage | Fuel consumption | Coal consumption rate |
| Cost | Ledger balance | Treasury balance |
| Provider health | Boiler status | Room's assigned boilers |

### Themed Tag Display

```
┌─────────────────────────────────────────────────────────────┐
│  Room Tags:                                                 │
│  🏭 Environment: Production    👥 Team: Platform            │
│  📁 Project: Customer-A        💰 Cost Center: Engineering  │
│  🌍 Region: US-East                                         │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Sprint 1: Foundation
- [ ] Tag data model and validation
- [ ] Taggable interface for resources
- [ ] Tag query parser

### Sprint 2: Control Rooms
- [ ] Control room CRUD API
- [ ] Resource matching logic
- [ ] Dashboard filtering

### Sprint 3: Usage & Cost
- [ ] Tagged usage tracking
- [ ] Cost aggregation per control room
- [ ] Billing report exports

### Sprint 4: UI/UX
- [ ] Control room switcher component
- [ ] Tag management interface
- [ ] Themed gauge displays

### Sprint 5: Polish
- [ ] Performance optimization
- [ ] Bulk operations
- [ ] Documentation

---

## API Examples

### Create Control Room

```bash
POST /v0/management/control-rooms
Content-Type: application/json

{
  "id": "ml-research",
  "name": "ML Research",
  "description": "Research team experiments and model testing",
  "tagFilter": "team:ml OR cost-center:research",
  "dashboardLayout": {
    "gauges": ["pressure", "latency", "tokens", "cost"],
    "theme": "brass-dark"
  }
}
```

### Tag Resources

```bash
POST /v0/management/providers/openai-production/tags
Content-Type: application/json

{
  "tags": [
    {"category": "env", "value": "production"},
    {"category": "team", "value": "platform"},
    {"category": "cost-center", "value": "engineering"}
  ]
}
```

### Get Usage for Control Room

```bash
GET /v0/management/usage?controlRoom=ml-research&start=2026-01-01&end=2026-01-31

Response:
{
  "controlRoom": "ml-research",
  "period": "2026-01-01 to 2026-01-31",
  "requests": 15420,
  "tokens": {
    "prompt": 4500000,
    "completion": 1200000,
    "total": 5700000
  },
  "cost": {
    "total": 142.50,
    "currency": "USD"
  },
  "providers": {
    "openai": { "requests": 12000, "cost": 110.00 },
    "anthropic": { "requests": 3420, "cost": 32.50 }
  }
}
```

---

## Migration Path

### From Current State

1. **Tag Existing Resources**
   - Default tag: `env:default`
   - Migrate providers to appropriate tags

2. **Create Default Control Room**
   - All resources visible by default
   - Backward compatible

3. **Gradual Adoption**
   - Teams create their own control rooms
   - Optional opt-in

---

## Open Questions

1. **Tag Inheritance**: Should resources inherit tags from their parent (e.g., provider tags applied to all its models)?

2. **Tag Limits**: Maximum tags per resource? Maximum control rooms?

3. **Performance**: How many control rooms can be active simultaneously?

4. **Cross-Room Visibility**: Should admins see across all rooms?

---

## Related Documents

- `docs/product-theme.md` - Steampunk UX guidelines
- `docs/strategy/competitive-positioning.md` - Multi-tenancy strategy
- `docs/operations/slo-and-alerting.md` - Gauge requirements
- `docs/architecture/integration-architecture.md` - Component integration

---

**Approved By**: Architecture Team (Team Alpha)
**Implementation Lead**: Team Golf (Documentation & Design)
**Target Release**: Milestone 1.2 (post-provider adapters)
