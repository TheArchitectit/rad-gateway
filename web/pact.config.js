/**
 * Pact Contract Testing Configuration
 * Sprint 7.1: Contract Testing Setup
 */

const path = require('path');

module.exports = {
  // Consumer configuration
  consumer: 'rad-gateway-admin-ui',
  provider: 'rad-gateway-api',

  // Pact broker settings (for CI/CD)
  pactBrokerUrl: process.env.PACT_BROKER_URL || 'http://localhost:9292',
  pactBrokerToken: process.env.PACT_BROKER_TOKEN,

  // Contract file locations
  contractsDir: path.resolve(__dirname, '../tests/pact/contracts'),

  // Consumer test settings
  consumerTests: {
    port: 8991,
    host: 'localhost',
    dir: path.resolve(__dirname, '../tests/pact/contracts'),
    log: path.resolve(__dirname, 'logs/pact-consumer.log'),
    logLevel: 'INFO',
  },

  // Provider verification settings
  providerTests: {
    providerBaseUrl: process.env.PROVIDER_BASE_URL || 'http://localhost:8080',
    pactUrls: [
      path.resolve(__dirname, '../tests/pact/contracts/*.json'),
    ],
    publishVerificationResult: process.env.CI === 'true',
    providerVersion: process.env.GIT_COMMIT || 'local',
    providerVersionBranch: process.env.GIT_BRANCH || 'main',
  },

  // API endpoints to test
  endpoints: {
    // Health
    health: '/health',

    // Providers
    providers: '/v0/admin/providers',
    providerHealth: '/v0/admin/providers/health',

    // API Keys
    apiKeys: '/v0/admin/api-keys',

    // Usage
    usage: '/v0/admin/usage',
    usageSummary: '/v0/admin/usage/summary',
    usageTrends: '/v0/admin/usage/trends',

    // Projects/Workspaces
    projects: '/v0/admin/workspaces',

    // Model Routes
    modelRoutes: '/v0/admin/model-routes',

    // Configuration
    config: '/v0/admin/config',
  },
};
