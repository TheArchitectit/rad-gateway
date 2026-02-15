# Phase 2 Deliverable: Data Models

## Current Core Models

Defined in `internal/models/models.go`:

- Message exchange:
  - `Message`
  - `ChatCompletionRequest`
  - `ChatCompletionResponse`
  - `ChatChoice`
- Generic response path:
  - `ResponseRequest`
  - `GenericResponse`
- Embeddings path:
  - `EmbeddingsRequest`
  - `EmbeddingsResponse`
  - `Embedding`
- Usage and provider transport:
  - `Usage`
  - `ProviderRequest`
  - `ProviderResult`

## Operational Data Stores

- Usage ledger model: `internal/usage/usage.go` (`Record`, `Sink`, `InMemory`)
- Trace event model: `internal/trace/trace.go` (`Event`, `Store`)
- Route candidate model: `internal/provider/provider.go` (`Candidate`)
- Config-level routing candidates: `internal/config/config.go` (`Candidate`, `Config.ModelRoutes`)

## Model Coverage Outcome

- Data contracts for current compatibility endpoints are established.
- In-memory storage supports bootstrap operations and management read endpoints.
- Persistent schema and migrations remain a later-phase deliverable.
