/**
 * Health Endpoint Contract Tests
 * Sprint 7.1: Consumer-Driven Contract Testing
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import pact from '@pact-foundation/pact';
import path from 'path';

const { PactV3 } = pact;

// Create a new Pact instance for each test file
let provider: any;

describe('Health API Contract', () => {
  beforeAll(async () => {
    provider = new PactV3({
      consumer: 'rad-gateway-admin-ui',
      provider: 'rad-gateway-api',
      port: 8991,
      host: 'localhost',
      dir: path.resolve(process.cwd(), '../tests/pact/contracts'),
      logLevel: 'warn',
    });
  });

  describe('GET /health', () => {
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
          headers: { 'Content-Type': 'application/json' },
          body: {
            status: 'ok',
            database: 'ok',
            driver: 'sqlite',
          },
        });

      return provider.executeTest(async (mockServer: any) => {
        const response = await fetch(`${mockServer.url}/health`);
        const data = await response.json();

        expect(response.status).toBe(200);
        expect(data.status).toBe('ok');
        expect(data.database).toBeDefined();
        expect(data.driver).toBeDefined();
      });
    });

    it('returns degraded status when database is down', () => {
      provider
        .given('the database is unavailable')
        .uponReceiving('a request for health status when degraded')
        .withRequest({
          method: 'GET',
          path: '/health',
        })
        .willRespondWith({
          status: 503,
          headers: { 'Content-Type': 'application/json' },
          body: {
            status: 'ok',
            database: 'degraded',
            driver: 'sqlite',
          },
        });

      return provider.executeTest(async (mockServer: any) => {
        const response = await fetch(`${mockServer.url}/health`);
        expect(response.status).toBe(503);
      });
    });
  });
});
