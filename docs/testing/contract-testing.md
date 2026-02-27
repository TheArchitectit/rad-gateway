# Contract Testing Guide

**Sprint 7.1**: Consumer-Driven Contract Testing with Pact

## Overview

Contract testing ensures that the frontend (consumer) and backend (provider) can communicate correctly. We use [Pact](https://pact.io/) for consumer-driven contract testing.

### What is Contract Testing?

Contract testing verifies that:
1. The consumer sends requests in the format the provider expects
2. The provider responds in the format the consumer expects
3. Changes don't break the contract between services

### Benefits

- **Faster Feedback**: Catch integration issues before deployment
- **Independent Development**: Teams can work in parallel
- **API Documentation**: Contracts serve as living documentation
- **Safe Changes**: Confidently refactor and upgrade

## Architecture

```
┌─────────────────┐         ┌─────────────────┐
│  Frontend (UI)  │ ─────── │  Backend (API)  │
│    Consumer     │  Pact   │    Provider     │
└─────────────────┘         └─────────────────┘
         │                           │
         ▼                           ▼
┌─────────────────┐         ┌─────────────────┐
│ Contract Tests  │         │ Provider Tests  │
│ Generate .json  │────────▶│ Verify against  │
│   contracts     │         │   contracts     │
└─────────────────┘         └─────────────────┘
         │                           │
         └──────────┬────────────────┘
                    ▼
         ┌─────────────────┐
         │  Pact Broker    │
         │ (Contract Hub)  │
         └─────────────────┘
```

## Consumer Tests (Frontend)

### Running Consumer Tests

```bash
cd web/
npm run test:pact:consumer
```

### Writing Consumer Tests

```typescript
import { PactV3 } from '@pact-foundation/pact';

const provider = new PactV3({
  consumer: 'rad-gateway-admin-ui',
  provider: 'rad-gateway-api',
  port: 8991,
  // ... config
});

describe('Health API', () => {
  it('returns health status', () => {
    provider
      .given('the API is running')
      .uponReceiving('a request for health status')
      .withRequest({
        method: 'GET',
        path: '/health',
      })
      .willRespondWith({
        status: 200,
        body: { status: 'ok', database: 'ok' },
      });

    return provider.executeTest(async (mockServer) => {
      const response = await fetch(`${mockServer.url}/health`);
      expect(response.status).toBe(200);
    });
  });
});
```

### Matchers

Use matchers for flexible assertions:

```typescript
import { MatchersV3 } from '@pact-foundation/pact';

const { string, number, uuid, datetime } = MatchersV3;

body: {
  id: uuid('550e8400-e29b-41d4-a716-446655440000'),
  name: string('OpenAI'),
  status: string('healthy'),
  createdAt: datetime('2024-01-15T10:00:00Z'),
}
```

## Provider Tests (Backend)

### Running Provider Tests

```bash
# Start the API server
go run ./cmd/rad-gateway

# Run provider verification
RUN_CONTRACT_TESTS=true \
PROVIDER_BASE_URL=http://localhost:8080 \
go test -v ./tests/pact/...
```

### Provider States

Provider states setup test data before verification:

```go
// In provider_test.go
func setupProviderState(t *testing.T, state string) {
    switch state {
    case "providers exist":
        seedTestProviders()
    case "API keys exist":
        seedTestAPIKeys()
    case "usage data exists":
        seedTestUsageData()
    }
}
```

### Adding New Provider States

1. Add state name in consumer test:
```typescript
.given('new state description')
```

2. Handle state in provider test:
```go
case "new state description":
    setupNewState()
```

## CI/CD Integration

### GitHub Actions Workflow

The `.github/workflows/contracts.yml` runs:

1. **Consumer Tests**: Generate contracts
2. **Publish Contracts**: Upload to Pact Broker
3. **Provider Verification**: Verify backend against contracts
4. **Can I Deploy**: Check if deployment is safe

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `PACT_BROKER_URL` | Pact Broker URL | No (CI only) |
| `PACT_BROKER_TOKEN` | Authentication token | No (CI only) |
| `RUN_CONTRACT_TESTS` | Enable contract tests | Yes |
| `PROVIDER_BASE_URL` | API server URL | Yes |

### Pact Broker

The Pact Broker provides:
- Contract storage and versioning
- Verification results tracking
- Dependency visualization
- "Can I Deploy" checks

#### Publishing Contracts

```bash
# Consumer publishes contracts
npm run pact:publish

# Provider publishes verification results
# (Done automatically in CI)
```

#### Webhook (Optional)

Trigger provider verification when contracts change:

```json
{
  "consumer": "rad-gateway-admin-ui",
  "provider": "rad-gateway-api",
  "events": [{ "name": "contract_content_changed" }],
  "request": {
    "method": "POST",
    "url": "https://ci.example.com/webhook/provider-verify"
  }
}
```

## Contract File Structure

Generated contracts are stored in `tests/pact/contracts/`:

```json
{
  "consumer": { "name": "rad-gateway-admin-ui" },
  "provider": { "name": "rad-gateway-api" },
  "interactions": [
    {
      "description": "returns health status",
      "providerState": "the API is running",
      "request": {
        "method": "GET",
        "path": "/health"
      },
      "response": {
        "status": 200,
        "body": { "status": "ok" }
      }
    }
  ],
  "metadata": {
    "pactSpecification": { "version": "3.0.0" }
  }
}
```

## Best Practices

### DO
- ✅ Test one interaction per test
- ✅ Use provider states for test data
- ✅ Use matchers for dynamic values
- ✅ Keep contracts focused on integration points
- ✅ Run contract tests in CI/CD
- ✅ Version contracts with consumer commits

### DON'T
- ❌ Test business logic in contracts
- ❌ Use hardcoded IDs without matchers
- ❌ Test error scenarios that aren't handled
- ❌ Modify contracts manually
- ❌ Skip provider verification

## Troubleshooting

### Consumer Test Fails

```bash
# Check mock server logs
cat web/logs/pact-consumer.log
```

### Provider Verification Fails

1. Check API server is running:
   ```bash
   curl http://localhost:8080/health
   ```

2. Verify provider state setup:
   ```bash
   RUN_CONTRACT_TESTS=true go test -v ./tests/pact/...
   ```

3. Check for breaking changes in API

### Contract Not Found

Contracts must be generated before provider verification:

```bash
# 1. Run consumer tests first
cd web && npm run test:pact:consumer

# 2. Then run provider tests
RUN_CONTRACT_TESTS=true go test ./tests/pact/...
```

## Sprint 7.1 Completion Checklist

- [x] Pact framework setup
- [x] Consumer tests for health endpoint
- [x] Consumer tests for providers endpoint
- [x] Provider verification framework
- [x] CI/CD pipeline configuration
- [x] Documentation

## Next Steps

1. **Expand Coverage**: Add contract tests for all API endpoints
2. **Pact Broker**: Deploy and configure for production use
3. **Webhook**: Auto-trigger provider verification on contract changes
4. **Bi-directional**: Add OpenAPI-based contract testing
5. **Can I Deploy**: Enable deployment gating based on contract status

## References

- [Pact Documentation](https://docs.pact.io/)
- [Pact JS](https://github.com/pact-foundation/pact-js)
- [Pact Go](https://github.com/pact-foundation/pact-go)
- [Consumer-Driven Contracts](https://martinfowler.com/articles/consumerDrivenContracts.html)
