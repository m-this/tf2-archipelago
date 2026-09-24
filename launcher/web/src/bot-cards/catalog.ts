/* eslint-disable max-lines -- one catalog keeps the nine stock classes and six named cards together */
import { Mercenary, robotIcons } from '@app/ui/tf2-art';
import { cardInnates } from './innates.generated';

export type BotForm = 'human' | 'robot' | 'giant';

export interface BotCard {
  readonly id: string;
  readonly name: string;
  readonly className: Mercenary;
  readonly classKey: string;
  readonly loadoutKey: string;
  readonly rarity: 'COMMON' | 'ELITE' | 'LEGENDARY';
  readonly stars: number;
  readonly weapons: readonly [string, string, string];
  readonly cosmetic: string;
  readonly unusualEffect?: string;
  readonly innates: readonly string[];
  readonly health: number;
  readonly giantHealth: number;
  readonly loadoutHealthDelta: number;
  readonly aim: number;
  readonly reaction: string;
  readonly model: string;
}

// Curated test-mode prototypes. Each innate is a distinct AP-eligible effect
// found through an item this class can equip; it applies across the card's kit.
// Keep these per-stack values aligned with gamedata/weapon_effects.go. AP
// percentage bonuses compose linearly, so the card shows the resulting bonus.
const innatePercentPerStack = {
  Damage: 10,
  'Firing speed': 10,
  'Clip size': 25,
  'ÜberCharge rate': 10,
  'Armor piercing': 25,
} as const;

type InnateName = keyof typeof innatePercentPerStack;

export const botCards: readonly BotCard[] = [
  card(
    'stock-scout',
    'Chucklenuts',
    'Scout',
    'scout',
    'stock',
    'COMMON',
    ['Scattergun', 'Pistol', 'Bat'],
    'Ghastly Gibus',
    ['Damage'],
    125,
    56,
    0.18,
  ),
  card(
    'stock-soldier',
    'Maggot',
    'Soldier',
    'soldier',
    'stock',
    'COMMON',
    ['Rocket Launcher', 'Shotgun', 'Shovel'],
    'Ghastly Gibus',
    ['Damage'],
    200,
    59,
    0.26,
  ),
  card(
    'stock-pyro',
    'BeepBeepBoop',
    'Pyro',
    'pyro',
    'stock',
    'COMMON',
    ['Flame Thrower', 'Shotgun', 'Fire Axe'],
    'Ghastly Gibus',
    ['Damage'],
    175,
    55,
    0.24,
  ),
  card(
    'stock-demoman',
    'Kaboom!',
    'Demoman',
    'demoman',
    'stock',
    'COMMON',
    ['Grenade Launcher', 'Stickybomb Launcher', 'Bottle'],
    'Ghastly Gibus',
    ['Damage'],
    175,
    55,
    0.25,
  ),
  card(
    'stock-heavy',
    'Nom Nom Nom',
    'Heavy',
    'heavyweapons',
    'stock',
    'COMMON',
    ['Minigun', 'Shotgun', 'Fists'],
    'Ghastly Gibus',
    ['Damage'],
    300,
    53,
    0.31,
  ),
  card(
    'stock-engineer',
    'MoreGun',
    'Engineer',
    'engineer',
    'stock',
    'COMMON',
    ['Shotgun', 'Pistol', 'Wrench'],
    'Ghastly Gibus',
    ['Damage'],
    125,
    54,
    0.24,
  ),
  card(
    'stock-medic',
    'Archimedes!',
    'Medic',
    'medic',
    'stock',
    'COMMON',
    ['Syringe Gun', 'Medi Gun', 'Bonesaw'],
    'Ghastly Gibus',
    ['Damage'],
    150,
    48,
    0.2,
  ),
  card(
    'stock-sniper',
    'A Professional With Standards',
    'Sniper',
    'sniper',
    'stock',
    'COMMON',
    ['Sniper Rifle', 'SMG', 'Kukri'],
    'Ghastly Gibus',
    ['Damage'],
    125,
    49,
    0.2,
  ),
  card(
    'stock-spy',
    'Gentlemanne of Leisure',
    'Spy',
    'spy',
    'stock',
    'COMMON',
    ['Revolver', 'Sapper', 'Knife'],
    'Ghastly Gibus',
    ['Damage'],
    125,
    49,
    0.16,
  ),
  card(
    'credit-to-team',
    'CreditToTeam',
    'Scout',
    'scout',
    'milk',
    'COMMON',
    ['Soda Popper', 'Mad Milk', 'Fan O’War'],
    'Baseball Bill’s Sports Shine',
    ['Damage'],
    125,
    56,
    0.18,
  ),
  card(
    'screamin-eagles',
    "Screamin' Eagles",
    'Soldier',
    'soldier',
    'beggar',
    'ELITE',
    ['Beggar’s Bazooka', 'Buff Banner', 'Escape Plan'],
    'Team Captain',
    ['Damage', 'Firing speed'],
    200,
    59,
    0.26,
  ),
  card(
    'ivan',
    'IvanTheSpaceBiker',
    'Heavy',
    'heavyweapons',
    'brass',
    'ELITE',
    ['Brass Beast', 'Family Business', 'Fists of Steel'],
    'Heavy Do-rag',
    ['Firing speed', 'Clip size'],
    300,
    53,
    0.31,
  ),
  card(
    'herr-doktor',
    'Herr Doktor',
    'Medic',
    'medic',
    'kritz',
    'LEGENDARY',
    ['Crusader’s Crossbow', 'Kritzkrieg', 'Übersaw'],
    'Blighted Beak',
    ['Damage', 'ÜberCharge rate', 'Firing speed'],
    150,
    48,
    0.2,
    0,
    'Burning Flames',
  ),
  card(
    'chell',
    'Chell',
    'Engineer',
    'engineer',
    'ranger',
    'COMMON',
    ['Rescue Ranger', 'Wrangler', 'Jag'],
    'Prairie Heel Biters',
    ['Damage'],
    125,
    54,
    0.24,
  ),
  card(
    'mentlegen',
    'Mentlegen',
    'Spy',
    'spy',
    'diamondback',
    'LEGENDARY',
    ['Diamondback', 'Red-Tape Recorder', 'Big Earner'],
    'Fancy Fedora',
    ['Damage', 'Firing speed', 'Armor piercing'],
    125,
    49,
    0.16,
    -25, // Big Earner's equipped -25 max-health penalty.
    'Scorching Flames',
  ),
];

// AP item names carry the two independent seed rolls. Rebuild the visible
// stats from the base card so tracker and admin can show the received tier.
export function rolledCard(base: BotCard, rarity: BotCard['rarity']): BotCard {
  const oldScale = base.stars === 3 ? 2 : base.stars === 2 ? 1.5 : 1;
  const stars = rarity === 'LEGENDARY' ? 3 : rarity === 'ELITE' ? 2 : 1;
  const scale = stars === 3 ? 2 : stars === 2 ? 1.5 : 1;
  return {
    ...base,
    rarity,
    stars,
    health: Math.round(
      ((base.health - base.loadoutHealthDelta) / oldScale) * scale + base.loadoutHealthDelta,
    ),
    giantHealth: Math.round(
      ((base.health - base.loadoutHealthDelta) / oldScale) * scale * 2 + base.loadoutHealthDelta,
    ),
    aim: Math.min(100, Math.round((base.aim / oldScale) * scale)),
    reaction: `${((Number.parseFloat(base.reaction) * oldScale) / scale).toFixed(2)}s`,
    innates: cardInnates[base.id]?.[rarity.toLowerCase()] ?? base.innates,
    unusualEffect: rarity === 'LEGENDARY' ? (base.unusualEffect ?? 'Burning Flames') : undefined,
  };
}

export function cardById(id: string): BotCard | undefined {
  return botCards.find((entry) => entry.id === id);
}

function card(
  id: string,
  name: string,
  className: Mercenary,
  classKey: string,
  loadoutKey: string,
  rarity: BotCard['rarity'],
  weapons: BotCard['weapons'],
  cosmetic: string,
  innates: readonly InnateName[],
  health: number,
  baseAim: number,
  baseReaction: number,
  loadoutHealthDelta = 0,
  unusualEffect?: string,
): BotCard {
  const multiplier = rarity === 'LEGENDARY' ? 2 : rarity === 'ELITE' ? 1.5 : 1;
  const stacks = rarity === 'LEGENDARY' ? 3 : rarity === 'ELITE' ? 2 : 1;
  return {
    id,
    name,
    className,
    classKey,
    loadoutKey,
    rarity,
    weapons,
    cosmetic,
    unusualEffect,
    innates: innates.map((innate) => `${innate} +${innatePercentPerStack[innate] * stacks}%`),
    stars: stacks,
    health: Math.round(health * multiplier + loadoutHealthDelta),
    giantHealth: Math.round(health * multiplier * 2 + loadoutHealthDelta),
    loadoutHealthDelta,
    // AIM is a bounded 0–100 accuracy preview, not an uncapped RPG rating.
    aim: Math.min(100, Math.round(baseAim * multiplier)),
    reaction: `${(baseReaction / multiplier).toFixed(2)}s`,
    model: robotIcons[className],
  };
}
