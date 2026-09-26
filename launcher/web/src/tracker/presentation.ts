import { BadgeTone } from '@app/ui/badge';

import { defaultHost } from './source-address';
import { Objective, TrackerSource, Weapon } from './types';

export function objectiveLabel(objective: Objective): string {
  if (objective.kind === 'wave_cleared') return String(objective.wave);
  if (objective.kind === 'tank_destroyed') return 'Tank';
  if (objective.kind === 'giant_killed') return 'Giant';
  if (objective.kind === 'mission_cleared') return 'Done';
  return '';
}

export function itemKind(name: string): string {
  if (name.startsWith('Weapon Buff:')) return 'Buff';
  if (name.startsWith('Mission Ticket:')) return 'Ticket';
  if (name.startsWith('Class:')) return 'Class';
  if (name.startsWith('Bot:')) return 'Bot';
  if (name.startsWith('Australium Medal:')) return 'Medal';
  if (name.startsWith('Trap:')) return 'Trap';
  if (name.startsWith('Progressive Weapon Slot') || name.endsWith(' Slot')) return 'Loadout';
  if (name === 'Grappling Hook') return 'Server';
  if (name === 'Cash Bundle') return 'Cash';
  return 'Unlock';
}

export function itemTone(kind: string): BadgeTone {
  if (kind === 'Buff' || kind === 'Server') return 'accent';
  if (kind === 'Class' || kind === 'Loadout') return 'info';
  if (kind === 'Bot') return 'accent';
  if (kind === 'Ticket' || kind === 'Cash') return 'good';
  if (kind === 'Medal') return 'warn';
  if (kind === 'Trap') return 'bad';
  return 'neutral';
}

export function hideBrokenImage(event: Event): void {
  (event.target as HTMLImageElement).hidden = true;
}

export function initialLocation(search: string): { demo: boolean; input?: string } {
  const params = new URLSearchParams(search);
  if (params.has('demo')) return { demo: true };
  const kind = params.has('room') ? 'room' : params.has('tracker') ? 'tracker' : undefined;
  if (kind === undefined) return { demo: false };
  const id = params.get(kind);
  if (id === null) return { demo: false };
  return { demo: false, input: `${params.get('host') ?? defaultHost}/${kind}/${id}` };
}

export function rememberSource(source: TrackerSource): void {
  const params = new URLSearchParams();
  params.set(source.kind, source.id);
  if (source.host !== defaultHost) params.set('host', source.host);
  history.replaceState(null, '', `${location.pathname}?${params}`);
}

export function buffParts(itemName: string): { weapon: string; effect: string } {
  const label = displayItemName(itemName).replace(/^Weapon Buff: /, '');
  const separator = label.indexOf(' — ');
  return separator < 0
    ? { weapon: label, effect: 'Weapon upgrade unlocked' }
    : { weapon: label.slice(0, separator), effect: label.slice(separator + 3) };
}

export function displayItemName(name: string): string {
  return name.replace(/^Weapon Buff: Saxxy(?= —|$)/, 'Weapon Buff: All-Class Melee');
}

export function weaponDisplayName(name: string): string {
  return name === 'Saxxy' ? 'All-Class Melee' : name;
}

export function trackerWeapon(
  weapons: ReadonlyMap<string, Weapon>,
  name: string,
): Weapon | undefined {
  const direct =
    weapons.get(name) ?? (name === 'All-Class Melee' ? weapons.get('Saxxy') : undefined);
  if (direct !== undefined) return direct;
  return [...weapons.values()].find((weapon) => weapon.aliases?.includes(name));
}
