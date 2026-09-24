import { Mercenary, robotIcons } from '@app/ui/tf2-art';
import { CardIdentity, CardTier, cardIdentities, cardInnates } from './cards.generated';

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
  readonly model: string;
}

/* Who a card is, what it carries and which innates it has come from the
 * launcher's Go catalogue through cards.generated.ts. What is left here is how a
 * card looks: the names of the items its loadout and cosmetic ids stand for. A
 * card the Go side has and this table does not is refused at load, rather than
 * drawn with blanks. */
interface Look {
  readonly weapons: readonly [string, string, string];
  readonly cosmetic: string;
  readonly unusualEffect?: string;
  // An equipped item that changes max health, as Big Earner's -25 does.
  readonly healthDelta?: number;
}

const looks: Readonly<Record<string, Look>> = {
  'stock-scout': { weapons: ['Scattergun', 'Pistol', 'Bat'], cosmetic: 'Ghastly Gibus' },
  'stock-soldier': { weapons: ['Rocket Launcher', 'Shotgun', 'Shovel'], cosmetic: 'Ghastly Gibus' },
  'stock-pyro': { weapons: ['Flame Thrower', 'Shotgun', 'Fire Axe'], cosmetic: 'Ghastly Gibus' },
  'stock-demoman': {
    weapons: ['Grenade Launcher', 'Stickybomb Launcher', 'Bottle'],
    cosmetic: 'Ghastly Gibus',
  },
  'stock-heavy': { weapons: ['Minigun', 'Shotgun', 'Fists'], cosmetic: 'Ghastly Gibus' },
  'stock-engineer': { weapons: ['Shotgun', 'Pistol', 'Wrench'], cosmetic: 'Ghastly Gibus' },
  'stock-medic': { weapons: ['Syringe Gun', 'Medi Gun', 'Bonesaw'], cosmetic: 'Ghastly Gibus' },
  'stock-sniper': { weapons: ['Sniper Rifle', 'SMG', 'Kukri'], cosmetic: 'Ghastly Gibus' },
  'stock-spy': { weapons: ['Revolver', 'Sapper', 'Knife'], cosmetic: 'Ghastly Gibus' },
  'credit-to-team': {
    weapons: ['Soda Popper', 'Mad Milk', 'Fan O’War'],
    cosmetic: 'Baseball Bill’s Sports Shine',
  },
  'screamin-eagles': {
    weapons: ['Beggar’s Bazooka', 'Buff Banner', 'Escape Plan'],
    cosmetic: 'Team Captain',
  },
  ivan: { weapons: ['Brass Beast', 'Family Business', 'Fists of Steel'], cosmetic: 'Heavy Do-rag' },
  'herr-doktor': {
    weapons: ['Crusader’s Crossbow', 'Kritzkrieg', 'Übersaw'],
    cosmetic: 'Blighted Beak',
    unusualEffect: 'Burning Flames',
  },
  chell: { weapons: ['Rescue Ranger', 'Wrangler', 'Jag'], cosmetic: 'Prairie Heel Biters' },
  mentlegen: {
    weapons: ['Diamondback', 'Red-Tape Recorder', 'Big Earner'],
    cosmetic: 'Fancy Fedora',
    unusualEffect: 'Scorching Flames',
    healthDelta: -25,
  },
};

const classes: Readonly<Record<string, { readonly name: Mercenary; readonly health: number }>> = {
  scout: { name: 'Scout', health: 125 },
  soldier: { name: 'Soldier', health: 200 },
  pyro: { name: 'Pyro', health: 175 },
  demoman: { name: 'Demoman', health: 175 },
  heavyweapons: { name: 'Heavy', health: 300 },
  engineer: { name: 'Engineer', health: 125 },
  medic: { name: 'Medic', health: 150 },
  sniper: { name: 'Sniper', health: 125 },
  spy: { name: 'Spy', health: 125 },
};

const stars: Readonly<Record<CardTier, number>> = { common: 1, elite: 2, legendary: 3 };
// The plugin's health bonus per tier: natural class health times this.
const healthScale: Readonly<Record<CardTier, number>> = { common: 1, elite: 1.5, legendary: 2 };

export const botCards: readonly BotCard[] = cardIdentities.map((identity) =>
  build(identity, identity.tier),
);

// AP item names carry the seed's tier roll. Rebuild the card at that tier so
// the tracker and the admin page show what the run actually received.
export function rolledCard(base: BotCard, rarity: BotCard['rarity']): BotCard {
  const identity = cardIdentities.find((entry) => entry.id === base.id);
  return identity ? build(identity, rarity.toLowerCase() as CardTier) : base;
}

export function cardById(id: string): BotCard | undefined {
  return botCards.find((entry) => entry.id === id);
}

function build(identity: CardIdentity, tier: CardTier): BotCard {
  const look = looks[identity.id];
  const kind = classes[identity.classKey];
  if (look === undefined || kind === undefined) {
    throw new Error(`bot card ${identity.id} has no look or no class in catalog.ts`);
  }
  const health = kind.health * healthScale[tier];
  const delta = look.healthDelta ?? 0;
  return {
    id: identity.id,
    name: identity.name,
    className: kind.name,
    classKey: identity.classKey,
    loadoutKey: identity.loadoutKey,
    rarity: tier.toUpperCase() as BotCard['rarity'],
    stars: stars[tier],
    weapons: look.weapons,
    cosmetic: look.cosmetic,
    unusualEffect: tier === 'legendary' ? (look.unusualEffect ?? 'Burning Flames') : undefined,
    innates: cardInnates[identity.id]?.[tier] ?? [],
    health: Math.round(health + delta),
    giantHealth: Math.round(health * 2 + delta),
    model: robotIcons[kind.name],
  };
}
