import { test, expect } from '@playwright/test';

test.describe('RAD Gateway UI Smoke Tests', () => {
  test('Dashboard loads successfully', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('Dashboard');
    await expect(page.locator('text=Overview of your AI gateway')).toBeVisible();
  });

  test('Control Rooms page loads', async ({ page }) => {
    await page.goto('/control-rooms');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('Control Rooms');
  });

  test('Providers page loads', async ({ page }) => {
    await page.goto('/providers');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('Providers');
  });

  test('API Keys page loads', async ({ page }) => {
    await page.goto('/api-keys');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('API Keys');
  });

  test('Projects page loads', async ({ page }) => {
    await page.goto('/projects');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('Projects');
  });

  test('Usage page loads', async ({ page }) => {
    await page.goto('/usage');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('Usage');
  });

  test('A2A page loads', async ({ page }) => {
    await page.goto('/a2a');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('A2A');
  });

  test('OAuth page loads', async ({ page }) => {
    await page.goto('/oauth');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('OAuth');
  });

  test('MCP page loads', async ({ page }) => {
    await page.goto('/mcp');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('MCP');
  });

  test('Reports page loads', async ({ page }) => {
    await page.goto('/reports');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('Reports');
  });

  test('Login page loads', async ({ page }) => {
    await page.goto('/login');
    await expect(page).toHaveTitle(/RAD Gateway/);
    await expect(page.locator('h1')).toContainText('Login');
    await expect(page.locator('button:has-text("Sign In")')).toBeVisible();
  });

  test('Navigation sidebar is visible', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('nav')).toBeVisible();
    await expect(page.locator('text=Dashboard')).toBeVisible();
    await expect(page.locator('text=Control Rooms')).toBeVisible();
    await expect(page.locator('text=Providers')).toBeVisible();
    await expect(page.locator('text=API Keys')).toBeVisible();
    await expect(page.locator('text=Projects')).toBeVisible();
  });

  test('Dashboard stats cards are visible', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('text=Active Providers')).toBeVisible();
    await expect(page.locator('text=API Calls Today')).toBeVisible();
    await expect(page.locator('text=Active API Keys')).toBeVisible();
    await expect(page.locator('text=Cost Today')).toBeVisible();
  });
});
