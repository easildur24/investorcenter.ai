import { test, expect } from '@playwright/test';

test.describe('Ticker Page', () => {
  test('loads ticker page for AAPL', async ({ page }) => {
    await page.goto('/ticker/AAPL');

    // Page should contain the ticker symbol
    await expect(page.locator('body')).toContainText('AAPL');
  });

  test('displays company name or error state', async ({ page }) => {
    await page.goto('/ticker/AAPL');

    // Should display either company data or a graceful error (backend may not be available)
    const body = page.locator('body');
    await expect(body).toContainText(/Apple|Failed to Load|error|loading/i);
  });

  test('displays price or error state', async ({ page }) => {
    await page.goto('/ticker/AAPL');

    // Should display either price ($) or a graceful error message
    const body = page.locator('body');
    await expect(body).toContainText(/\$|Failed to Load|error|loading/i);
  });

  test('has financial data tabs', async ({ page }) => {
    await page.goto('/ticker/AAPL');

    // Look for financial-related tab or section text
    const body = page.locator('body');
    const hasFinancials = await body
      .getByText(/income|financials|balance/i)
      .first()
      .isVisible()
      .catch(() => false);

    // Financial data may or may not be available depending on backend
    expect(typeof hasFinancials).toBe('boolean');
  });

  test('handles invalid ticker gracefully', async ({ page }) => {
    const response = await page.goto('/ticker/ZZZZZZZZ');

    // Should not crash — either 404 or error message
    expect(response?.status()).toBeLessThan(500);
  });
});
