/**
 * Providers API Contract Tests
 * Sprint 7.1: Consumer-Driven Contract Testing
 */

import { describe, it, expect } from 'vitest';
import { provider, jsonHeaders } from './setup';

describe('Providers API Contract', () => {
  describe('GET /v0/admin/providers', () => {
    it('returns list of providers', async () => {
      await provider
        .addInteraction({
          state: 'providers exist',
          uponReceiving: 'a request for all providers',
          withRequest: {
            method: 'GET',
            path: '/v0/admin/providers',
            headers: {
              Authorization: 'Bearer test-token',
            },
          },
          willRespondWith: {
            status: 200,
            headers: jsonHeaders,
            body: {
              providers: [
                {
                  id: '550e8400-e29b-41d4-a716-446655440000',
                  name: 'openai',
                  displayName: 'OpenAI',
                  status: 'healthy',
                  circuitBreaker: 'closed',
                  lastCheck: '2024-01-15T10:00:00.000Z',
                  requestCount24h: 1000,
                  errorRate24h: 0.01,
                  latencyMs: 150,
                  models: ['gpt-4o', 'gpt-4o-mini'],
                },
              ],
            },
          },
        })
        .executeTest(async (mockServer) => {
          const response = await fetch(`${mockServer.url}/v0/admin/providers`, {
            headers: { Authorization: 'Bearer test-token' },
          });
          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.providers).toBeInstanceOf(Array);
          expect(data.providers[0]).toHaveProperty('id');
          expect(data.providers[0]).toHaveProperty('name');
          expect(data.providers[0]).toHaveProperty('status');
        });
    });

    it('returns 401 when unauthorized', async () => {
      await provider
        .addInteraction({
          state: 'no authentication provided',
          uponReceiving: 'a request for providers without auth',
          withRequest: {
            method: 'GET',
            path: '/v0/admin/providers',
          },
          willRespondWith: {
            status: 401,
            headers: jsonHeaders,
            body: {
              error: {
                message: 'Unauthorized',
                code: 'UNAUTHORIZED',
              },
            },
          },
        })
        .executeTest(async (mockServer) => {
          const response = await fetch(`${mockServer.url}/v0/admin/providers`);
          expect(response.status).toBe(401);
        });
    });
  });

  describe('GET /v0/admin/providers/health', () => {
    it('returns providers health status', async () => {
      await provider
        .addInteraction({
          state: 'providers exist',
          uponReceiving: 'a request for providers health',
          withRequest: {
            method: 'GET',
            path: '/v0/admin/providers/health',
            headers: {
              Authorization: 'Bearer test-token',
            },
          },
          willRespondWith: {
            status: 200,
            headers: jsonHeaders,
            body: {
              providers: [
                {
                  name: 'openai',
                  status: 'healthy',
                  latencyMs: 150,
                  checkedAt: '2024-01-15T10:00:00.000Z',
                },
              ],
            },
          },
        })
        .executeTest(async (mockServer) => {
          const response = await fetch(`${mockServer.url}/v0/admin/providers/health`, {
            headers: { Authorization: 'Bearer test-token' },
          });
          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.providers).toBeInstanceOf(Array);
          expect(data.providers[0]).toHaveProperty('status');
          expect(data.providers[0]).toHaveProperty('latencyMs');
        });
    });
  });
});
