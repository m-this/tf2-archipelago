import type { Page } from '@playwright/test';

import { expect, test } from './fixtures';

// Settings pages and the app's own sections share names: Bots is both.
const settingsPages = (page: Page) => page.getByRole('navigation', { name: 'Settings pages' });

/**
 * The settings screen is a renderer over form.Model: no page here is written by
 * hand. What these check is that the real model, encoded by the real converter,
 * draws the right control for each Kind and sends an answer back that the Go
 * side applies.
 */
test.describe('the settings screen', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/settings');
    await expect(page.getByRole('link', { name: 'Player options' })).toBeVisible();
  });

  test('lands on the first page when the URL names none', async ({ page }) => {
    await expect(page).toHaveURL(/\/settings\/player-options$/);
    await expect(page.getByRole('heading', { name: 'Player options' })).toBeVisible();

    // And again when the draft is already open: the page is what was missing.
    await page.goto('/settings');
    await expect(page).toHaveURL(/\/settings\/player-options$/);
  });

  test('draws the pages the model declares, not a list written here', async ({ page }) => {
    const pages = page.getByRole('navigation', { name: 'Settings pages' });
    for (const name of [
      'Player options',
      'Rewards',
      'Balancing',
      'Missions',
      'Archipelago room',
      'Game server',
      'Bots',
      'Networking',
    ]) {
      await expect(pages.getByRole('link', { name, exact: true })).toBeVisible();
    }

    // Loadouts is filed under Bots, so it is a tab of that page and not a
    // page of its own.
    await expect(pages.getByRole('link', { name: 'Loadouts', exact: true })).toHaveCount(0);
    await pages.getByRole('link', { name: 'Bots', exact: true }).click();
    const sections = page.getByRole('tablist', { name: 'Sections of Bots' });
    for (const name of ['Team', 'Classes', 'Looks', 'Loadouts']) {
      await expect(sections.getByRole('tab', { name })).toBeVisible();
    }
  });

  test('a page with sections shows one at a time, as tabs', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Bots', exact: true }).click();
    await expect(page.getByLabel('Seat 1', { exact: true })).toBeVisible();
    await expect(page.getByLabel('Cosmetic items')).toHaveCount(0);

    await page.getByRole('tab', { name: 'Looks' }).click();
    await expect(page).toHaveURL(/\/settings\/bots\/looks$/);
    await expect(page.getByLabel('Cosmetic items')).toBeVisible();
    await expect(page.getByLabel('Seat 1', { exact: true })).toHaveCount(0);

    // A search flattens the tabs: the setting you cannot name the tab of is
    // the one you search for.
    await page.getByRole('searchbox', { name: 'Search settings' }).fill('unusual');
    await expect(page.getByLabel('Unusual effects')).toBeVisible();
  });

  test('the loadout builder saves, loads and removes', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Bots', exact: true }).click();
    await page.getByRole('tab', { name: 'Loadouts' }).click();

    const saved = page.getByRole('list', { name: 'Saved loadouts' });
    await expect(saved.getByText('gas runner (Pyro)')).toBeVisible();

    await page.getByRole('group', { name: 'Class' }).getByRole('button', { name: 'Scout' }).click();
    await page.getByLabel('Primary').selectOption({ label: 'The Soda Popper' });
    await page.getByLabel('Name').fill('pop');
    await page.getByRole('button', { name: 'Save this loadout' }).click();
    await expect(saved.getByText('pop (Scout)')).toBeVisible();

    // Two presses to remove: one asks, the second does it.
    const row = saved.getByRole('listitem').filter({ hasText: 'pop (Scout)' });
    await row.getByRole('button', { name: 'Remove' }).click();
    await row.getByRole('button', { name: 'Really remove' }).click();
    await expect(saved.getByText('pop (Scout)')).toHaveCount(0);
  });

  // The footer offered to go next from the page the launcher had opened rather
  // than the one on screen, so Next said "Player options" while the player was
  // standing on Missions.
  test('the footer walks the pages the player is actually on', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Missions', exact: true }).click();
    await expect(page.getByText('Section 4 of 8')).toBeVisible();
    await expect(page.getByRole('button', { name: /Next: Archipelago room/ })).toBeVisible();

    await page.getByRole('button', { name: /Next: Archipelago room/ }).click();
    await expect(page).toHaveURL(/\/settings\/archipelago-room$/);
    await expect(page.getByText('Section 5 of 8')).toBeVisible();
  });

  test('every kind the model uses gets its own control', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Game server', exact: true }).click();
    await expect(page.getByLabel('Server name')).toHaveAttribute('type', 'text');
    await expect(page.getByLabel('Server password')).toHaveAttribute('type', 'password');
    await expect(page.getByLabel('Game port')).toHaveAttribute('type', 'number');

    await settingsPages(page).getByRole('link', { name: 'Rewards', exact: true }).click();
    await expect(page.getByLabel('Mission tickets')).toHaveRole('combobox');
    await expect(page.getByLabel('Cash rewards')).toHaveRole('checkbox');
  });

  test('a typed answer reaches the launcher and comes back applied', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Rewards', exact: true }).click();
    await page.getByLabel('Traps (%)').fill('42');

    // Applied by form.Apply on the Go side and rebuilt on the stream: leaving
    // the page and coming back reads the launcher's answer, not a local one.
    await settingsPages(page).getByRole('link', { name: 'Balancing', exact: true }).click();
    await settingsPages(page).getByRole('link', { name: 'Rewards', exact: true }).click();
    await expect(page.getByLabel('Traps (%)')).toHaveValue('42');
  });

  test('a Confirm asks before it does anything', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Game server', exact: true }).click();
    await page.getByRole('button', { name: 'Reset settings' }).click();

    await expect(page.getByRole('button', { name: 'Yes, do it' })).toBeVisible();
    await page.getByRole('button', { name: 'Cancel' }).click();
    await expect(page.getByRole('button', { name: 'Reset settings' })).toBeVisible();
  });

  test('a Browse row offers a folder picker, because a browser has no dialog', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Player options', exact: true }).click();
    await page.getByRole('button', { name: 'Browse' }).first().click();

    const picker = page.getByRole('dialog', { name: 'Choose a folder' });
    await expect(picker).toBeVisible();
    await expect(picker.getByRole('button', { name: 'Up' })).toBeVisible();
    await picker.getByRole('button', { name: 'Cancel' }).click();
    await expect(picker).toHaveCount(0);
  });

  test('a Show button opens the file and says which one', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Player options', exact: true }).click();
    await page.getByRole('button', { name: 'Show the settings file' }).click();
    await expect(page.getByText('opened /home/player/.config/tf2ap/settings.json')).toBeVisible();
  });

  // The footer is the only thing that says whether there is anything to save.
  // It went quiet once when the store lost track of what had been answered, and
  // Save sat disabled over a screen full of changes.
  test('says whether there is anything to save, and only offers Save then', async ({ page }) => {
    await expect(page.getByText('All saved')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
    await expect(page.getByRole('button', { name: 'Discard' })).toBeDisabled();

    await settingsPages(page).getByRole('link', { name: 'Rewards', exact: true }).click();
    await page.getByLabel('Traps (%)').fill('42');

    await expect(page.getByText('Unsaved changes')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeEnabled();
  });

  test('Save writes what was typed', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Rewards', exact: true }).click();
    await page.getByLabel('Traps (%)').fill('37');
    await expect(page.getByText('Unsaved changes')).toBeVisible();

    await page.getByRole('button', { name: 'Save', exact: true }).click();
    await expect(page.getByText('All saved')).toBeVisible();

    await settingsPages(page).getByRole('link', { name: 'Balancing', exact: true }).click();
    await settingsPages(page).getByRole('link', { name: 'Rewards', exact: true }).click();
    await expect(page.getByLabel('Traps (%)')).toHaveValue('37');
  });

  test('Save is answered, and a refusal keeps the answers on screen', async ({ page }) => {
    await settingsPages(page).getByRole('link', { name: 'Player options', exact: true }).click();
    await page.getByLabel('Install folder').fill('');
    await page.getByRole('button', { name: 'Save', exact: true }).click();

    await expect(page.getByText('the server folder cannot be empty')).toBeVisible();
    await expect(page.getByLabel('Install folder')).toBeVisible();
  });
});

test.describe('the mission table', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/settings');
    await settingsPages(page).getByRole('link', { name: 'Missions', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Mission pool' })).toBeVisible();
  });

  test('opens sorted by name and sorts on any column', async ({ page }) => {
    const names = page.locator('td.name');
    const first = await names.first().textContent();

    await page.getByRole('columnheader', { name: 'Mission' }).getByRole('button').click();
    await expect(names.first()).not.toHaveText(first ?? '');
  });

  test('searching narrows it, and Tick shown means the rows on screen', async ({ page }) => {
    const all = await page.locator('td.name').count();
    expect(all).toBeGreaterThan(20);

    await page.getByRole('searchbox', { name: 'Find a mission' }).fill('mvm_coaltown');
    const shown = await page.locator('td.name').count();
    expect(shown).toBeGreaterThan(0);
    expect(shown).toBeLessThan(all);

    // Tick shown means the rows on screen, not every row there is: the search
    // is part of what the player meant by all.
    await page.getByRole('button', { name: 'Untick shown' }).click();
    await expect(page.getByText(`${all - shown} of ${all} in the pool`)).toBeVisible();

    await page.getByRole('button', { name: 'Tick shown', exact: true }).click();
    await expect(page.getByText(`${all} of ${all} in the pool`)).toBeVisible();
  });

  // The pool used to be drawn twice: a toggle row per mission above the table.
  test('is the only place a mission is ticked', async ({ page }) => {
    const table = page.getByRole('table');
    const rows = await table.getByRole('row').count();
    const ticks = await table.getByRole('checkbox').count();
    expect(ticks).toBe(rows - 1);
    // The four ticks above the table are the packs, community toggle and
    // managed SigMod toggle, not missions.
    expect(await page.getByRole('checkbox').count()).toBe(ticks + 4);
  });

  test('says why a mission is not ready rather than hiding it', async ({ page }) => {
    await expect(page.getByText('Below Advanced floor').first()).toBeVisible();
    await expect(page.getByText('Medieval').first()).toBeVisible();
    await expect(page.getByText('Community missions are off').first()).toBeVisible();
  });
});
