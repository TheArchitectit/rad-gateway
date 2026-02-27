/**
 * Pact Consumer Test Setup
 * Sprint 7.1: Contract Testing - Consumer Setup
 */

import { PactV4 } from '@pact-foundation/pact';
import path from 'path';

export const provider = new PactV4({
  consumer: 'rad-gateway-admin-ui',
  provider: 'rad-gateway-api',
  port: 8991,
  host: 'localhost',
  dir: path.resolve(process.cwd(), '../tests/pact/contracts'),
  logLevel: 'warn',
});

export const API_BASE = '/v0/admin';

// Common response headers
export const jsonHeaders = {
  'Content-Type': 'application/json',
};
