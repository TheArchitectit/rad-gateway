# RAD Gateway Seed Data

This package provides comprehensive seed data and fixtures for RAD Gateway development and testing.

## Overview

The seed package contains:

- **generator.go** - Core seed data generation utilities
- **scenarios.go** - Pre-defined realistic data scenarios
- **seeder.go** - Database seeding orchestration
- **fixtures.go** - Unit test fixtures

## Scenarios

### development (Default)
Rich development dataset with:
- 5 workspaces (Acme Corp, Engineering, Platform Team, Research Lab, Customer Success)
- 3-10 users per workspace
- 8 predefined roles
- 19 permissions
- 5 provider configurations (OpenAI, Anthropic, Gemini, Azure)
- 4-7 API keys per workspace
- Realistic usage records

### production-like
Production-like dataset with:
- 15 workspaces
- 100+ users
- 10,000+ usage records
- Realistic traffic patterns

### stress-test
Large-scale dataset for performance testing:
- 100 workspaces
- 1,000 users
- 100,000 usage records

### demo
Polished demo environment:
- Single demo workspace
- 3 demo users with different roles
- 3 active providers
- Beautiful dashboard layout

### minimal
Minimal required data:
- 1 workspace
- 1 admin user
- 1 provider

## Usage

### Command Line Tool

```bash
# Seed with development data (default)
go run cmd/seed/main.go

# Seed with specific scenario
go run cmd/seed/main.go --scenario production-like

# Truncate and re-seed
go run cmd/seed/main.go --scenario demo --truncate

# Dry run to preview changes
go run cmd/seed/main.go --scenario stress-test --dry-run

# Seed specific workspace only
go run cmd/seed/main.go --workspace ws_acme_001

# List all scenarios
go run cmd/seed/main.go --scenarios

# PostgreSQL support
go run cmd/seed/main.go --db-driver postgres --db-dsn "postgres://user:pass@localhost/radgateway"
```

### Programmatic Usage

```go
package main

import (
    "context"
    "radgateway/internal/db"
    "radgateway/internal/db/seeds"
)

func main() {
    // Connect to database
    database, _ := db.New(db.Config{
        Driver: "sqlite",
        DSN:    "radgateway.db",
    })
    defer database.Close()

    // Create seeder
    seeder := seeds.NewSeeder(database, logger)

    // Seed with options
    err := seeder.Seed(context.Background(), seeds.SeedOptions{
        Scenario: "development",
        Truncate: true,
    })
}
```

### Using Fixtures in Tests

```go
package mypackage_test

import (
    "testing"
    "radgateway/internal/db/seeds"
)

func TestSomething(t *testing.T) {
    fixtures := seeds.NewFixtures()

    // Use pre-defined fixtures
    workspace := fixtures.Workspaces["acme"]
    user := fixtures.Users["alice"]
    provider := fixtures.Providers["openai"]

    // Access relationships
    userRoles := fixtures.UserRoles()
    providerTags := fixtures.ProviderTags()
}
```

## Data Structure

### Workspaces
- Multi-tenancy boundaries
- Settings JSON with feature flags
- Tag-based filtering support

### Users
- Email-based authentication
- Password hashes (bcrypt compatible)
- Last login tracking
- Status: active, pending, disabled

### Providers
- Multiple AI provider types (openai, anthropic, gemini, azure)
- Configuration JSON with models, limits
- Health status tracking
- Circuit breaker support

### API Keys
- SHA-256 hashed keys
- Rate limiting
- Model/API restrictions
- Expiration tracking

### Tags
- Hierarchical category:value format
- Workspace-scoped
- Attached to providers and API keys

### Control Rooms
- Tag-filtered operational views
- Dashboard layout JSON
- User access control

### Quotas
- Types: tokens, cost, requests
- Periods: minute, hourly, daily, monthly
- Scopes: workspace, api_key, user
- Warning thresholds

### Usage Records
- Request tracking with trace IDs
- Token and cost tracking
- Error logging
- Provider routing history

## Extending

### Adding a New Scenario

1. Create scenario function in `scenarios.go`:

```go
func CustomScenario() Scenario {
    return Scenario{
        Name:        "custom",
        Description: "Custom scenario description",
        Generator:   generateCustomData,
    }
}

func generateCustomData(g *Generator) *FullSeedData {
    data := &FullSeedData{}
    // Generate your data...
    return data
}
```

2. Register in `AllScenarios()`:

```go
func AllScenarios() []Scenario {
    return []Scenario{
        // ... existing scenarios ...
        CustomScenario(),
    }
}
```

### Adding New Fixtures

1. Add to `Fixtures` struct in `fixtures.go`
2. Initialize in constructor
3. Add helper methods if needed

## Security Notes

- All API keys in seed data use SHA-256 hashes
- Password hashes are placeholder values - use proper bcrypt in production
- Encrypted API keys use placeholder encryption markers
- No real credentials are included

## License

Part of RAD Gateway - See project LICENSE file
