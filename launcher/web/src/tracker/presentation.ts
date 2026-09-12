import { BadgeTone } from '@app/ui/badge';

import { defaultHost } from './source-address';
import { Objective, TrackerSource } from './types';

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
  if (name.startsWith('Australium Medal:')) return 'Medal';
  if (name.startsWith('Trap:')) return 'Trap';
  if (name === 'Progressive Weapon Slot') return 'Loadout';
  if (name === 'Grappling Hook') return 'Server';
  if (name === 'Cash Bundle') return 'Cash';
  return 'Unlock';
}

export function itemTone(kind: string): BadgeTone {
  if (kind === 'Buff' || kind === 'Server') return 'accent';
  if (kind === 'Class' || kind === 'Loadout') return 'info';
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
