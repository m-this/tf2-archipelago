import { expect, test } from './fixtures';

test.describe('the Play screen', () => {
  test('shows the connect line and how to use it', async ({ page }) => {
    await page.goto('/session');

    await expect(page.getByText('connect 127.0.0.1:27015; password ""')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Join with Steam' })).toBeEnabled();
    await expect(page.getByText('Or open the TF2 console and paste the line above.')).toBeVisible();
  });

  test('says what the run has handed this slot, and what it has not', async ({ page }) => {
    await page.goto('/session');
    const glance = page.getByRole('region', { name: 'What you can play' });

    await expect(glance.getByText('Classes · 2 of 9')).toBeVisible();
    // A locked mercenary is shown, not hidden: what you cannot play yet is half
    // of what the panel is telling you.
    await expect(glance.getByTitle('Scout is unlocked')).toBeVisible();
    await expect(glance.getByTitle('Pyro is still locked')).toBeVisible();
    await expect(glance.getByText('Latest:')).toBeVisible();
  });

  test('shows who holds the other seats, with a way to change them', async ({ page }) => {
    await page.goto('/session');
    const team = page.getByRole('region', { name: 'Your bot team' });

    await expect(team.getByText('the mod picks').first()).toBeVisible();
    await expect(team.getByText(/RED fills to 6, humans included/)).toBeVisible();

    await team.getByRole('link', { name: /Change/ }).click();
    await expect(page).toHaveURL(/\/bots$/);
  });

  test('tells played apart from cleared elsewhere', async ({ page }) => {
    await page.goto('/session');
    await expect(
      page.getByRole('row', { name: /Disk Deletion/ }).getByText('played'),
    ).toBeVisible();

    // Another world's !collect sends every check it still holds, so the room can
    // hold a mission's check that this server never played. The two words are
    // the whole reason the column exists.
    await expect(
      page.getByRole('row', { name: /Mean Machines/ }).getByText('cleared elsewhere'),
    ).toBeVisible();
  });

  test('carries the tier, which the bridge does not send', async ({ page }) => {
    await page.goto('/session');
    // gamedata knows Crash Course is Normal and Desperation is Expert. A browser
    // guessing that from the name would be guessing.
    await expect(page.getByRole('row', { name: /Crash Course/ }).getByText('Normal')).toBeVisible();
    await expect(page.getByRole('row', { name: /Desperation/ }).getByText('Expert')).toBeVisible();
  });

  test('offers Play only where it would do something', async ({ page }) => {
    await page.goto('/session');
    const locked = page.getByRole('row', { name: /Broken Parts/ });
    const ready = page.getByRole('row', { name: /Crash Course/ });

    // Stopped: nothing to load a mission onto.
    await expect(ready.getByRole('button', { name: 'Play' })).toHaveCount(0);

    await page.getByRole('button', { name: 'Start server' }).click();
    await expect(ready.getByRole('button', { name: 'Play' })).toBeVisible();
    // Still nothing for a mission the run has not unlocked.
    await expect(locked.getByRole('button', { name: 'Play' })).toHaveCount(0);

    await ready.getByRole('button', { name: 'Play' }).click();
    await expect(page.getByText('next mission is mvm_coaltown')).toBeVisible();
  });

  /* A wave picker beside Play, so a team can go straight to the wave they
     want rather than replaying a mission to reach it. It starts on the wave
     the team got to, which is the one they came back for. */
  test('offers a wave to start at, beside Play', async ({ page }) => {
    await page.goto('/session');
    const started = page.getByRole('row', { name: /Ctrl\+Alt\+Destruction/ });
    const picker = started.getByLabel('Start Ctrl+Alt+Destruction at a wave');

    // Nothing to load a mission into until the server is up.
    await expect(picker).toHaveCount(0);
    await page.getByRole('button', { name: 'Start server' }).click();

    await expect(picker).toBeVisible();
    await expect(picker).toHaveValue('4');
    await expect(started.getByRole('button', { name: 'Play' })).toBeVisible();

    await picker.selectOption('6');
    await expect(page.getByText(/resuming mvm_coaltown_advanced at wave 6/)).toBeVisible();
  });

  test('sorts on any column', async ({ page }) => {
    await page.goto('/session');
    const names = page.locator('td.name');
    await page.getByRole('columnheader', { name: 'Mission' }).getByRole('button').click();
    await expect(names.first()).toHaveText('Broken Parts');
  });
});
