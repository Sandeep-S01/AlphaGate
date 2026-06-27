import { expect, test } from '@playwright/test';

test.describe('Controlled trading readiness', () => {
  const routeTradingShell = async (page) => {
    await page.route('**/api/v1/market/candles?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [
          { open_time: '2026-06-26T00:00:00Z', open: '100', high: '102', low: '99', close: '101', volume: '10' },
          { open_time: '2026-06-26T00:01:00Z', open: '101', high: '103', low: '100', close: '102', volume: '12' }
        ]
      })
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
  };

  test('shows live blocked reasons from lifecycle, risk, kill switch, and reconciliation state', async ({ page }) => {
    await routeTradingShell(page);
    await page.route('**/api/v1/safety/status', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { kill_switch_active: true, reason: 'operator verification hold' } })
    }));
    await page.route('**/api/v1/strategy/lifecycle?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [{
          id: 'life-1',
          strategy_name: 'multi-factor-momentum',
          symbol: 'BTCUSDT',
          interval: '1h',
          state: 'PAPER_TRADING',
          reason: 'Needs operator approval after paper evidence'
        }]
      })
    }));
    await page.route('**/api/v1/reconciliation/runs?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [{ id: 'recon-1', status: 'mismatch', mismatches: [{ type: 'balance', reason: 'USDT mismatch' }] }] })
    }));
    await page.route('**/api/v1/risk/settings', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          enabled: true,
          allowed_symbols: ['BTCUSDT'],
          max_order_quote_amount: 250,
          max_total_exposure_quote_amount: 1000,
          max_open_positions: 1,
          allow_buy: true,
          allow_sell: false
        }
      })
    }));
    await page.route('**/api/v1/risk-decisions?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [{
          id: 'risk-1',
          symbol: 'BTCUSDT',
          signal_side: 'sell',
          decision: 'rejected',
          reason: 'sell side disabled by risk settings'
        }]
      })
    }));
    await page.route('**/api/v1/execution/status', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          mode: 'paper',
          paper_enabled: true,
          exchange_adapter: 'binance_disabled',
          live_trading_enabled: false,
          retry_attempts: 3,
          timeout: '5s',
          last_error: 'binance live trading is disabled'
        }
      })
    }));

    await page.goto('/');

    await expect(page.getByText('LIVE BLOCKED')).toBeVisible();
    await expect(page.getByText(/Kill switch is armed/)).toBeVisible();
    await expect(page.getByText(/Lifecycle is PAPER_TRADING/)).toBeVisible();
    await expect(page.getByText(/Reconciliation mismatch detected/)).toBeVisible();
    await expect(page.getByText(/Live exchange adapter is disabled/)).toBeVisible();
    await expect(page.getByText(/Latest risk rejection/i)).toBeVisible();
    await expect(page.getByText(/sell side disabled by risk settings/i)).toBeVisible();
    const readiness = page.locator('section').filter({ hasText: 'Trade Readiness' });
    await expect(readiness.getByText(/Allowed Symbols/i)).toBeVisible();
    await expect(readiness.getByText('BTCUSDT')).toBeVisible();
    await expect(readiness.getByText(/Max Order/i)).toBeVisible();
    await expect(readiness.getByText('$250.00')).toBeVisible();
    await expect(readiness.getByText(/Execution Adapter/i)).toBeVisible();
    await expect(readiness.getByText(/binance_disabled/i)).toBeVisible();
    await expect(readiness.getByText(/Retry Policy/i)).toBeVisible();
    await expect(readiness.getByText(/3 attempts \/ 5s/i)).toBeVisible();
  });

  test('advances lifecycle one gate at a time from the readiness panel', async ({ page }) => {
    await routeTradingShell(page);

    let advancedPayload = null;
    let lifecycleState = 'PAPER_TRADING';
    await page.route('**/api/v1/safety/status', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { kill_switch_active: false } })
    }));
    await page.route('**/api/v1/strategy/lifecycle?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [{
          id: 'life-1',
          strategy_name: 'multi-factor-momentum',
          symbol: 'BTCUSDT',
          interval: '1h',
          state: lifecycleState,
          reason: lifecycleState === 'PAPER_TRADING' ? 'Paper soak complete' : 'Operator approved paper evidence'
        }]
      })
    }));
    await page.route('**/api/v1/strategy/lifecycle/life-1/advance', async route => {
      advancedPayload = JSON.parse(route.request().postData() || '{}');
      lifecycleState = advancedPayload.state;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: 'life-1',
            strategy_name: 'multi-factor-momentum',
            symbol: 'BTCUSDT',
            interval: '1h',
            state: lifecycleState,
            reason: advancedPayload.reason,
            updated_by: advancedPayload.updated_by
          }
        })
      });
    });
    await page.route('**/api/v1/reconciliation/runs?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [{ id: 'recon-1', status: 'matched', mismatches: [] }] })
    }));
    await page.route('**/api/v1/risk/settings', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { enabled: true, allowed_symbols: ['BTCUSDT'], allow_buy: true, allow_sell: true } })
    }));
    await page.route('**/api/v1/risk-decisions?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [] })
    }));
    await page.route('**/api/v1/execution/status', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          mode: 'paper',
          paper_enabled: true,
          exchange_adapter: 'binance_disabled',
          live_trading_enabled: false,
          retry_attempts: 3,
          timeout: '5s',
          last_error: 'binance live trading is disabled'
        }
      })
    }));

    await page.goto('/');

    const readiness = page.locator('section').filter({ hasText: 'Trade Readiness' });
    await expect(readiness.locator('span').filter({ hasText: /^PAPER_TRADING$/ })).toBeVisible();
    await expect(readiness.getByRole('button', { name: 'Advance lifecycle to APPROVED' })).toBeVisible();
    await expect(readiness.getByRole('button', { name: 'Advance lifecycle to LIVE_ENABLED' })).toHaveCount(0);

    await readiness.getByRole('button', { name: 'Advance lifecycle to APPROVED' }).click();

    await expect(readiness.locator('span').filter({ hasText: /^APPROVED$/ })).toBeVisible();
    await expect(readiness.getByRole('button', { name: 'Advance lifecycle to LIVE_ENABLED' })).toBeVisible();
    expect(advancedPayload).toMatchObject({
      state: 'APPROVED',
      updated_by: 'operator'
    });
  });

  test('keeps live blocked when execution status is unavailable', async ({ page }) => {
    await routeTradingShell(page);

    await page.route('**/api/v1/safety/status', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { kill_switch_active: false } })
    }));
    await page.route('**/api/v1/strategy/lifecycle?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: [{
          id: 'life-1',
          strategy_name: 'multi-factor-momentum',
          symbol: 'BTCUSDT',
          interval: '1h',
          state: 'LIVE_ENABLED',
          reason: 'approved for live foundation test'
        }]
      })
    }));
    await page.route('**/api/v1/reconciliation/runs?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [{ id: 'recon-1', status: 'matched', mismatches: [] }] })
    }));
    await page.route('**/api/v1/risk/settings', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: { enabled: true, allowed_symbols: ['BTCUSDT'], allow_buy: true, allow_sell: true } })
    }));
    await page.route('**/api/v1/risk-decisions?**', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ data: [] })
    }));
    await page.route('**/api/v1/execution/status', route => route.fulfill({
      status: 503,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'execution status unavailable' })
    }));

    await page.goto('/');

    const readiness = page.locator('section').filter({ hasText: 'Trade Readiness' });
    await expect(readiness.getByText('LIVE BLOCKED')).toBeVisible();
    await expect(readiness.getByText(/Execution status unavailable/)).toBeVisible();
    await expect(readiness.getByText('LIVE READY')).toHaveCount(0);
  });
});
