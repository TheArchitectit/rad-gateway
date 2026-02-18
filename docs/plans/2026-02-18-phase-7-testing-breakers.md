# Phase 7: The Breakers (Testing & QA)

## Overview

Deploy comprehensive testing infrastructure for RAD Gateway to ensure provider adapter parity, A2A model card functionality, and performance benchmarks.

## Context

- **Current Test Coverage**: 35+ test files across providers, auth, streaming, cost, RBAC
- **Recent Changes**: Hybrid database (PostgreSQL JSONB + Redis), A2A model cards, JWT hardening
- **Build Issue**: `cmd/migrate/main.go` has incorrect module path
- **Worktree**: `/mnt/ollama/git/RADAPI01/.worktrees/phase-7-testing`

## Goals

1. Fix pre-existing build issues
2. Create contract tests for provider adapters (OpenAI, Anthropic, Gemini)
3. Build integration tests for A2A model card CRUD operations
4. Establish performance benchmarks for hybrid database
5. Create regression test suite

## Tasks

### Task 1: Fix Build Issues
**Assigned to**: Fixer Agent
**Priority**: Critical

Fix `cmd/migrate/main.go`:
- Change `github.com/anthropics/rad-gateway/internal/db` to `radgateway/internal/db`
- Ensure migrations directory exists or adjust embed pattern
- Verify `go build ./...` succeeds

### Task 2: Provider Contract Tests
**Assigned to**: Contract Test Specialist
**Priority**: High

Create comprehensive contract tests for:
- OpenAI adapter (`internal/provider/openai/`)
- Anthropic adapter (`internal/provider/anthropic/`)
- Gemini adapter (`internal/provider/gemini/`)

Each provider needs:
- Request/response transformation validation
- Error handling contract tests
- Streaming response contract tests
- Mock provider deterministic responses

### Task 3: A2A Model Card Integration Tests
**Assigned to**: SDET
**Priority**: High

Test A2A model card CRUD with hybrid database:
- Create model card (writes to PostgreSQL + Redis)
- Read with cache hit (Redis)
- Read with cache miss (PostgreSQL fallback)
- Update with cache invalidation
- Delete with cache cleanup
- Cache TTL expiration
- Connection pool behavior under load

### Task 4: Performance Benchmarks
**Assigned to**: Performance Engineer
**Priority**: Medium

Create benchmarks in `tests/benchmarks/`:
- Database query latency (PostgreSQL vs Redis)
- Cache hit/miss ratio impact
- JWT validation throughput
- Provider adapter latency
- End-to-end request latency

Use Go's `testing.B` with sub-benchmarks.

### Task 5: Regression Test Suite
**Assigned to**: QA Architect
**Priority**: Medium

Create regression suite covering:
- All critical paths (auth, routing, providers)
- Security fixes verification (admin auth, JWT)
- Database migration paths
- Configuration loading
- Secret management

## Deliverables

1. `cmd/migrate/main.go` - Fixed module imports
2. `tests/contract/openai_test.go` - OpenAI contract tests
3. `tests/contract/anthropic_test.go` - Anthropic contract tests
4. `tests/contract/gemini_test.go` - Gemini contract tests
5. `tests/integration/a2a_model_cards_test.go` - A2A integration tests
6. `tests/benchmarks/database_bench_test.go` - Database benchmarks
7. `tests/benchmarks/jwt_bench_test.go` - JWT benchmarks
8. `tests/regression/critical_paths_test.go` - Regression suite

## Success Criteria

- [ ] `go build ./...` succeeds with no errors
- [ ] `go test ./...` passes (all existing + new tests)
- [ ] Contract tests validate provider API parity
- [ ] A2A model card tests verify hybrid database behavior
- [ ] Benchmarks establish performance baselines
- [ ] Regression suite runs in under 60 seconds

## Team Composition

| Role | Responsibility |
|------|----------------|
| Fixer Agent | Pre-existing build issues |
| Contract Test Specialist | Provider contract validation |
| SDET | A2A integration tests |
| Performance Engineer | Benchmarks |
| QA Architect | Regression suite strategy |

## Verification Commands

```bash
# Build
go build ./...

# Test
go test ./... -v

# Benchmarks
go test ./tests/benchmarks/... -bench=. -benchmem

# Race detection
go test ./... -race

# Coverage
go test ./... -cover
```
