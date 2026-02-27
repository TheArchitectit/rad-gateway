/**
 * Usage API Contract Tests
 * Sprint 7.1: Consumer-Driven Contract Testing
 */

import { describe, it, expect } from 'vitest';
import { PactV3 } from '@pact-foundation/pact';
import path from 'path';

const provider = new PactV3({
  consumer: 'rad-gateway-admin-ui',
  provider: 'rad-gateway-api',
  port: 8993,
  host: 'localhost',
  dir: path.resolve(process.cwd(), '../tests/pact/contracts'),
  logLevel: 'warn',
});

describe('Usage API Contract', () => {
  describe('GET /v0/admin/usage', () => {
    it('returns usage records', async () => {
      await provider
        .given('usage data exists')
        .uponReceiving('a request for usage records')
        .withRequest({
          method: 'GET',
          path: '/v0/admin/usage',
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
                id: '550e8400-e29b-41d4-a716-446655440010',
                timestamp: '2024-01-15T10:00:00.000Z',
                requestId: 'req-123',
                traceId: 'trace-456',
                apiKeyName: 'Production Key',
                incomingApiType: 'chat',
                incomingModel: 'gpt-4o',
                selectedModel: 'gpt-4o',
                providerId: 'openai',
                responseStatus: 'success',
                durationMs: 250,
                promptTokens: 100,
                completionTokens: 50,
                totalTokens: 150,
                costUsd: 0.0025,
              },
            ],
            summary: {
              totalRequests: 1,
              totalTokens: 150,
              totalCostUsd: 0.0025,
              avgDurationMs: 250,
              errorRate: 0,
            },
            total: 1,
            page: 1,
            pageSize: 25,
            hasMore: false,
          },
        })
        .executeTest(async (mockServer: { url: string }) => {
          const response = await fetch(
            `${mockServer.url}/v0/admin/usage?page=1&pageSize=25&startTime=2024-01-01T00:00:00Z&endTime=2024-01-02T00:00:00Z`,
            {
              headers: { Authorization: 'Bearer test-token' },
            }
          );
          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.data).toBeInstanceOf(Array);
          expect(data.summary).toBeDefined();
          expect(data.summary.totalRequests).toBeGreaterThanOrEqual(0);
        });
    });
  });

  describe('GET /v0/admin/usage/summary', () => {
    it('returns usage summary', async () => {
      await provider
        .given('usage data exists')
        .uponReceiving('a request for usage summary')
        .withRequest({
          method: 'GET',
          path: '/v0/admin/usage/summary',
          headers: {
            Authorization: 'Bearer test-token',
          },
        })
        .willRespondWith({
          status: 200,
          headers: { 'Content-Type': 'application/json' },
          body: {
            totalRequests: 1000,
            totalTokens: 500000,
            totalCostUsd: 1.25,
            avgDurationMs: 300,
            errorRate: 0.02,
          },
        })
        .executeTest(async (mockServer: { url: string }) => {
          const response = await fetch(
            `${mockServer.url}/v0/admin/usage/summary?startTime=2024-01-01T00:00:00Z&endTime=2024-01-02T00:00:00Z`,
            {
              headers: { Authorization: 'Bearer test-token' },
            }
          );
          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.totalRequests).toBeDefined();
          expect(data.totalTokens).toBeDefined();
          expect(data.totalCostUsd).toBeDefined();
        });
    });
  });

  describe('GET /v0/admin/usage/trends', () => {
    it('returns usage trends over time', async () => {
      await provider
        .given('usage data exists')
        .uponReceiving('a request for usage trends')
        .withRequest({
          method: 'GET',
          path: '/v0/admin/usage/trends',
          headers: {
            Authorization: 'Bearer test-token',
          },
        })
        .willRespondWith({
          status: 200,
          headers: { 'Content-Type': 'application/json' },
          body: {
            timeRange: {
              start: '2024-01-01T00:00:00Z',
              end: '2024-01-02T00:00:00Z',
            },
            interval: 'hour',
            points: [
              {
                timestamp: '2024-01-01T00:00:00Z',
                requestCount: 10,
                tokenCount: 5000,
                costUsd: 0.0125,
                avgLatencyMs: 250,
                errorCount: 0,
              },
            ],
          },
        })
        .executeTest(async (mockServer: { url: string }) => {
          const response = await fetch(
            `${mockServer.url}/v0/admin/usage/trends?startTime=2024-01-01T00:00:00Z&endTime=2024-01-02T00:00:00Z&interval=hour`,
            {
              headers: { Authorization: 'Bearer test-token' },
            }
          );
          const data = await response.json();

          expect(response.status).toBe(200);
          expect(data.points).toBeInstanceOf(Array);
          expect(data.interval).toBeDefined();
        });
    });
  });

  describe('POST /v0/admin/usage/export', () => {
    it('creates a usage export', async () => {
      await provider
        .given('usage data exists')
        .uponReceiving('a request to export usage data')
        .withRequest({
          method: 'POST',
          path: '/v0/admin/usage/export',
          headers: {
            Authorization: 'Bearer test-token',
            'Content-Type': 'application/json',
          },
          body: {
            startTime: '2024-01-01T00:00:00Z',
            endTime: '2024-01-02T00:00:00Z',
            format: 'csv',
            includeCost: true,
          },
        })
        .willRespondWith({
          status: 201,
          headers: { 'Content-Type': 'application/json' },
          body: {
            exportId: 'export-123',
            status: 'pending',
            recordCount: 0,
          },
        })
        .executeTest(async (mockServer: { url: string }) => {
          const response = await fetch(`${mockServer.url}/v0/admin/usage/export`, {
            method: 'POST',
            headers: {
              Authorization: 'Bearer test-token',
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              startTime: '2024-01-01T00:00:00Z',
              endTime: '2024-01-02T00:00:00Z',
              format: 'csv',
              includeCost: true,
            }),
          });
          const data = await response.json();

          expect(response.status).toBe(201);
          expect(data.exportId).toBeDefined();
          expect(data.status).toBeDefined();
        });
    });
  });
});
