# Testing Improvement Sprints

**Date**: 2026-03-01
**Goal**: Address all test failures and establish reliable CI/CD

---

## Overview

| Sprint | Focus | Duration | Priority |
|--------|-------|----------|----------|
| **Sprint A** | Critical Fixes | 1 day | 🔴 Critical |
| **Sprint B** | Test Infrastructure | 2 days | 🟠 High |
| **Sprint C** | CI/CD Pipeline | 2 days | 🟠 High |
| **Sprint D** | Integration Tests | 3 days | 🟡 Medium |
| **Sprint E** | Coverage & Polish | 2 days | 🟢 Low |

**Total**: 10 days

---

## Sprint A: Critical Fixes (Day 1)

**Goal**: Fix immediate blockers preventing tests from running

### Task A1: Fix Web UI Dependencies

**File**: `web/package.json`

```bash
cd web
npm install --save-dev @radix-ui/react-slot @testing-library/user-event
npm install  # reinstall all dependencies
```

**Verify**:
```bash
npm test -- src/components/ui/__tests__/button.test.tsx
npm test -- src/components/ui/__tests__/dialog.test.tsx
```

**Expected**: Tests should run without import errors

---

### Task A2: Fix Syntax Error in ProtectedRoute Test

**File**: `web/src/components/auth/ProtectedRoute.test.tsx:263`

**Problem**:
```tsx
// Line 263
{ resource: 'providers', action: 'delete' },
]]}>  // ❌ Extra ] before }>
```

**Fix**:
```tsx
{ resource: 'providers', action: 'delete' },
]}>  // ✅ Remove extra ]
```

**Verify**:
```bash
npm test -- src/components/auth/ProtectedRoute.test.tsx
```

---

### Task A3: Fix Cedar Auth Build

**File**: `internal/auth/cedar/pdp_test.go`

**Issues**:
1. Line 8: Unused import `"github.com/stretchr/testify/require"`
2. Line 15: Unused variable `ctx`

**Fix**:
```go
// Remove unused import
// import "github.com/stretchr/testify/require"  // DELETE

// Either remove ctx or use it
ctx := context.Background()  // Use it or remove
```

**Verify**:
```bash
go build ./internal/auth/cedar/...
go test -short ./internal/auth/cedar/...
```

---

### Task A4: Fix A2A JSON Type Mismatch

**File**: `internal/a2a/task_manager_test.go:133`

**Problem**: `[]uint8` vs `json.RawMessage` type mismatch

**Fix**:
```go
// Change assertion to compare as json.RawMessage
require.Equal(t, json.RawMessage(expected), actual)
```

**Verify**:
```bash
go test -v ./internal/a2a/... -run TestTaskManager_CreateTask
```

---

### Sprint A Deliverables
- [ ] All Web UI component tests run without import errors
- [ ] Cedar auth package builds successfully
- [ ] A2A tests pass

---

## Sprint B: Test Infrastructure (Days 2-3)

**Goal**: Separate unit tests from integration tests

### Task B1: Create Test Categories

**Create file**: `internal/testutil/categories.go`

```go
//go:build unit
// +build unit

package testutil

// Unit tests - no external dependencies
// Run: go test -tags=unit ./...
```

```go
//go:build integration
// +build integration

package testutil

// Integration tests - requires external services
// Run: go test -tags=integration ./...
```

---

### Task B2: Mock Redis for Cache Tests

**File**: `internal/cache/mock_redis_test.go` (new)

```go
package cache

import (
    "testing"
    "github.com/stretchr/testify/mock"
)

type MockRedisClient struct {
    mock.Mock
}

func (m *MockRedisClient) Get(key string) (string, error) {
    args := m.Called(key)
    return args.String(0), args.Error(1)
}

// Implement other methods...
```

**Update**: `internal/cache/model_card_cache_typed_test.go`

Add build tag:
```go
//go:build integration
// +build integration
```

**Create unit test version**:
`internal/cache/model_card_cache_unit_test.go`

```go
//go:build unit
// +build unit

package cache

func TestTypedModelCardCache_GetAndSet_Unit(t *testing.T) {
    // Use in-memory mock
    mockClient := NewMockRedisClient()
    cache := NewTypedModelCardCache(mockClient, time.Minute)
    // Test...
}
```

---

### Task B3: Mock Database for Unit Tests

**File**: `internal/db/testutil/db_mock.go` (new)

```go
package testutil

import (
    "database/sql"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
)

func NewMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, error) {
    return sqlmock.New()
}
```

**Update**: `internal/db/optimization_test.go`

Mark as integration test:
```go
//go:build integration
// +build integration

package db
```

**Create**: `internal/db/optimization_unit_test.go`

```go
//go:build unit
// +build unit

package db

import (
    "testing"
    "radgateway/internal/db/testutil"
)

func TestQueryBuilder_Unit(t *testing.T) {
    db, mock, err := testutil.NewMockDB(t)
    require.NoError(t, err)
    defer db.Close()

    // Test query building with mocks
}
```

---

### Task B4: Update Makefile

**File**: `Makefile`

```makefile
.PHONY: test test-unit test-integration test-all

# Run only unit tests (fast, no external deps)
test-unit:
	go test -tags=unit -short ./...

# Run integration tests (requires Redis, DB)
test-integration:
	go test -tags=integration ./...

# Run all tests
test-all:
	go test ./...

# Default: unit tests only
test: test-unit
```

---

### Sprint B Deliverables
- [ ] Build tags implemented (`//go:build unit` vs `//go:build integration`)
- [ ] Mock implementations for Redis
- [ ] Mock implementations for Database
- [ ] Makefile targets for different test categories
- [ ] Documentation on running tests

---

## Sprint C: CI/CD Pipeline (Days 4-5)

**Goal**: Automated testing with GitHub Actions

### Task C1: Create GitHub Actions Workflow

**File**: `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  backend-unit:
    name: Backend Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

      - name: Download dependencies
        run: go mod download

      - name: Run unit tests
        run: make test-unit

      - name: Generate coverage
        run: go test -tags=unit -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out
          flags: backend-unit

  backend-integration:
    name: Backend Integration Tests
    runs-on: ubuntu-latest
    services:
      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: radgateway_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run integration tests
        run: make test-integration

  frontend:
    name: Frontend Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: web/package-lock.json

      - name: Install dependencies
        working-directory: web
        run: npm ci

      - name: Run lint
        working-directory: web
        run: npm run lint

      - name: Run type check
        working-directory: web
        run: npm run typecheck

      - name: Run tests
        working-directory: web
        run: npm test

  build:
    name: Build Check
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build backend
        run: go build -o rad-gateway ./cmd/rad-gateway

      - name: Build frontend
        working-directory: web
        run: npm ci && npm run build
```

---

### Task C2: Add Pre-commit Hooks

**File**: `.pre-commit-config.yaml`

```yaml
repos:
  - repo: local
    hooks:
      - id: go-fmt
        name: Go Format
        entry: gofmt -w
        language: system
        files: \.go$

      - id: go-vet
        name: Go Vet
        entry: go vet ./...
        language: system
        files: \.go$
        pass_filenames: false

      - id: go-test-unit
        name: Go Unit Tests
        entry: make test-unit
        language: system
        files: \.go$
        pass_filenames: false

      - id: web-lint
        name: Web Lint
        entry: bash -c 'cd web && npm run lint'
        language: system
        files: ^web/.*\.(ts|tsx|js|jsx)$
```

**Install**:
```bash
pip install pre-commit
pre-commit install
```

---

### Task C3: Test Documentation

**File**: `docs/testing/TESTING.md`

```markdown
# Testing Guide

## Quick Start

```bash
# Run only unit tests (fast)
make test-unit

# Run integration tests (requires Redis + DB)
make test-integration

# Run all tests
make test-all
```

## Test Categories

### Unit Tests
No external dependencies. Fast and reliable.
```bash
go test -tags=unit ./...
```

### Integration Tests
Require external services:
- Redis (for cache tests)
- PostgreSQL (for DB tests)

Start services:
```bash
docker-compose -f docker-compose.test.yml up -d
```

Then run:
```bash
go test -tags=integration ./...
```

## Backend Tests

### Provider Adapters
```bash
go test -v ./internal/provider/...
```

### Core Components
```bash
go test -v ./internal/core/...
go test -v ./internal/routing/...
```

## Frontend Tests

```bash
cd web
npm test
```

## Writing Tests

### Unit Test Template
```go
//go:build unit
// +build unit

package mypackage

import "testing"

func TestFeature_Unit(t *testing.T) {
    // Use mocks, no external deps
}
```

### Integration Test Template
```go
//go:build integration
// +build integration

package mypackage

func TestFeature_Integration(t *testing.T) {
    // Requires external services
    // Skip if not available
    if os.Getenv("REDIS_ADDR") == "" {
        t.Skip("Redis not available")
    }
}
```
```

---

### Sprint C Deliverables
- [ ] GitHub Actions workflow file
- [ ] Pre-commit hooks configured
- [ ] Test documentation
- [ ] CI badge in README

---

## Sprint D: Integration Tests (Days 6-8)

**Goal**: Comprehensive integration testing

### Task D1: Contract Tests

**File**: `web/src/__tests__/pact/apikeys.contract.test.ts` (update)

```typescript
import { pactWith } from 'jest-pact';
import { apiClient } from '@/api/client';

// Skip if no backend running
const describeContract = process.env.CI ? describe : describe.skip;

describeContract('API Keys API', () => {
  // Contract tests...
});
```

---

### Task D2: API Integration Tests

**File**: `internal/api/integration_test.go` (new)

```go
//go:build integration
// +build integration

package api

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthEndpoint_Integration(t *testing.T) {
    // Start test server
    handler := setupTestServer()
    server := httptest.NewServer(handler)
    defer server.Close()

    // Make request
    resp, err := http.Get(server.URL + "/health")
    require.NoError(t, err)
    require.Equal(t, 200, resp.StatusCode)
}
```

---

### Task D3: E2E Tests with Playwright

**File**: `web/e2e/login.spec.ts` (update)

```typescript
import { test, expect } from '@playwright/test';

test.describe('Authentication', () => {
  test('user can login', async ({ page }) => {
    await page.goto('/login');

    await page.fill('[name="username"]', 'admin');
    await page.fill('[name="password"]', 'admin');
    await page.click('button[type="submit"]');

    await expect(page).toHaveURL('/');
    await expect(page.locator('text=Dashboard')).toBeVisible();
  });
});
```

**Playwright Config** (`web/playwright.config.ts`):

```typescript
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:8090',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
```

---

### Task D4: Test Data Fixtures

**File**: `internal/db/fixtures/users.yml`

```yaml
users:
  - id: test-user-1
    email: admin@rad.local
    role: admin

  - id: test-user-2
    email: developer@rad.local
    role: developer
```

**File**: `internal/db/fixtures/loader.go`

```go
package fixtures

import (
    "database/sql"
    "gopkg.in/yaml.v3"
)

type Loader struct {
    db *sql.DB
}

func (l *Loader) Load(path string) error {
    // Load YAML fixtures
}
```

---

### Sprint D Deliverables
- [ ] Contract tests with Pact
- [ ] API integration tests
- [ ] Playwright E2E tests
- [ ] Test data fixtures
- [ ] Docker Compose for test environment

---

## Sprint E: Coverage & Polish (Days 9-10)

**Goal**: Improve test coverage and documentation

### Task E1: Coverage Reporting

**File**: `.github/workflows/coverage.yml`

```yaml
name: Coverage

on:
  push:
    branches: [main]

jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run tests with coverage
        run: |
          go test -tags=unit -coverprofile=coverage.out ./...
          go tool cover -html=coverage.out -o coverage.html

      - name: Upload to Codecov
        uses: codecov/codecov-action@v4
        with:
          file: ./coverage.out
```

---

### Task E2: Coverage Badges

**Update**: `README.md`

```markdown
[![CI](https://github.com/TheArchitectit/rad-gateway/actions/workflows/ci.yml/badge.svg)](https://github.com/TheArchitectit/rad-gateway/actions)
[![Coverage](https://codecov.io/gh/TheArchitectit/rad-gateway/branch/main/graph/badge.svg)](https://codecov.io/gh/TheArchitectit/rad-gateway)
[![Go Report Card](https://goreportcard.com/badge/github.com/TheArchitectit/rad-gateway)](https://goreportcard.com/report/github.com/TheArchitectit/rad-gateway)
```

---

### Task E3: Test Quality Metrics

**File**: `docs/testing/QUALITY.md`

```markdown
# Test Quality Metrics

## Current Status

| Package | Coverage | Status |
|---------|----------|--------|
| Provider Adapters | 85% | ✅ |
| Core Components | 72% | ✅ |
| Web UI | 45% | ⚠️ |

## Goals

- Backend: 80% coverage
- Frontend: 70% coverage
- E2E: Critical paths covered
```

---

### Sprint E Deliverables
- [ ] Coverage reporting in CI
- [ ] Coverage badges in README
- [ ] Quality metrics documentation
- [ ] Test runbook for developers

---

## Timeline

| Day | Sprint | Tasks |
|-----|--------|-------|
| 1 | A | Fix dependencies, syntax, build errors |
| 2-3 | B | Mock infrastructure, build tags |
| 4-5 | C | CI/CD pipeline, pre-commit hooks |
| 6-8 | D | Integration tests, E2E tests |
| 9-10 | E | Coverage reporting, documentation |

---

## Success Criteria

- [ ] All unit tests pass without external services
- [ ] CI pipeline runs in < 10 minutes
- [ ] Coverage badges visible in README
- [ ] New code requires tests to pass
- [ ] Documentation explains test categories

---

**Start Date**: Upon approval
**Estimated Completion**: 10 days
