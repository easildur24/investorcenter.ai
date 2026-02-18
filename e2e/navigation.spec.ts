import { test, expect } from '@playwright/test';

test.describe('Navigation', () => {
  test('navigates from homepage to screener', async ({ page }) => {
    await page.goto('/');

    // Find and click screener link
    const screenerLink = page.getByRole('link', {
      name: /screener/i,
    });

    if (await screenerLink.first().isVisible()) {
      await screenerLink.first().click();
      await expect(page).toHaveURL(/screener/);
    }
  });

  test('navigates from homepage to IC Score page', async ({ page }) => {
    await page.goto('/');

    const icScoreLink = page.getByRole('link', {
      name: /ic.?score/i,
    });

    if (await icScoreLink.first().isVisible()) {
      await icScoreLink.first().click();
      await expect(page).toHaveURL(/ic-score/);
    }
  });

  test('back button returns to previous page', async ({ page }) => {
    await page.goto('/');
    await page.goto('/screener');

    await page.goBack();
    await expect(page).toHaveURL('/');
  });

  test('direct URL navigation works for all main routes', async ({
    page,
  }) => {
    const routes = ['/', '/screener', '/ic-score'];

    for (const route of routes) {
      const response = await page.goto(route);
      expect(response?.status()).toBeLessThan(500);
    }
  });
});
