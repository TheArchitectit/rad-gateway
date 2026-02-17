# Changelog

All notable changes to RAD Gateway (Brass Relay) will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v0.4.0] - 2026-02-17

### Phase 4: The Data Wardens

#### Database Optimization
- Added database indexes (`internal/db/indexes.sql`)
- Created query optimization utilities (`internal/db/optimization.go`)
- Added query benchmarks (`internal/db/optimization_test.go`)
- Implemented migration runner with versioning (`internal/db/migrator.go`)

#### Seed Data
- Created seed data generator (`internal/db/seeds/generator.go`)
- Added test scenarios (`internal/db/seeds/scenarios.go`)
- Created test fixtures (`internal/db/seeds/fixtures.go`)
- Added seeder implementation (`internal/db/seeds/seeder.go`)
- Created seed CLI tool (`cmd/seed/main.go`)

#### Performance Monitoring
- Added database metrics collection (`internal/db/metrics.go`)
- Implemented slow query detection (`internal/db/slowquery.go`)
- Created migration CLI tool (`cmd/migrate/main.go`)

## [v0.3.0] - 2026-02-17

### Phase 3: The Backend Core

#### Database Layer
- Implemented database interface pattern
- Added SQLite driver with connection pooling
- Added PostgreSQL driver with UPSERT support
- Created migration system with 9 migration files
- Added repository layer for all entities

#### RBAC System
- Implemented role definitions (Admin/Developer/Viewer)
- Added permission system with bit flags
- Created RBAC HTTP middleware
- Added project isolation logic

#### Cost Tracking Service
- Created cost calculation engine
- Implemented usage aggregation with time windows
- Added background worker for batch processing
- Created cost service API
- Added comprehensive tests

#### Admin API Endpoints
- Project CRUD with bulk operations
- API key management (create, revoke, rotate)
- Usage query endpoints with filtering
- Cost tracking endpoints with forecasting
- Quota management endpoints
- Provider management endpoints

#### Frontend Skeleton
- React + Zustand + TypeScript setup
- Created stores (auth, ui, workspace)
- Added custom hooks for data fetching
- Implemented API client with error handling
- Created TypeScript types

### Security
- Fixed critical auth bypass vulnerability (removed `/v0/management/` from auth bypass)

## [v0.2.0] - 2026-02-17

### Phase 2: The UI/UX Core

#### Frontend Feature Specification
- Defined 15 features (FL-001 to FL-015)
- Super power features: Real-time Control Rooms, Visual Provider Mesh, ML Cost Forecasting
- Drag-and-drop dashboard builder
- Trace timeline explorer

#### Component Architecture
- Atomic design structure (atoms → molecules → organisms → templates → pages)
- 18 routes organized into 5 sections
- Zustand + TanStack Query state management
- Error boundaries at all levels

#### State Management Implementation
- Zustand stores (authStore, uiStore, workspaceStore)
- Custom hooks (useAuth, useWorkspace, useProviders, useApiKeys, useUsage)
- API client with fetch and error handling

## [v0.1.0] - 2026-02-17

### Phase 1: The Architects

#### Requirements & Schema Design
- Defined functional requirements (FR-001 to FR-020)
- Non-functional requirements (NFR-001 to NFR-010)
- Database schema design for SQLite/PostgreSQL
- API architecture specifications

#### Architecture Decisions
- Interface-based database pattern
- 3-role RBAC system (Admin/Developer/Viewer)
- Project-level isolation
- Cost tracking (offline calculation initially)
- Quota management (basic rate limiting)

#### Documentation
- Feature parity analysis (AxonHub, Plexus comparison)
- Architecture synthesis report
- Security architecture review
- Frontend specifications
- Database schema design

### Features
- Multi-provider support (OpenAI, Anthropic, Gemini)
- OpenAI-compatible API endpoints
- SSE streaming with backpressure handling
- Circuit breaker pattern
- Load balancing strategies
- Health checking framework
- Infisical secrets integration
- Structured logging with slog

### Security
- API key authentication (Bearer, x-api-key, x-goog-api-key)
- Admin endpoints (authenticated after security fix)

[Unreleased]: https://github.com/rad-gateway/rad-gateway/compare/v0.4.0...HEAD
[v0.4.0]: https://github.com/rad-gateway/rad-gateway/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/rad-gateway/rad-gateway/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/rad-gateway/rad-gateway/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/rad-gateway/rad-gateway/releases/tag/v0.1.0
