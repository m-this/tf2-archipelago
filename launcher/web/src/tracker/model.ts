import { Mercenary, mercenaries } from '@app/ui/tf2-art';

import {
  BuffEffect,
  BuffWeaponView,
  Mission,
  MissionView,
  Objective,
  SlotData,
  TrackerSource,
  TrackerView,
  Weapon,
} from './types';
import { itemKind, itemTone } from './presentation';

const startSlots: Readonly<Record<string, number>> = {
  normal: 1,
  intermediate: 1,
  advanced: 2,
  expert: 3,
  haunted: 3,
};

export function buildView(source: TrackerSource, player: number): TrackerView {
  const slotData = slotDataFor(source, player);
  const missions = activeMissions(source, slotData);
  const checked = checkedFor(source, player);
  const owned = ownedNames(source, player, slotData, missions);
  const missionViews = missions.map((mission) => {
    const clear = missionClear(mission);
    const complete = clear !== undefined && checked.has(clear.id);
    const ticket = owned.has(`Mission Ticket: ${mission.name}`);
    return {
      ...mission,
      complete,
      locked: slotData.mission_ticket_importance !== 'useful' && !ticket,
      finalBoss: slotData.goal === 'final_boss' && mission.pop_file === slotData.goal_mission,
      progress: mission.locations.filter((location) => checked.has(location.id)).length,
    };
  });
  const start = missions.find((mission) => mission.pop_file === slotData.start_mission);
  const inferred = startSlots[start?.difficulty ?? ''] ?? 0;
  const slotCount = Math.min(3, (owned.get('Progressive Weapon Slot') ?? 0) + inferred);
  const completed = missionViews.filter((mission) => mission.complete).length;
  const total =
    source.mode === 'demo'
      ? missions.reduce((sum, mission) => sum + mission.locations.length, 0)
      : (rowFor(source.totals, player)?.total_locations ??
        missions.reduce((sum, mission) => sum + mission.locations.length, 0));
  const goal = describeGoal(slotData, missionViews, completed);

  return {
    player,
    playerName:
      source.names.get(player) ??
      (source.mode === 'demo' ? 'RED Team Server' : `TF2 slot ${player}`),
    missions: missionViews,
    checked,
    owned,
    classes: mercenaries.map((name) => ({ name, unlocked: owned.has(`Class: ${name}`) })),
    classCount: mercenaries.filter((name) => owned.has(`Class: ${name}`)).length,
    unlocks: [...owned.entries()]
      .map(([name, count]) => {
        const kind = itemKind(name);
        return { kind, name: name.replace(/^Weapon Buff: /, ''), count, tone: itemTone(kind) };
      })
      .sort((left, right) => left.name.localeCompare(right.name)),
    slotCount,
    grapplingHook: owned.has('Grappling Hook'),
    completed,
    total,
    ...goal,
    oldInventory: source.mode === 'live' && slotData.tracker?.starting_items === undefined,
  };
}

export function buffsFor(
  className: Mercenary,
  owned: ReadonlyMap<string, number>,
  weapons: ReadonlyMap<string, Weapon>,
): readonly BuffWeaponView[] {
  const grouped = new Map<string, { effects: BuffEffect[]; total: number }>();
  for (const [name, count] of owned) {
    if (!name.startsWith('Weapon Buff:')) continue;
    const parts = buffParts(name);
    if (!(weapons.get(parts.weapon)?.classes ?? []).includes(className)) continue;
    const entry = grouped.get(parts.weapon) ?? { effects: [], total: 0 };
    entry.effects.push({ effect: parts.effect, count });
    entry.total += count;
    grouped.set(parts.weapon, entry);
  }
  return [...grouped.entries()]
    .map(([weapon, value]) => ({
      weapon,
      icon: weapons.get(weapon)?.icon,
      effects: value.effects.toSorted(
        (left, right) => right.count - left.count || left.effect.localeCompare(right.effect),
      ),
      total: value.total,
    }))
    .toSorted((left, right) => right.total - left.total || left.weapon.localeCompare(right.weapon));
}

function activeMissions(source: TrackerSource, slotData: SlotData): readonly Mission[] {
  if (source.mode === 'demo') return source.catalog;
  if (slotData.tracker?.missions !== undefined) return slotData.tracker.missions;
  const byPop = new Map(source.catalog.map((mission) => [mission.pop_file, mission]));
  return (slotData.missions ?? [])
    .map((popFile) => byPop.get(popFile))
    .filter((mission): mission is Mission => mission !== undefined);
}

function checkedFor(source: TrackerSource, player: number): ReadonlySet<number> {
  return new Set(rowFor(source.checks, player)?.locations ?? []);
}

function ownedNames(
  source: TrackerSource,
  player: number,
  slotData: SlotData,
  missions: readonly Mission[],
): ReadonlyMap<string, number> {
  const names = new Map<string, number>();
  for (const item of rowFor(source.received, player)?.items ?? []) {
    const id = Number('item' in item ? item.item : item[0]);
    const name = source.itemNames.get(id) ?? `Unknown item ${id}`;
    names.set(name, (names.get(name) ?? 0) + 1);
  }
  for (const item of slotData.tracker?.starting_items ?? slotData.starting_items ?? []) {
    const name = typeof item === 'number' ? source.itemNames.get(item) : item;
    if (name !== undefined) names.set(name, (names.get(name) ?? 0) + 1);
  }
  const start = missions.find((mission) => mission.pop_file === slotData.start_mission);
  if (start !== undefined) {
    const ticket = `Mission Ticket: ${start.name}`;
    names.set(ticket, Math.max(1, names.get(ticket) ?? 0));
  }
  return names;
}

function slotDataFor(source: TrackerSource, player: number): SlotData {
  return source.demoSlotData ?? rowFor(source.slots, player)?.slot_data ?? {};
}

function rowFor<T extends { readonly player: number }>(
  rows: readonly T[],
  player: number,
): T | undefined {
  return rows.find((row) => Number(row.player) === player);
}

function missionClear(mission: Mission): Objective | undefined {
  return mission.locations.find((location) => location.kind === 'mission_cleared');
}

function describeGoal(
  slotData: SlotData,
  missions: readonly MissionView[],
  completed: number,
): Pick<TrackerView, 'goalTitle' | 'goalDescription' | 'goalPercent'> {
  if (slotData.goal === 'missionsanity') {
    const target = Number(slotData.missionsanity_target) || missions.length || 1;
    const progress = Math.min(completed, target);
    return {
      goalTitle: 'Missionsanity',
      goalDescription: `Clear ${target} missions in any order. ${progress} completed so far.`,
      goalPercent: Math.round((progress / target) * 100),
    };
  }
  const goal = missions.find((mission) => mission.pop_file === slotData.goal_mission);
  const progress = goal?.complete === true ? 1 : 0;
  return {
    goalTitle: goal === undefined ? 'Final Boss' : `Final Boss: ${goal.name}`,
    goalDescription:
      goal === undefined
        ? 'Clear the marked final mission to finish the run.'
        : `Clear ${goal.name}, the marked final mission, to finish the run.`,
    goalPercent: progress * 100,
  };
}

function buffParts(itemName: string): { weapon: string; effect: string } {
  const label = itemName.replace(/^Weapon Buff: /, '');
  const separator = label.indexOf(' — ');
  return separator < 0
    ? { weapon: label, effect: 'Weapon upgrade unlocked' }
    : { weapon: label.slice(0, separator), effect: label.slice(separator + 3) };
}
