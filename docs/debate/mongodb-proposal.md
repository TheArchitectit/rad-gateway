# MongoDB for A2A Model Card Storage: The Idealist's Case

**Author**: Database Administrator (Idealist Advocate)
**Date**: 2026-02-18
**Status**: Proposal for Team Database Debate
**Target**: A2A (Agent-to-Agent) Model Card Storage

---

## Executive Summary

MongoDB is the **ideal database choice** for storing A2A Model Cards in RAD Gateway. As a document-oriented database with native JSON support, MongoDB aligns perfectly with the evolving A2A specification, enabling rapid iteration, flexible schema evolution, and horizontal scaling for multi-tenant SaaS deployments.

**Key Advantages**:
- Native JSON document storage matches A2A's JSON-based Agent Cards
- Schema flexibility accommodates the evolving A2A specification
- Horizontal scaling supports multi-tenant growth
- Rich aggregation framework enables powerful model discovery
- Document versioning tracks model iterations naturally

---

## 1. Why MongoDB Fits A2A Model Cards

### 1.1 Native JSON Alignment

A2A Model Cards are inherently JSON documents. MongoDB stores them as BSON (Binary JSON), preserving structure without transformation:

```json
{
  "name": "brass-relay-gateway",
  "description": "AI API Gateway with A2A support",
  "url": "https://gateway.brassrelay.com/a2a",
  "provider": {
    "organization": "Brass Relay",
    "url": "https://brassrelay.com"
  },
  "version": "1.0.0",
  "capabilities": {
    "streaming": true,
    "pushNotifications": false,
    "stateTransitionHistory": true
  },
  "skills": [
    {
      "id": "llm-routing",
      "name": "LLM Request Routing",
      "tags": ["llm", "routing", "gateway"],
      "examples": ["Route chat completion to best available model"]
    }
  ]
}
```

**PostgreSQL Alternative**: Requires JSONB columns with complex GIN indexes, losing document-native benefits.

### 1.2 Schema Flexibility for Evolving Specs

The A2A specification is actively evolving. MongoDB's schema-less design accommodates changes without migrations:

| A2A Evolution | MongoDB Approach | PostgreSQL Approach |
|--------------|------------------|---------------------|
| New skill parameter | Add field to new documents | ALTER TABLE, schema migration |
| Nested capability objects | Natural document nesting | JSONB with path queries |
| Optional authentication fields | Simply omit from documents | NULL columns or JSONB |
| Version-specific formats | Store different shapes in same collection | Multiple tables or complex JSONB |

### 1.3 Document Versioning

Track model card iterations naturally with MongoDB's document model:

```javascript
// Model card with embedded version history
db.modelCards.insertOne({
  "agentId": "sentiment-analyzer-v2",
  "currentVersion": {
    "name": "Sentiment Analyzer Pro",
    "version": "2.1.0",
    "skills": [...]
  },
  "versionHistory": [
    {
      "version": "2.0.0",
      "deprecatedAt": ISODate("2026-01-15"),
      "card": { /* previous version */ }
    },
    {
      "version": "1.0.0",
      "deprecatedAt": ISODate("2025-11-01"),
      "card": { /* original version */ }
    }
  ],
  "createdAt": ISODate("2025-11-01"),
  "updatedAt": ISODate("2026-02-18")
})
```

---

## 2. Schema Design Examples

### 2.1 Core Collections

```javascript
// ============================================
// Collection: modelCards
// Primary storage for A2A Agent Cards
// ============================================
{
  "_id": ObjectId("..."),
  "workspaceId": ObjectId("..."),  // Multi-tenancy
  "agentId": "sentiment-analyzer-v2",
  "status": "active",  // active, deprecated, draft

  // A2A-compliant Agent Card
  "card": {
    "name": "Sentiment Analysis Agent",
    "description": "Analyzes sentiment in customer feedback",
    "url": "https://agents.brassrelay.com/sentiment",
    "provider": {
      "organization": "Acme AI",
      "url": "https://acme.ai"
    },
    "version": "2.1.0",
    "documentationUrl": "https://docs.acme.ai/sentiment",
    "capabilities": {
      "streaming": true,
      "pushNotifications": false,
      "stateTransitionHistory": true
    },
    "authentication": {
      "schemes": ["Bearer", "OAuth2"],
      "credentials": "https://auth.acme.ai"
    },
    "defaultInputModes": ["text"],
    "defaultOutputModes": ["text", "structured"],
    "skills": [
      {
        "id": "analyze-sentiment",
        "name": "Analyze Sentiment",
        "description": "Returns sentiment score and classification",
        "tags": ["sentiment", "nlp", "classification"],
        "examples": ["How positive is this review?"],
        "inputModes": ["text"],
        "outputModes": ["structured"],
        "parameters": {
          "type": "object",
          "properties": {
            "text": {"type": "string"},
            "language": {"type": "string", "default": "auto"}
          }
        }
      }
    ]
  },

  // Indexing and search
  "tags": ["nlp", "sentiment", "customer-feedback"],
  "categories": ["analytics", "text-processing"],

  // Operational metadata
  "healthStatus": {
    "lastCheck": ISODate("2026-02-18T10:30:00Z"),
    "healthy": true,
    "latencyMs": 45,
    "consecutiveFailures": 0
  },

  // RBAC
  "ownerId": ObjectId("..."),
  "visibility": "public",  // public, workspace, private
  "allowedWorkspaces": [ObjectId("...")],  // For cross-workspace sharing

  // Timestamps
  "createdAt": ISODate("2025-11-01T00:00:00Z"),
  "updatedAt": ISODate("2026-02-18T08:15:00Z"),
  "version": 5  // Optimistic locking
}

// Indexes
db.modelCards.createIndex({"workspaceId": 1, "agentId": 1}, {unique: true})
db.modelCards.createIndex({"card.skills.id": 1})
db.modelCards.createIndex({"card.skills.tags": 1})
db.modelCards.createIndex({"tags": 1})
db.modelCards.createIndex({"card.name": "text", "card.description": "text"})
db.modelCards.createIndex({"healthStatus.healthy": 1, "healthStatus.lastCheck": 1})
```

```javascript
// ============================================
// Collection: skillRegistry
// Denormalized skill index for fast discovery
// ============================================
{
  "_id": ObjectId("..."),
  "skillId": "analyze-sentiment",
  "agentId": "sentiment-analyzer-v2",
  "workspaceId": ObjectId("..."),

  // Skill definition (denormalized from modelCards)
  "skill": {
    "id": "analyze-sentiment",
    "name": "Analyze Sentiment",
    "description": "Returns sentiment score and classification",
    "tags": ["sentiment", "nlp", "classification"],
    "inputModes": ["text"],
    "outputModes": ["structured"],
    "parameters": { /* JSON Schema */ }
  },

  // Discovery metadata
  "popularity": {
    "totalInvocations": 15432,
    "avgLatencyMs": 45,
    "successRate": 0.987
  },

  // Search optimization
  "searchTokens": ["sentiment", "analyze", "nlp", "classification", "emotion"],

  "createdAt": ISODate("2025-11-01T00:00:00Z"),
  "updatedAt": ISODate("2026-02-18T08:15:00Z")
}

// Indexes
db.skillRegistry.createIndex({"skillId": 1, "workspaceId": 1})
db.skillRegistry.createIndex({"skill.tags": 1})
db.skillRegistry.createIndex({"searchTokens": 1})
db.skillRegistry.createIndex({"popularity.totalInvocations": -1})
```

```javascript
// ============================================
// Collection: modelCardVersions
// Version history for audit and rollback
// ============================================
{
  "_id": ObjectId("..."),
  "agentId": "sentiment-analyzer-v2",
  "workspaceId": ObjectId("..."),
  "version": "2.0.0",
  "card": { /* Full card snapshot */ },
  "changeReason": "Added multi-language support",
  "changedBy": ObjectId("..."),
  "changedAt": ISODate("2026-01-15T00:00:00Z"),
  "isActive": false
}

// Indexes
db.modelCardVersions.createIndex({"agentId": 1, "version": -1})
```

### 2.2 Supporting Collections

```javascript
// ============================================
// Collection: agentTasks
// A2A task storage with state machine
// ============================================
{
  "_id": ObjectId("..."),
  "taskId": "task-uuid-v4",
  "workspaceId": ObjectId("..."),

  // Task lifecycle
  "status": "completed",  // submitted, working, input-required, completed, canceled, failed
  "sessionId": "session-uuid",
  "parentId": null,  // For subtasks

  // A2A request/response
  "request": {
    "message": {
      "role": "user",
      "parts": [{"type": "text", "text": "Analyze this feedback"}]
    },
    "metadata": {"priority": "high"}
  },

  "response": {
    "artifacts": [
      {
        "type": "structured",
        "name": "sentiment-result",
        "parts": [{
          "type": "data",
          "data": {"score": 0.85, "label": "positive"}
        }]
      }
    ]
  },

  // Routing
  "sourceAgent": "client-dashboard",
  "targetAgent": "sentiment-analyzer-v2",
  "delegationChain": ["client-dashboard"],

  // State transitions
  "transitions": [
    {"from": "submitted", "to": "working", "at": ISODate("...")},
    {"from": "working", "to": "completed", "at": ISODate("...")}
  ],

  // Error handling
  "error": null,

  // TTL for cleanup
  "expiresAt": ISODate("2026-02-25T00:00:00Z"),

  "createdAt": ISODate("2026-02-18T10:00:00Z"),
  "updatedAt": ISODate("2026-02-18T10:00:45Z"),
  "completedAt": ISODate("2026-02-18T10:00:45Z")
}

// Indexes with TTL
db.agentTasks.createIndex({"taskId": 1}, {unique: true})
db.agentTasks.createIndex({"sessionId": 1})
db.agentTasks.createIndex({"status": 1, "createdAt": -1})
db.agentTasks.createIndex({"expiresAt": 1}, {expireAfterSeconds: 0})  // TTL index
```

---

## 3. Implementation Plan

### 3.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    RAD Gateway (Go)                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   A2A       │  │   MongoDB   │  │   Discovery         │  │
│  │   Handlers  │──│   Driver    │──│   Service           │  │
│  └─────────────┘  │  (mongo-go) │  │                     │  │
│         │         └─────────────┘  └─────────────────────┘  │
│         │                                                   │
│  ┌─────────────┐                                            │
│  │  Change     │◄──────────────────────────────────────┐    │
│  │  Streams    │                                       │    │
│  └─────────────┘                                       │    │
└────────────────────────────────────────────────────────┼────┘
                                                         │
                              ┌──────────────────────────┼────┐
                              │      MongoDB Cluster     │    │
                              │  ┌─────────────────────┐ │    │
                              │  │   Replica Set       │◄┘    │
                              │  │   (Primary + 2x Sec)│      │
                              │  └─────────────────────┘      │
                              │           │                   │
                              │  ┌────────▼────────┐          │
                              │  │  Change Streams │          │
                              │  │  (Real-time)    │          │
                              │  └─────────────────┘          │
                              └───────────────────────────────┘
```

### 3.2 Go Integration

```go
// internal/a2a/mongodb/store.go

package mongodb

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "radgateway/internal/a2a"
)

type ModelCardStore struct {
    client          *mongo.Client
    database        *mongo.Database
    modelCards      *mongo.Collection
    skillRegistry   *mongo.Collection
    tasks           *mongo.Collection
}

func NewStore(ctx context.Context, uri string) (*ModelCardStore, error) {
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }

    db := client.Database("radgateway_a2a")

    store := &ModelCardStore{
        client:        client,
        database:      db,
        modelCards:    db.Collection("modelCards"),
        skillRegistry: db.Collection("skillRegistry"),
        tasks:         db.Collection("agentTasks"),
    }

    if err := store.createIndexes(ctx); err != nil {
        return nil, err
    }

    return store, nil
}

func (s *ModelCardStore) createIndexes(ctx context.Context) error {
    // Model cards indexes
    modelCardIndexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{"workspaceId", 1}, {"agentId", 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            Keys: bson.D{{"card.skills.tags", 1}},
        },
        {
            Keys: bson.D{{"card.name", "text"}, {"card.description", "text"}},
        },
        {
            Keys: bson.D{{"healthStatus.healthy", 1}, {"healthStatus.lastCheck", 1}},
        },
    }

    if _, err := s.modelCards.Indexes().CreateMany(ctx, modelCardIndexes); err != nil {
        return err
    }

    // Skill registry indexes
    skillIndexes := []mongo.IndexModel{
        {
            Keys: bson.D{{"skillId", 1}, {"workspaceId", 1}},
        },
        {
            Keys: bson.D{{"skill.tags", 1}},
        },
        {
            Keys: bson.D{{"popularity.totalInvocations", -1}},
        },
    }

    if _, err := s.skillRegistry.Indexes().CreateMany(ctx, skillIndexes); err != nil {
        return err
    }

    // Task indexes with TTL
    taskIndexes := []mongo.IndexModel{
        {
            Keys:    bson.D{{"taskId", 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            Keys:    bson.D{{"expiresAt", 1}},
            Options: options.Index().SetExpireAfterSeconds(0),
        },
    }

    _, err := s.tasks.Indexes().CreateMany(ctx, taskIndexes)
    return err
}

// CreateModelCard stores a new A2A model card
func (s *ModelCardStore) CreateModelCard(ctx context.Context, card *a2a.ModelCard) error {
    card.CreatedAt = time.Now()
    card.UpdatedAt = time.Now()
    card.Version = 1

    _, err := s.modelCards.InsertOne(ctx, card)
    return err
}

// FindModelCardsBySkill discovers agents by skill tags
func (s *ModelCardStore) FindModelCardsBySkill(
    ctx context.Context,
    workspaceId string,
    tags []string,
) ([]*a2a.ModelCard, error) {
    filter := bson.M{
        "workspaceId":       workspaceId,
        "status":            "active",
        "card.skills.tags":  bson.M{"$in": tags},
        "healthStatus.healthy": true,
    }

    opts := options.Find().
        SetSort(bson.D{{"healthStatus.latencyMs", 1}}).
        SetLimit(100)

    cursor, err := s.modelCards.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var cards []*a2a.ModelCard
    if err := cursor.All(ctx, &cards); err != nil {
        return nil, err
    }

    return cards, nil
}

// SearchModelCards performs full-text search on agent cards
func (s *ModelCardStore) SearchModelCards(
    ctx context.Context,
    workspaceId string,
    query string,
) ([]*a2a.ModelCard, error) {
    filter := bson.M{
        "$text": bson.M{"$search": query},
        "workspaceId": workspaceId,
        "status": "active",
    }

    opts := options.Find().
        SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}}).
        SetSort(bson.D{{"score", bson.M{"$meta": "textScore"}}})

    cursor, err := s.modelCards.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var cards []*a2a.ModelCard
    if err := cursor.All(ctx, &cards); err != nil {
        return nil, err
    }

    return cards, nil
}

// AggregateSkills returns skill statistics using MongoDB aggregation
func (s *ModelCardStore) AggregateSkills(
    ctx context.Context,
    workspaceId string,
) ([]SkillStats, error) {
    pipeline := mongo.Pipeline{
        {{
            "$match", bson.M{
                "workspaceId": workspaceId,
                "status": "active",
            },
        }},
        {{"$unwind", "$card.skills"}},
        {{
            "$group", bson.M{
                "_id": "$card.skills.id",
                "skillName": bson.M{"$first": "$card.skills.name"},
                "agentCount": bson.M{"$sum": 1},
                "tags": bson.M{"$addToSet": "$card.skills.tags"},
            },
        }},
        {{"$sort", bson.M{"agentCount": -1}}},
    }

    cursor, err := s.modelCards.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var stats []SkillStats
    if err := cursor.All(ctx, &stats); err != nil {
        return nil, err
    }

    return stats, nil
}

// WatchModelCards subscribes to change streams for real-time updates
func (s *ModelCardStore) WatchModelCards(
    ctx context.Context,
    workspaceId string,
) (*mongo.ChangeStream, error) {
    pipeline := mongo.Pipeline{{
        {"$match", bson.M{
            "fullDocument.workspaceId": workspaceId,
        }},
    }}

    opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)

    return s.modelCards.Watch(ctx, pipeline, opts)
}
```

### 3.3 Integration with Existing PostgreSQL

MongoDB complements PostgreSQL for A2A-specific data while maintaining existing relational data:

| Data Type | PostgreSQL | MongoDB |
|-----------|-----------|---------|
| Users, RBAC | ✅ | ❌ |
| API Keys | ✅ | ❌ |
| Usage Records | ✅ | ❌ |
| A2A Model Cards | ❌ | ✅ |
| A2A Tasks | ❌ | ✅ |
| Skill Registry | ❌ | ✅ |
| Real-time Discovery | ❌ | ✅ |

**Synchronization Strategy**:
```go
// When creating API key in PostgreSQL, create reference in MongoDB
func (s *Service) CreateAPIKey(ctx context.Context, req CreateKeyRequest) (*APIKey, error) {
    // 1. Create in PostgreSQL (source of truth)
    key, err := s.pgStore.CreateAPIKey(ctx, req)
    if err != nil {
        return nil, err
    }

    // 2. Sync to MongoDB for A2A lookups
    if err := s.mongoStore.SyncAPIKeyReference(ctx, key); err != nil {
        // Log but don't fail - MongoDB is secondary
        s.logger.Warn("failed to sync API key to MongoDB", "error", err)
    }

    return key, nil
}
```

---

## 4. Cost and Operational Considerations

### 4.1 Deployment Options

| Option | Cost | Best For |
|--------|------|----------|
| MongoDB Community (Self-hosted) | Infrastructure only | Development, small deployments |
| MongoDB Atlas Serverless | Pay-per-operation | Variable workloads, startups |
| MongoDB Atlas M10+ | ~$60/month base | Production, guaranteed performance |
| MongoDB Atlas Cluster (Multi-region) | ~$200+/month | HA, disaster recovery |

### 4.2 Resource Requirements

**Small Deployment** (Development/Small Team):
- 1x MongoDB instance (2 CPU, 4GB RAM)
- 20GB storage
- Estimated cost: $30-50/month (self-hosted) or $60/month (Atlas)

**Medium Deployment** (Production, Single Region):
- 3x MongoDB replica set (2 CPU, 8GB RAM each)
- 100GB SSD storage
- Estimated cost: $150-200/month (self-hosted) or $200/month (Atlas M30)

**Large Deployment** (Multi-tenant SaaS):
- Sharded cluster (2 shards, 3 nodes each)
- 500GB+ SSD storage
- Estimated cost: $500-800/month (Atlas M50+)

### 4.3 Operational Complexity

**Advantages**:
- Automatic failover with replica sets
- Online indexing (no downtime)
- Built-in monitoring (Atlas) or Prometheus exporter
- Automated backups (point-in-time recovery)

**Considerations**:
- New technology to learn for PostgreSQL-focused team
- Requires understanding of eventual consistency (for reads from secondaries)
- Separate connection pooling and monitoring

### 4.4 Comparison with PostgreSQL JSONB

| Feature | MongoDB | PostgreSQL JSONB |
|---------|---------|------------------|
| Native JSON | ✅ Native BSON | ✅ JSONB |
| Schema validation | ✅ JSON Schema | ✅ JSON Schema (v14+) |
| Text search | ✅ Full-text indexes | ✅ GIN indexes |
| Horizontal scaling | ✅ Sharding | ❌ Complex (Citus) |
| Aggregation | ✅ Pipeline framework | ✅ JSONB functions |
| Change streams | ✅ Built-in | ❌ Requires triggers |
| Document size limit | 16MB | 1GB (TOAST) |
| Array operations | ✅ Native | ⚠️ Complex |
| Nested queries | ✅ Dot notation | ⚠️ Path operators |

---

## 5. Addressing Concerns

### 5.1 "We Already Have PostgreSQL"

**Response**: MongoDB complements PostgreSQL, it doesn't replace it:
- Keep PostgreSQL for relational data (users, RBAC, usage records)
- Use MongoDB for document-native A2A data
- Each database handles what it does best

### 5.2 "Operational Overhead"

**Response**: Modern MongoDB reduces operational burden:
- MongoDB Atlas provides fully managed service
- Automated backups, patching, and scaling
- Integration with existing observability stack (Prometheus, Grafana)
- Go driver is mature and well-maintained

### 5.3 "Team Learning Curve"

**Response**: Minimal learning curve for Go developers:
- MongoDB query syntax is intuitive (JSON-like)
- Go driver follows standard patterns
- A2A domain maps naturally to documents
- Can start with simple CRUD, add complexity incrementally

### 5.4 "Data Consistency"

**Response**: MongoDB offers tunable consistency:
- Default: Strong consistency (read from primary)
- Optional: Eventual consistency (read from secondaries)
- Multi-document ACID transactions available (since v4.0)
- For A2A use case: eventual consistency is often acceptable

---

## 6. Success Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| Model Card Query Latency | < 10ms (p99) | MongoDB profiler |
| Skill Discovery Latency | < 50ms (p99) | Application metrics |
| Write Throughput | > 1000 ops/sec | Load testing |
| Availability | 99.9% | Uptime monitoring |
| Schema Migration Time | 0 minutes | No migrations needed |
| Developer Velocity | +20% | Sprint retrospectives |

---

## 7. Conclusion

MongoDB is the **ideal choice** for A2A Model Card storage because:

1. **Natural Fit**: A2A Agent Cards are JSON documents - MongoDB stores them natively
2. **Schema Evolution**: The A2A spec will evolve - MongoDB accommodates change without migrations
3. **Discovery Power**: Rich aggregation framework enables sophisticated model discovery
4. **Horizontal Scaling**: Sharding supports multi-tenant SaaS growth
5. **Document Versioning**: Track model iterations naturally

**Recommendation**: Implement MongoDB alongside PostgreSQL for A2A-specific data, leveraging each database for its strengths.

---

## Appendix A: Go Dependencies

```go
// go.mod
require (
    go.mongodb.org/mongo-driver v1.14.0
)
```

## Appendix B: Docker Compose for Development

```yaml
version: '3.8'
services:
  mongodb:
    image: mongo:7.0
    ports:
      - "27017:27017"
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: password
      MONGO_INITDB_DATABASE: radgateway_a2a
    volumes:
      - mongodb_data:/data/db
    command: ["--replSet", "rs0", "--bind_ip_all"]

  # Initialize replica set
  mongo-init:
    image: mongo:7.0
    depends_on:
      - mongodb
    entrypoint: >
      bash -c "
        sleep 5 &&
        mongosh --host mongodb:27017 -u admin -p password --authenticationDatabase admin --eval 'rs.initiate({_id: \"rs0\", members: [{_id: 0, host: \"mongodb:27017\"}]})'
      "

volumes:
  mongodb_data:
```

---

*Document Version*: 1.0
*Last Updated*: 2026-02-18
*Owner*: Database Administrator (Idealist)
*Next Review*: After team debate
