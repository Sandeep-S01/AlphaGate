import { expect, test } from '@playwright/test';

test.describe('Ops reconciliation workflow', () => {
  test('runs reconciliation from Ops and refreshes the result table', async ({ page }) => {
    let runCount = 0;
    const createdRun = {
      id: 'recon-e2e-001',
      status: 'matched',
      mismatches: [],
      created_at: '2026-06-26T16:48:00Z'
    };

    await page.route('**/api/v1/ops/pipeline-runs', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [] })
    }));
    await page.route('**/api/v1/ops/streams', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [] })
    }));
    await page.route('**/api/v1/reconciliation/runs?limit=10', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: runCount > 0 ? [createdRun] : [] })
    }));
    await page.route('**/api/v1/reconciliation/runs', async route => {
      if (route.request().method() !== 'POST') {
        await route.fallback();
        return;
      }

      runCount += 1;
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ data: createdRun })
      });
    });

    await page.goto('/');
    await page.getByRole('button', { name: 'Ops' }).click();

    await expect(page.getByRole('cell', { name: 'No reconciliation runs recorded.' })).toBeVisible();

    await page.getByRole('button', { name: 'Run Reconciliation' }).click();

    await expect(page.getByText('Reconciliation completed: matched.')).toBeVisible();
    await expect(page.getByRole('cell', { name: 'recon-e2e' })).toBeVisible();
    await expect(page.getByText('MATCHED').first()).toBeVisible();
    expect(runCount).toBe(1);
  });

  test('shows critical mismatch severity and details after reconciliation', async ({ page }) => {
    let runCount = 0;
    const criticalRun = {
      id: 'recon-critical-001',
      status: 'mismatch',
      created_at: '2026-06-26T16:50:00Z',
      mismatches: [{
        kind: 'order',
        key: 'paper-client-1',
        internal_value: 'submitted',
        external_value: 'missing',
        severity: 'critical'
      }]
    };

    await page.route('**/api/v1/ops/pipeline-runs', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [] })
    }));
    await page.route('**/api/v1/ops/streams', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [] })
    }));
    await page.route('**/api/v1/reconciliation/runs?limit=10', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: runCount > 0 ? [criticalRun] : [] })
    }));
    await page.route('**/api/v1/reconciliation/runs', async route => {
      if (route.request().method() !== 'POST') {
        await route.fallback();
        return;
      }

      runCount += 1;
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({ data: criticalRun })
      });
    });

    await page.goto('/');
    await page.getByRole('button', { name: 'Ops' }).click();
    await page.getByRole('button', { name: 'Run Reconciliation' }).click();

    await expect(page.getByText('Reconciliation completed with mismatches.')).toBeVisible();
    await expect(page.locator('span').filter({ hasText: /^CRITICAL$/ })).toBeVisible();
    await expect(page.getByText('order: paper-client-1')).toBeVisible();
    await expect(page.getByText('submitted -> missing')).toBeVisible();
    await expect(page.getByText(/Kill switch may be armed/)).toBeVisible();
  });
});
