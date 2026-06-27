import { expect, test } from '@playwright/test';

test.describe('Paper order lifecycle visibility', () => {
  test('shows partial and failed order lifecycle details in the orders tab', async ({ page }) => {
    await page.route('**/api/v1/market/candles?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [] })
    }));
    await page.route('**/api/v1/strategy/settings', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { strategy_name: 'multi-factor-momentum' } })
    }));
    await page.route('**/api/v1/paper/orders?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          {
            ID: 'order-partial-1',
            ClientOrderID: 'paper-client-1',
            ExchangeOrderID: 'paper-exchange-1',
            StrategyName: 'multi-factor-momentum',
            Symbol: 'BTCUSDT',
            Side: 'buy',
            Quantity: 0.08,
            RequestedQuantity: 0.10,
            FilledQuantity: 0.04,
            Price: 65000,
            AverageFillPrice: 65010,
            Status: 'partially_filled',
            SubmittedAt: '2026-06-26T10:01:00Z',
            CreatedAt: '2026-06-26T10:00:00Z',
            UpdatedAt: '2026-06-26T10:02:00Z'
          },
          {
            ID: 'order-failed-1',
            ClientOrderID: 'paper-client-2',
            ExchangeOrderID: '',
            StrategyName: 'multi-factor-momentum',
            Symbol: 'BTCUSDT',
            Side: 'sell',
            Quantity: 0,
            RequestedQuantity: 0.12,
            FilledQuantity: 0,
            Price: 65100,
            AverageFillPrice: 0,
            Status: 'failed',
            FailureReason: 'exchange timeout after retry policy',
            SubmittedAt: '2026-06-26T10:03:00Z',
            CreatedAt: '2026-06-26T10:03:00Z',
            UpdatedAt: '2026-06-26T10:04:00Z'
          }
        ]
      })
    }));
    await page.route('**/api/v1/paper/trades?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/signals?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/audit/events?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/ops/pipeline-runs?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/ops/streams', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/safety/status', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { kill_switch_active: false } }) }));
    await page.route('**/api/v1/strategy/lifecycle?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/reconciliation/runs?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/risk/settings', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { enabled: true } }) }));
    await page.route('**/api/v1/risk-decisions?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));

    await page.goto('/');

    await expect(page.getByRole('columnheader', { name: 'Client Order' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Requested/Filled' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Avg Fill' })).toBeVisible();
    await expect(page.getByText('paper-client-1')).toBeVisible();
    await expect(page.getByText('paper-exchange-1')).toBeVisible();
    await expect(page.getByText('0.1000 / 0.0400')).toBeVisible();
    await expect(page.getByText('65010.00')).toBeVisible();
    await expect(page.getByText('partially_filled')).toBeVisible();
    await expect(page.getByText('paper-client-2')).toBeVisible();
    await expect(page.getByText('exchange timeout after retry policy')).toBeVisible();
    await expect(page.getByText('failed')).toBeVisible();
  });
});
