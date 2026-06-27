import { expect, test } from '@playwright/test';

test.describe('Risk settings hardening controls', () => {
  test('edits hardened risk settings and includes them in the save payload', async ({ page }) => {
    let savedPayload = null;

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
    await page.route('**/api/v1/paper/orders?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/paper/trades?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/signals?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/audit/events?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/ops/pipeline-runs?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/ops/streams', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/safety/status', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: { kill_switch_active: false } }) }));
    await page.route('**/api/v1/strategy/lifecycle?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/reconciliation/runs?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/risk-decisions?**', route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [] }) }));
    await page.route('**/api/v1/risk/settings', async route => {
      if (route.request().method() === 'PUT') {
        savedPayload = JSON.parse(route.request().postData() || '{}');
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: savedPayload })
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            enabled: true,
            min_signal_strength: 0.2,
            max_signal_strength: 0.95,
            max_quote_amount: 500,
            max_order_quote_amount: 250,
            max_position_quote_amount: 750,
            max_total_exposure_quote_amount: 1500,
            max_open_positions: 2,
            max_daily_loss: 150,
            max_daily_trades: 10,
            allow_buy: true,
            allow_sell: true,
            allowed_symbols: ['BTCUSDT', 'ETHUSDT'],
            cooldown_seconds: 60
          }
        })
      });
    });

    await page.goto('/');
    await page.getByRole('button', { name: 'Risk' }).click();

    await expect(page.getByLabel('Allowed symbols')).toHaveValue('BTCUSDT, ETHUSDT');
    await expect(page.getByLabel('Max order quote amount')).toHaveValue('250');
    await expect(page.getByLabel('Max position quote amount')).toHaveValue('750');
    await expect(page.getByLabel('Max total exposure quote amount')).toHaveValue('1500');
    await expect(page.getByLabel('Max open positions')).toHaveValue('2');

    await page.getByLabel('Allowed symbols').fill('BTCUSDT, SOLUSDT');
    await page.getByLabel('Max order quote amount').fill('300');
    await page.getByLabel('Max position quote amount').fill('900');
    await page.getByLabel('Max total exposure quote amount').fill('1800');
    await page.getByLabel('Max open positions').fill('3');
    await page.getByRole('button', { name: /Save Parameters/i }).click();

    expect(savedPayload).toMatchObject({
      allowed_symbols: ['BTCUSDT', 'SOLUSDT'],
      max_order_quote_amount: 300,
      max_position_quote_amount: 900,
      max_total_exposure_quote_amount: 1800,
      max_open_positions: 3
    });
  });
});
