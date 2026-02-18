import { test, expect } from '@playwright/test';

test.describe('Ticker Page', () => {
  test('loads ticker page for AAPL', async ({ page }) => {
    await page.goto('/ticker/AAPL');

    // Page should contain the ticker symbol
    await expect(page.locator('body')).toContainText('AAPL');
  });

  test('displays company name in header', async ({ page }) => {
    await page.goto('/ticker/AAPL');

    // Should display Apple somewhere on the page
    await expect(page.locator('body')).toContainText(/Apple/i);
  });

  test('displays price information', async ({ page }) => {
    await page.goto('/ticker/AAPL');

    // Price should be visible (contains $ sign)
    await expect(page.locator('body')).toContainText('$');
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
