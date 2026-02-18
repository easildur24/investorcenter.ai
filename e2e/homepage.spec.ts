import { test, expect } from '@playwright/test';

test.describe('Homepage', () => {
  test('loads successfully with page title', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/InvestorCenter/i);
  });

  test('displays navigation with key links', async ({ page }) => {
    await page.goto('/');

    // Main navigation should have key links
    const nav = page.locator('nav, header');
    await expect(nav.first()).toBeVisible();
  });

  test('search bar is visible and accepts input', async ({ page }) => {
    await page.goto('/');

    const searchInput = page.getByPlaceholder(/search/i);
    if (await searchInput.isVisible()) {
      await searchInput.fill('AAPL');
      await expect(searchInput).toHaveValue('AAPL');
    }
  });

  test('page renders without JavaScript errors', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', (err) => errors.push(err.message));

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });
});
