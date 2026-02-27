/**
 * API Keys Contract Tests
 * Sprint 7.1: Consumer-Driven Contract Testing
 */

import { describe, it, expect } from 'vitest';
import { PactV3 } from '@pact-foundation/pact';
import path from 'path';

const provider = new PactV3({
  consumer: 'rad-gateway-admin-ui',
  provider: 'rad-gateway-api',
  port: 8992,
  host: 'localhost',
  dir: path.resolve(process.cwd(), '../tests/pact/contracts'),
  logLevel: 'warn',
});

describe('API Keys API Contract', () => {
  describe('GET /v0/admin/api-keys', () => {
    it('returns list of API keys', async () => {
      await provider
        .given('API keys exist')
        .uponReceiving('a request for all API keys')
        .withRequest({
          method: 'GET',
          path: '/v0/admin/api-keys',
          headers: {
            Authorization: 'Bearer test-token',
          },
        })
        .willRespondWith({
          status: 200,
          headers: { 'Content-Type': 'application/json' },
          body: {
            data: [
              {
                id: '550e8400-e29b-41d4-a716-446655440001',
                name: 'Production Key',
                keyPreview: 'sk-...abcd',
                permissions: ['read', 'write'],
                rateLimit: 1000,
                usageLimit: 10000,
                currentUsage: 5000,
                createdAt: '2024-01-15T10:00:00.000Z',
                expiresAt: '2025-01-15T10:00:00.000Z',
                lastUsedAt: '2024-01-15T12:00:00.000Z',
                status: 'active',
                workspaceId: '550e8400-e29b-41d4-a716-446655440002',
              },
            ],
            meta: {
              totalCount: 1,
              hasMore: false,
            },
          },
        })
        .executeTest(async (mockServer: { url: string }) => {
          const response = await fetch(`${mockServer.url}/v0/admin/api-keys`, {
            headers: { Authorization: 'Bearer test-token' },
          });
          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.data).toBeInstanceOf(Array);
          expect(data.data[0]).toHaveProperty('id');
          expect(data.data[0]).toHaveProperty('name');
          expect(data.data[0]).toHaveProperty('keyPreview');
        });
    });
  });

  describe('POST /v0/admin/api-keys', () => {
    it('creates a new API key', async () => {
      await provider
        .given('workspace exists')
        .uponReceiving('a request to create an API key')
        .withRequest({
          method: 'POST',
          path: '/v0/admin/api-keys',
          headers: {
            Authorization: 'Bearer test-token',
            'Content-Type': 'application/json',
          },
          body: {
            name: 'New API Key',
            permissions: ['read'],
            rateLimit: 100,
          },
        })
        .willRespondWith({
          status: 201,
          headers: { 'Content-Type': 'application/json' },
          body: {
            id: '550e8400-e29b-41d4-a716-446655440003',
            name: 'New API Key',
            keyPreview: 'sk-...efgh',
            permissions: ['read'],
            rateLimit: 100,
            status: 'active',
            createdAt: '2024-01-15T10:00:00.000Z',
            workspaceId: '550e8400-e29b-41d4-a716-446655440002',
          },
        })
        .executeTest(async (mockServer: { url: string }) => {
          const response = await fetch(`${mockServer.url}/v0/admin/api-keys`, {
            method: 'POST',
            headers: {
              Authorization: 'Bearer test-token',
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              name: 'New API Key',
              permissions: ['read'],
              rateLimit: 100,
            }),
          });
          const data = await response.json();

          expect(response.status).toBe(201);
          expect(data.id).toBeDefined();
          expect(data.name).toBe('New API Key');
        });
    });
  });

  describe('DELETE /v0/admin/api-keys/:id', () => {
    it('revokes an API key', async () => {
      await provider
        .given('API key exists')
        .uponReceiving('a request to revoke an API key')
        .withRequest({
          method: 'DELETE',
          path: '/v0/admin/api-keys/550e8400-e29b-41d4-a716-446655440001',
          headers: {
            Authorization: 'Bearer test-token',
          },
        })
        .willRespondWith({
          status: 204,
        })
        .executeTest(async (mockServer: { url: string }) => {
          const response = await fetch(
            `${mockServer.url}/v0/admin/api-keys/550e8400-e29b-41d4-a716-446655440001`,
            {
              method: 'DELETE',
              headers: { Authorization: 'Bearer test-token' },
            }
          );

          expect(response.status).toBe(204);
        });
    });
  });
});
