import { expect, test } from './fixtures';

test('recruits named defenders and reorders their persistent seats', async ({ page }) => {
  await page.goto('/bots');
  const collection = page.getByRole('region', { name: 'Bot card collection' });
  const squad = page.getByRole('region', { name: 'Selected bot priority' });

  await collection.getByRole('button', { name: 'Recruit CreditToTeam' }).click();
  await expect(squad.getByRole('heading', { name: 'CreditToTeam' })).toBeVisible();
  await collection.getByRole('button', { name: 'Recruit Herr Doktor' }).click();
  await expect(squad.getByRole('heading', { name: 'Herr Doktor' })).toBeVisible();

  await squad
    .locator('.entry')
    .nth(1)
    .locator('.drag-handle')
    .dragTo(squad.locator('.entry').first(), {
      targetPosition: { x: 12, y: 12 },
      steps: 25,
    });
  await expect(squad.locator('.entry').first()).toContainText('Herr Doktor');
  await expect(page.getByLabel('Seat 1', { exact: true })).toHaveValue('medic');
  await expect(page.getByLabel('Name for Seat 1')).toHaveValue('Herr Doktor');
  await expect(page.getByLabel('Loadout for Seat 1')).toHaveValue('kritz');
  if (process.env['TF2AP_CAPTURE'] === '1') {
    await page.addStyleTag({ content: 'header, footer { position: static !important; }' });
    await page
      .locator('app-panel')
      .first()
      .screenshot({ path: '../../dist/botcards-admin-demo.png' });
  }
});

test('a full manual team can recruit cards without a false full warning', async ({ page }) => {
  await page.goto('/bots');
  for (let seat = 1; seat <= 6; seat++) {
    await page.getByLabel(`Seat ${seat}`, { exact: true }).selectOption('scout');
  }
  const collection = page.getByRole('region', { name: 'Bot card collection' });
  await expect(
    collection.getByRole('button', { name: 'Replace manual Seat 6 with CreditToTeam' }),
  ).toBeVisible();
  await collection.getByRole('button', { name: 'Replace manual Seat 6 with CreditToTeam' }).click();
  await expect(page.getByRole('region', { name: 'Selected bot priority' })).toContainText(
    'CreditToTeam',
  );
  await expect(
    collection.getByRole('button', { name: 'Replace manual Seat 5 with Herr Doktor' }),
  ).toBeVisible();
  await collection.getByRole('button', { name: 'Replace manual Seat 5 with Herr Doktor' }).click();
  await expect(
    page.getByRole('region', { name: 'Selected bot priority' }).locator('.entry'),
  ).toHaveCount(2);
  await expect(page.getByRole('status')).not.toContainText('All six seats are filled');
});

test('recruits and previews a full six-card team', async ({ page }) => {
  await page.goto('/bots');
  for (const name of [
    'CreditToTeam',
    "Screamin' Eagles",
    'IvanTheSpaceBiker',
    'Herr Doktor',
    'Chell',
    'Mentlegen',
  ]) {
    await page
      .getByRole('region', { name: 'Bot card collection' })
      .getByRole('button', { name: `Recruit ${name}` })
      .click();
    await expect(
      page.getByRole('region', { name: 'Selected bot priority' }).getByRole('heading', { name }),
    ).toBeVisible();
  }
  const squad = page.getByRole('region', { name: 'Selected bot priority' });
  await expect(squad.locator('.entry')).toHaveCount(6);
  await expect(page.getByLabel('Name for Seat 6')).toHaveValue('Mentlegen');
  await expect(squad.locator('.trading-card[data-rarity="common"]')).toHaveCount(2);
  await expect(squad.locator('.trading-card[data-rarity="elite"]')).toHaveCount(2);
  await expect(squad.locator('.trading-card[data-rarity="legendary"]')).toHaveCount(2);
  // A Legendary Medic carries twice the class's 150.
  await expect(
    squad.locator('.trading-card').filter({ hasText: 'Herr Doktor' }).locator('.stat-cell'),
  ).toContainText('300');
  await expect(
    squad.locator('.trading-card').filter({ hasText: 'Herr Doktor' }).locator('.perk li'),
  ).toHaveCount(3);
  await expect(
    squad.locator('.trading-card').filter({ hasText: 'Herr Doktor' }).locator('.perk'),
  ).toContainText('Damage +30%');
  await expect(
    squad.locator('.trading-card').filter({ hasText: 'Herr Doktor' }).locator('.perk'),
  ).toContainText('ÜberCharge rate +30%');
  await expect(
    squad.locator('.trading-card').filter({ hasText: 'IvanTheSpaceBiker' }).locator('.perk'),
  ).toContainText('Clip size +50%');
  await expect(
    squad.locator('.trading-card').filter({ hasText: 'Mentlegen' }).locator('.perk'),
  ).toContainText('Armor piercing +75%');
  await squad.getByRole('combobox', { name: 'Form for Herr Doktor' }).selectOption('giant');
  await expect(squad.locator('.trading-card.giant').filter({ hasText: 'Herr Doktor' })).toHaveCount(
    1,
  );
  await expect(squad.getByRole('combobox', { name: 'Form for Herr Doktor' })).toHaveValue('giant');
  await expect(
    squad
      .locator('.trading-card.giant')
      .filter({ hasText: 'Herr Doktor' })
      .locator('.stat-cell')
      .first(),
  ).toContainText('600');
  await squad.getByRole('combobox', { name: 'Form for Mentlegen' }).selectOption('robot');
  await expect(
    squad.locator('.trading-card').filter({ hasText: 'Mentlegen' }).locator('.stat-cell').first(),
  ).toContainText('225');
  const statOverflow = await squad
    .locator('.stats')
    .evaluateAll((stats) => stats.some((stat) => stat.scrollWidth > stat.clientWidth + 1));
  expect(statOverflow).toBe(false);
  await page.setViewportSize({ width: 390, height: 844 });
  const mobileStatOverflow = await squad
    .locator('.stats')
    .evaluateAll((stats) => stats.some((stat) => stat.scrollWidth > stat.clientWidth + 1));
  expect(mobileStatOverflow).toBe(false);
  await expect(page.getByRole('button', { name: 'Apply' })).toBeEnabled();
});

test('tracker displays unlocked cards with tier and innate stacks', async ({ page }) => {
  await page.goto('http://127.0.0.1:8472/?demo=1');
  const collection = page.locator('.bot-card-grid');
  await expect(collection.locator('.trading-card')).toHaveCount(3);
  await expect(collection.locator('.trading-card[data-rarity="common"]')).toHaveCount(1);
  await expect(collection.locator('.trading-card[data-rarity="elite"]')).toHaveCount(1);
  await expect(collection.locator('.trading-card[data-rarity="legendary"]')).toHaveCount(1);
  await expect(collection.locator('.trading-card.human')).toHaveCount(1);
  await expect(collection.locator('.trading-card.giant')).toHaveCount(1);
  await expect(collection).toContainText('RED MERC');
  await expect(collection).toContainText('INNATE');
  await expect(collection).toContainText('Damage +30%');
  await expect(collection).toContainText('Unusual Blighted Beak');
  const wikiRobotImages = collection.locator('img[src*="Robot_"]');
  await expect(wikiRobotImages).toHaveCount(2);
  expect(
    await wikiRobotImages.evaluateAll((images) =>
      images.every((image) => (image as HTMLImageElement).naturalWidth > 0),
    ),
  ).toBe(true);
  if (process.env['TF2AP_CAPTURE'] === '1') {
    await collection.screenshot({ path: '../../dist/botcards-tracker-demo.png' });
  }
});
