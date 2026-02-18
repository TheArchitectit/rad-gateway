# PostgreSQL Proposal for A2A Model Card Storage

**Role**: The Pessimist (Team Database Debate)
**Position**: FOR PostgreSQL, AGAINST MongoDB
**Date**: 2026-02-18

## Executive Summary

Adding MongoDB for A2A model card storage is a mistake. We already run PostgreSQL for Infisical, have established connection pools, monitoring, and operational expertise. Why introduce another database, another set of backups, another monitoring stack, and another attack surface? PostgreSQL with JSONB handles flexible schemas without sacrificing ACID guarantees. This proposal demonstrates why PostgreSQL is the pragmatic, operationally sound choice.

## The Case Against Adding MongoDB

### 1. Operational Reality Check

| Factor | PostgreSQL | MongoDB |
|--------|-----------|---------|
| **Already Running** | Yes (Infisical uses it) | No (new infrastructure) |
| **Backup Strategy** | Established | Needs new tooling |
| **Monitoring** | Prometheus metrics active | New dashboards needed |
| **Team Expertise** | SQL knowledge exists | Learning curve required |
| **Security Hardening** | Completed | Starts from zero |
| **Connection Pooling** | Configured | Needs setup |

**Reality**: Every new database is a new system to break at 3 AM. PostgreSQL is already battle-tested in our environment.

### 2. The "Flexible Schema" Fallacy

MongoDB advocates claim document stores handle changing schemas better. But:

- **A2A Agent Cards have a defined spec** - they're not truly schemaless
- **PostgreSQL JSONB** handles nested structures, arrays, and dynamic fields
- **Schema validation** can be enforced at the application layer or via PostgreSQL's JSON Schema constraints

```sql
-- PostgreSQL handles nested JSON without drama
SELECT
    card->>'name' as agent_name,
    card->'capabilities'->>'streaming' as supports_streaming,
    jsonb_array_elements(card->'skills')->>'id' as skill_id
FROM a2a_agent_cards
WHERE card @> '{"capabilities": {"streaming": true}}';
```

### 3. The Consistency Problem

Model cards contain critical metadata:
- Authentication requirements
- Rate limits
- Access control rules
- Cost attribution data

**Eventual consistency is unacceptable here.** When an admin updates an agent's authentication scheme, all subsequent requests MUST see the new configuration. PostgreSQL provides immediate consistency. MongoDB's default eventual consistency risks security and billing errors.

### 4. Transaction Complexity

Consider the A2A workflow:

```
1. Create task -> INSERT tasks table
2. Update agent health -> UPDATE agent_cards table
3. Record usage -> INSERT usage_records
4. Update quota -> UPDATE quota_assignments
```

With PostgreSQL, wrap in a transaction. All succeed or all fail. Data stays consistent.

With MongoDB across multiple collections or databases, you need:
- Application-level two-phase commits
- Saga patterns
- Eventual consistency reconciliation jobs

**Complexity we don't need.**

## PostgreSQL JSONB Schema Design

### Core Tables

```sql
-- A2A Agent Cards stored as JSONB
CREATE TABLE a2a_agent_cards (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    slug TEXT NOT NULL,

    -- The full Agent Card as JSONB
    card JSONB NOT NULL,

    -- Extracted fields for querying (computed or synced)
    name TEXT GENERATED ALWAYS AS (card->>'name') STORED,
    provider_org TEXT GENERATED ALWAYS AS (card->'provider'->>'organization') STORED,
    is_active BOOLEAN DEFAULT TRUE,

    -- Status tracking
    last_health_check_at TIMESTAMP,
    health_status TEXT DEFAULT 'unknown',
    consecutive_failures INTEGER DEFAULT 0,

    -- Metadata
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(workspace_id, slug)
);

-- Indexes for common queries
CREATE INDEX idx_agent_cards_workspace ON a2a_agent_cards(workspace_id);
CREATE INDEX idx_agent_cards_name ON a2a_agent_cards(name);
CREATE INDEX idx_agent_cards_health ON a2a_agent_cards(health_status) WHERE is_active = TRUE;

-- GIN index for JSONB queries (essential for performance)
CREATE INDEX idx_agent_cards_card_gin ON a2a_agent_cards USING GIN (card);

-- Specific indexes for common JSONB lookups
CREATE INDEX idx_agent_cards_capabilities ON a2a_agent_cards USING GIN ((card->'capabilities'));
CREATE INDEX idx_agent_cards_skills ON a2a_agent_cards USING GIN ((card->'skills'));
```

### Task Storage with JSONB

```sql
-- A2A Tasks with flexible metadata
CREATE TABLE a2a_tasks (
    id TEXT PRIMARY KEY,
    parent_id TEXT REFERENCES a2a_tasks(id),
    session_id TEXT NOT NULL,

    -- Task state
    status TEXT NOT NULL DEFAULT 'submitted',

    -- Core A2A structures as JSONB
    message JSONB NOT NULL,        -- A2A Message structure
    metadata JSONB DEFAULT '{}',  -- Flexible task metadata
    artifacts JSONB DEFAULT '[]', -- A2A Artifact array
    error JSONB,                  -- TaskError when failed

    -- Routing
    source_agent_id TEXT REFERENCES a2a_agent_cards(id),
    target_agent_id TEXT REFERENCES a2a_agent_cards(id),
    delegation_chain TEXT[],      -- PostgreSQL array type

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours'),

    -- Indexes
    CONSTRAINT valid_status CHECK (status IN (
        'submitted', 'working', 'input-required',
        'completed', 'canceled', 'failed'
    ))
);

CREATE INDEX idx_tasks_session ON a2a_tasks(session_id);
CREATE INDEX idx_tasks_status ON a2a_tasks(status);
CREATE INDEX idx_tasks_target ON a2a_tasks(target_agent_id, status);
CREATE INDEX idx_tasks_expires ON a2a_tasks(expires_at) WHERE status NOT IN ('completed', 'canceled', 'failed');

-- GIN indexes for JSON queries
CREATE INDEX idx_tasks_message_gin ON a2a_tasks USING GIN (message);
CREATE INDEX idx_tasks_metadata_gin ON a2a_tasks USING GIN (metadata);
```

### Skill Indexing Table

```sql
-- Separate table for skill-based queries (optional optimization)
CREATE TABLE a2a_agent_skills (
    id SERIAL PRIMARY KEY,
    agent_card_id TEXT NOT NULL REFERENCES a2a_agent_cards(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL,
    skill_name TEXT NOT NULL,
    skill_data JSONB NOT NULL,  -- Full skill definition

    -- Extracted for querying
    tags TEXT[],  -- PostgreSQL array
    input_modes TEXT[],
    output_modes TEXT[],

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_skills_agent ON a2a_agent_skills(agent_card_id);
CREATE INDEX idx_skills_id ON a2a_agent_skills(skill_id);
CREATE INDEX idx_skills_tags ON a2a_agent_skills USING GIN (tags);
CREATE INDEX idx_skills_modes ON a2a_agent_skills USING GIN (input_modes, output_modes);
```

## JSONB Query Patterns

### 1. Find Agents by Capability

```sql
-- Find all agents that support streaming
SELECT id, name, card->>'url' as endpoint
FROM a2a_agent_cards
WHERE card @> '{"capabilities": {"streaming": true}}';
```

### 2. Skill-Based Search

```sql
-- Find agents with specific skills
SELECT DISTINCT ac.id, ac.name
FROM a2a_agent_cards ac
JOIN a2a_agent_skills s ON s.agent_card_id = ac.id
WHERE s.tags @> ARRAY['sentiment-analysis']
  AND s.input_modes @> ARRAY['text'];
```

### 3. Complex JSON Extraction

```sql
-- Extract nested authentication config
SELECT
    id,
    card->>'name' as agent_name,
    card->'authentication'->>'schemes' as auth_schemes,
    jsonb_array_elements_text(card->'defaultInputModes') as input_mode
FROM a2a_agent_cards
WHERE card->'authentication' @> '{"schemes": ["OAuth2"]}';
```

### 4. Partial Updates

```sql
-- Update specific nested field without rewriting entire document
UPDATE a2a_agent_cards
SET
    card = jsonb_set(
        card,
        '{capabilities,streaming}',
        'false'::jsonb
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE id = 'agent-123';
```

## Migration Strategy from SQLite

### Phase 1: Schema Creation

```sql
-- Run in PostgreSQL
-- Tables created as shown above
-- Indexes created concurrently to avoid locks
```

### Phase 2: Data Migration

```go
// migration.go
func MigrateFromSQLite(sqliteDB, postgresDB *sql.DB) error {
    // Export from SQLite
    rows, err := sqliteDB.Query(`
        SELECT id, workspace_id, slug, settings, created_at, updated_at
        FROM providers WHERE provider_type = 'a2a-agent'
    `)
    if err != nil {
        return err
    }
    defer rows.Close()

    // Transform and import to PostgreSQL
    tx, err := postgresDB.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.Prepare(`
        INSERT INTO a2a_agent_cards (id, workspace_id, slug, card, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `)
    if err != nil {
        return err
    }

    for rows.Next() {
        var id, workspaceID, slug string
        var settings []byte
        var createdAt, updatedAt time.Time

        if err := rows.Scan(&id, &workspaceID, &slug, &settings, &createdAt, &updatedAt); err != nil {
            return err
        }

        // Transform settings JSON to Agent Card format
        card := transformToAgentCard(settings)

        _, err = stmt.Exec(id, workspaceID, slug, card, createdAt, updatedAt)
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### Phase 3: Dual-Write Period

```go
// During migration, write to both databases
func (s *Service) CreateAgentCard(ctx context.Context, card *AgentCard) error {
    // Write to PostgreSQL (new source of truth)
    if err := s.pgStore.Create(ctx, card); err != nil {
        return err
    }

    // Async write to SQLite for backward compatibility
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _ = s.sqliteStore.Create(ctx, card)
    }()

    return nil
}
```

### Phase 4: Cutover

1. **Read from PostgreSQL**: Update application to read from PostgreSQL
2. **Validation**: Run consistency checks between stores
3. **Decommission SQLite**: Remove SQLite A2A tables after validation

## Operational Cost Comparison

### Infrastructure Costs

| Item | PostgreSQL (Existing) | MongoDB (New) |
|------|---------------------|---------------|
| **Compute** | $0 (reuse existing) | $200-500/mo (new instances) |
| **Storage** | $20/mo (incremental) | $100/mo (minimum) |
| **Backup** | $10/mo (incremental) | $50/mo (new system) |
| **Monitoring** | $0 (existing dashboards) | $100/mo (new tooling) |
| **Network** | $0 (same VPC) | $50/mo (new peering) |
| **Total Monthly** | **$30** | **$500-800** |

### Engineering Costs

| Activity | PostgreSQL | MongoDB |
|----------|-----------|---------|
| **Setup** | 0 days (done) | 3-5 days |
| **Learning Curve** | 0 days (known) | 5-10 days |
| **Migration Development** | 2 days | 5 days |
| **Testing** | 2 days | 5 days |
| **Security Hardening** | 0 days (done) | 3-5 days |
| **Monitoring Setup** | 1 day | 3 days |
| **Runbook Creation** | 1 day | 3 days |
| **Total Engineering** | **6 days** | **27-36 days** |

**At $1500/day loaded cost**: PostgreSQL = $9,000, MongoDB = $40,500-$54,000

### Ongoing Maintenance

| Activity | PostgreSQL | MongoDB |
|----------|-----------|---------|
| **Weekly Health Checks** | Existing process | New process (+30 min/week) |
| **Patch Management** | Existing automation | New automation (+4 hours/quarter) |
| **Backup Verification** | Existing | New (+2 hours/month) |
| **Performance Tuning** | Known patterns | Learning required (+8 hours/quarter) |
| **Incident Response** | Team familiar | On-call learning curve |

## Risk Analysis: Why MongoDB is Risky

### Risk 1: Split-Brain in Production

With two databases, application logic must decide which to query. Incorrect routing causes:
- Stale data reads
- Split-brain scenarios
- Data inconsistency

**PostgreSQL**: Single source of truth. No routing logic needed.

### Risk 2: Backup Consistency

Backing up PostgreSQL + MongoDB consistently requires:
- Distributed transactions
- Coordinated snapshots
- Point-in-time recovery coordination

**PostgreSQL**: Single backup stream. Consistent by design.

### Risk 3: Transaction Boundaries

Cross-database transactions require:
- Saga patterns
- Outbox patterns
- Eventual consistency reconciliation

**PostgreSQL**: ACID transactions across all tables. Simple, correct.

### Risk 4: Security Surface Area

Every database adds:
- New authentication system
- New authorization model
- New audit logging
- New encryption keys
- New compliance scope

**PostgreSQL**: Security already hardened. Audit logging in place.

## Performance: PostgreSQL JSONB is Fast Enough

### Benchmark Assumptions

- 10,000 Agent Cards
- Average card size: 10KB
- 100 concurrent reads
- 10 writes/second

### Query Performance

| Query Type | PostgreSQL JSONB | MongoDB |
|------------|------------------|---------|
| **Fetch by ID** | 1-2ms | 1-2ms |
| **GIN Index Query** | 5-10ms | 5-15ms |
| **Full-Text Search** | 20-50ms | 15-40ms |
| **Aggregation** | 50-100ms | 30-80ms |

**Verdict**: Comparable performance for our use case.

### Write Performance

| Operation | PostgreSQL | MongoDB |
|-----------|-----------|---------|
| **Insert** | 5ms | 3ms |
| **Update (JSONB)** | 8ms | 5ms |
| **Partial Update** | 5ms | 5ms |

**Verdict**: MongoDB slightly faster, but difference negligible for our scale.

### Scaling Considerations

Our projections:
- Year 1: 1,000 agents
- Year 3: 10,000 agents
- Each agent card: ~10KB
- Total data: ~100MB

PostgreSQL handles 100MB without breaking a sweat. We don't need horizontal sharding yet.

## Counter-Arguments Addressed

### "But MongoDB is better for JSON!"

PostgreSQL JSONB:
- Binary storage (not text like JSON)
- Indexable with GIN indexes
- Supports partial updates
- Schema validation available
- ACID transactions

**Verdict**: PostgreSQL JSONB is production-proven for JSON workloads.

### "But what if we need to scale horizontally?"

- Current scale: ~100MB
- PostgreSQL vertical scaling: Handles TBs
- Read replicas: Easy with PostgreSQL
- When we outgrow single-node PostgreSQL, we'll have revenue to fund proper architecture

**Verdict**: Premature optimization. PostgreSQL scales further than most teams need.

### "But the developers know MongoDB!"

- Team knows SQL from Infisical
- A2A Agent Cards are read-heavy, analytical workloads
- SQL is better for joins, aggregations, reporting

**Verdict**: SQL knowledge exists. JSONB learning curve is minimal.

### "But MongoDB Atlas is managed!"

- We're already managing PostgreSQL for Infisical
- Another managed service = another vendor relationship
- Another bill to track
- Another compliance audit

**Verdict**: Operational overhead of "managed" still exists.

## Conclusion: The Pessimist's Verdict

Adding MongoDB is a solution looking for a problem. We have:

✅ **PostgreSQL running** (Infisical uses it)
✅ **Team SQL expertise**
✅ **JSONB for flexible schemas**
✅ **ACID transactions**
✅ **Existing monitoring and backups**
✅ **Security hardening complete**

Why add complexity, cost, and operational risk for marginal gains?

**The Pessimist's Recommendation**: Use PostgreSQL with JSONB. It meets all requirements, leverages existing infrastructure, and keeps operational costs low. Save MongoDB consideration for when we have a problem PostgreSQL can't solve.

---

## Appendix: Implementation Checklist

### Immediate Actions

- [ ] Create `a2a_agent_cards` table with JSONB
- [ ] Create `a2a_tasks` table with JSONB
- [ ] Add GIN indexes for JSONB columns
- [ ] Implement migration from existing SQLite providers table

### Performance Optimization

- [ ] Set `autovacuum` for frequent-updated JSONB tables
- [ ] Monitor GIN index usage with `pg_stat_user_indexes`
- [ ] Configure `work_mem` for complex JSONB queries

### Operational Readiness

- [ ] Add PostgreSQL metrics to existing dashboards
- [ ] Create runbook for JSONB query optimization
- [ ] Document JSONB update patterns

---

*Document Version*: 1.0
*Pessimist's Confidence Level*: 95% that PostgreSQL is the right choice
*Risk Assessment*: Low (proven technology, existing infrastructure)
