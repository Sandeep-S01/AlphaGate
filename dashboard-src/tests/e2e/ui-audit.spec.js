import { expect, test } from '@playwright/test';

test.describe('Sentra UI audit behavior', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('keeps the audited trading information architecture visible', async ({ page }) => {
    await expect(page.getByRole('button', { name: 'Trading' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Research' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Strategy' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Risk' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Ops' })).toBeVisible();

    const logTabs = page.locator('button', { hasText: /^(orders|trades|signals|audit|events)$/i });
    await expect(logTabs).toHaveCount(5);
    await expect(page.getByRole('button', { name: 'events' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'L2 Book' })).toHaveCount(0);

    await page.getByRole('button', { name: 'events' }).click();
    await expect(page.getByText('L2 Book Snapshot')).toBeVisible();
    await expect(page.getByText(/No L2 book levels loaded/)).toBeVisible();
  });

  test('shows prescriptive empty states and stale status feedback', async ({ page }) => {
    await expect(page.getByText(/No orders logged\. Strategy is running in paper mode/)).toBeVisible();
    await expect(page.getByRole('status', { name: /API stale|API offline|API online/ })).toBeVisible();
    await expect(page.getByLabel(/BTCUSDT market context/)).toBeVisible();
  });

  test('supports keyboard command palette and settings modal accessibility', async ({ page }) => {
    await page.locator('body').click({ position: { x: 10, y: 10 } });
    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+K' : 'Control+K');
    await expect(page.getByLabel('Search commands')).toBeFocused();
    await expect(page.getByRole('button', { name: /Switch Workspace to \[ Trading \]/ })).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.getByLabel('Search commands')).toHaveCount(0);

    await page.getByRole('button', { name: 'Configure settings' }).click();
    await expect(page.getByRole('dialog', { name: 'Terminal Configurations' })).toBeVisible();
    await expect(page.getByLabel('Close settings')).toBeFocused();
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: 'Terminal Configurations' })).toHaveCount(0);
  });

  test('requires hold confirmation for high-risk trading actions', async ({ page }) => {
    const buyButton = page.getByRole('button', { name: 'Hold to confirm manual buy order' });
    await expect(buyButton).toBeVisible();
    await buyButton.dispatchEvent('pointerdown');
    await expect(buyButton).toContainText(/Hold to confirm BUY/);
    await buyButton.dispatchEvent('pointerup');
    await expect(buyButton).toContainText('Order Buy');

    const safetyButton = page.getByRole('button', { name: /Hold to (arm|disarm) safety block/i });
    await safetyButton.dispatchEvent('pointerdown');
    await expect(safetyButton).toContainText(/Hold to confirm/);
    await safetyButton.dispatchEvent('pointerup');
  });

  test('shows backtest cost diagnostics after executing research backtest', async ({ page }) => {
    await page.getByRole('button', { name: 'Research' }).click();
    await page.getByRole('button', { name: 'Execute Backtest' }).click();

    await expect(page.getByText('GROSS PNL')).toBeVisible({ timeout: 20000 });
    await expect(page.getByText('FEES')).toBeVisible();
    await expect(page.getByText('SLIPPAGE COST')).toBeVisible();
    await expect(page.getByText('ROUND TRIP COST')).toBeVisible();
    await expect(page.getByText('BREAK EVEN MOVE')).toBeVisible();
    await expect(page.getByText('Execution Failed')).toHaveCount(0);
  });
});
