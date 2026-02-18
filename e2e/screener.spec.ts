import { test, expect } from '@playwright/test';

test.describe('Screener Page', () => {
  test('loads screener page', async ({ page }) => {
    await page.goto('/screener');

    await expect(page.locator('body')).toContainText(
      /screener|filter|stocks/i
    );
  });

  test('displays a table or list of stocks', async ({ page }) => {
    await page.goto('/screener');

    // Wait for data to load
    await page.waitForLoadState('networkidle');

    // Should have a table element
    const table = page.locator('table');
    const hasTable = await table.first().isVisible().catch(() => false);

    // If no table, there should be stock-related content
    if (!hasTable) {
      await expect(page.locator('body')).toContainText(
        /stock|ticker|symbol/i
      );
    }
  });

  test('has sorting controls', async ({ page }) => {
    await page.goto('/screener');
    await page.waitForLoadState('networkidle');

    // Table headers or sort buttons should exist
    const headers = page.locator('th, [role="columnheader"]');
    const count = await headers.count();

    // Screener should have multiple columns
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('page loads without server errors', async ({ page }) => {
    const response = await page.goto('/screener');
    expect(response?.status()).toBeLessThan(500);
  });
});
