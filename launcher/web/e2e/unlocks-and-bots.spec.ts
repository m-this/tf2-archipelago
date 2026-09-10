import { expect, test } from './fixtures';

test.describe('the Unlocks screen', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/unlocks');
  });

  test('counts them and says where to see the buffs in the game', async ({ page }) => {
    await expect(page.getByText(/unlocks so far/)).toBeVisible();
    await expect(page.getByText('!ap buffs')).toBeVisible();
  });

  test('shows a buff held twice as a level, and everything else as a dash', async ({ page }) => {
    await expect(
      page.getByRole('row', { name: /Scattergun damage/ }).getByText('×2'),
    ).toBeVisible();
    await expect(page.getByRole('row', { name: /Scout/ }).first().getByText('-')).toBeVisible();
  });

  // The kinds are a closed set the bridge names. A chip that appeared only once
  // something of that kind arrived would move the other chips under the
  // player's finger.
  test('offers the same filters whatever has arrived', async ({ page }) => {
    const filters = page.getByRole('group', { name: 'Filter by kind' });
    for (const name of [
      'All',
      'Classes',
      'Weapon slots',
      'Missions',
      'Weapon buffs',
      'Server levers',
    ]) {
      await expect(filters.getByRole('button', { name, exact: true })).toBeVisible();
    }

    await filters.getByRole('button', { name: 'Classes' }).click();
    await expect(page.getByText('Scattergun damage')).toHaveCount(0);
    await expect(page.getByRole('row', { name: /Soldier/ })).toBeVisible();

    await filters.getByRole('button', { name: 'All' }).click();
    await expect(page.getByText('Scattergun damage')).toBeVisible();
  });
});

test.describe('the Bots screen', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/bots');

    // The page opens the draft itself once the first frame says it is closed.
    await expect(page.getByRole('heading', { name: 'Saved lineups' })).toBeVisible();
  });

  test('gives every seat a class and a loadout to choose', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'The lineup' })).toBeVisible();
    await expect(page.getByLabel('Seat 1', { exact: true })).toBeVisible();
    await expect(page.getByLabel('Loadout for Seat 1')).toBeVisible();

    await page.getByLabel('Seat 1', { exact: true }).selectOption({ label: 'Soldier' });
    await expect(page.getByText(/Seat 1 → Soldier/)).toBeVisible();
  });

  // A whole lineup was built here once and was gone on the next visit: the
  // seats are a draft, and Apply is what writes it.
  test('has Apply, and it is offered once a seat changes', async ({ page }) => {
    await expect(page.getByRole('button', { name: 'Apply' })).toBeDisabled();
    await page.getByLabel('Seat 2', { exact: true }).selectOption({ label: 'Medic' });
    await expect(page.getByRole('button', { name: 'Apply' })).toBeEnabled();
    await page.getByRole('button', { name: 'Apply' }).click();
    await expect(page.getByText('Applied.')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Apply' })).toBeDisabled();

    // The lineup is drawn again from the saved settings, and it used to come
    // back as stock: the select's value was set before its options existed.
    await expect(page.getByLabel('Seat 2', { exact: true })).toHaveValue('medic');
  });

  test('sets how many RED fills to, humans included', async ({ page }) => {
    await expect(page.getByLabel('Fill RED to')).toBeVisible();
    await expect(page.getByText('players, humans included.')).toBeVisible();
  });

  test('saves the current seats under a name', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Saved lineups' })).toBeVisible();
    await expect(page.getByLabel('Lineup name')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save as new' })).toBeVisible();
  });

  test('the switcher will not leave RED with nothing to draw from', async ({ page }) => {
    const chips = page.getByRole('group', { name: 'Classes the mod may draw' });
    const names = ['Scout', 'Soldier', 'Pyro', 'Demoman', 'Heavy', 'Engineer', 'Medic', 'Sniper'];
    for (const name of names) {
      await chips.getByRole('button', { name, exact: true }).click();
    }
    await expect(chips.getByRole('button', { name: 'Scout', exact: true })).toHaveAttribute(
      'aria-pressed',
      'false',
    );

    // The ninth press is refused: a lineup the mod cannot draw from leaves the
    // seats empty, and an empty seat is a wave short of six defenders.
    await chips.getByRole('button', { name: 'Spy', exact: true }).click();
    await expect(chips.getByRole('button', { name: 'Spy', exact: true })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
  });
});
