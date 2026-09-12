import { Objective, TrackerSource, Weapon } from './types';

const game = 'Team Fortress 2 Mann vs Machine';

export function sampleSource(host: string): TrackerSource {
  const catalog = [
    mission(2, 'mvm_decoy_intermediate', "Doe's Doom", 'intermediate', 7),
    mission(15, 'mvm_mannworks_ironman', 'Mean Machines', 'advanced', 7),
    mission(24, 'mvm_mannhattan_advanced1', 'Empire Escalation', 'advanced', 5),
    mission(31, 'mvm_bigrock_advanced1', 'Broken Parts', 'expert', 7),
  ];
  const checked = [
    ...catalog[0].locations.map((location) => location.id),
    ...catalog[1].locations.slice(0, 5).map((location) => location.id),
    ...catalog[2].locations.slice(0, 2).map((location) => location.id),
  ];
  const itemNames = new Map<number, string>([
    [7_443_001_002, "Mission Ticket: Doe's Doom"],
    [7_443_001_015, 'Mission Ticket: Mean Machines'],
    [7_443_001_024, 'Mission Ticket: Empire Escalation'],
    [7_443_002_001, 'Class: Scout'],
    [7_443_002_002, 'Class: Soldier'],
    [7_443_002_005, 'Class: Heavy'],
    [7_443_002_006, 'Class: Engineer'],
    [7_443_002_007, 'Class: Medic'],
    [7_443_003_000, 'Progressive Weapon Slot'],
    [7_443_005_001, 'Weapon Buff: Air Strike — +10% firing speed'],
    [7_443_005_002, 'Weapon Buff: Minigun — +10% damage'],
    [7_443_005_003, 'Weapon Buff: Mad Milk — +15% slow duration'],
    [7_443_005_004, "Weapon Buff: Crusader's Crossbow — +25% healing"],
    [7_443_005_005, 'Weapon Buff: Grenade Launcher — +20% explosion radius'],
    [7_443_005_006, 'Weapon Buff: Air Strike — +10% damage'],
    [7_443_005_007, 'Weapon Buff: Air Strike — +15% reload speed'],
    [7_443_005_008, 'Weapon Buff: Minigun — +15 health on kill'],
    [7_443_007_001, 'Grappling Hook'],
    [7_443_004_001, 'Cash Bundle'],
  ]);
  const receivedIds = [
    7_443_001_002, 7_443_001_015, 7_443_001_024, 7_443_002_001, 7_443_002_002, 7_443_002_005,
    7_443_002_006, 7_443_002_007, 7_443_003_000, 7_443_005_001, 7_443_005_001, 7_443_005_002,
    7_443_005_002, 7_443_005_002, 7_443_005_002, 7_443_005_003, 7_443_005_004, 7_443_005_004,
    7_443_005_004, 7_443_005_005, 7_443_005_005, 7_443_005_005, 7_443_005_005, 7_443_005_005,
    7_443_005_005, 7_443_005_006, 7_443_005_006, 7_443_005_006, 7_443_005_007, 7_443_005_008,
    7_443_005_008, 7_443_007_001, 7_443_004_001, 7_443_004_001, 7_443_004_001,
  ];
  const weapons = [
    weapon('Air Strike', ['Soldier'], 'f87faf790afc0d04056479f1566f09f1.png'),
    weapon('Minigun', ['Heavy'], 'a74b4d01f56a2f93c537b4d5e8b73715.png'),
    weapon('Mad Milk', ['Scout'], '56b94729a6e2f3fa8f0992d6fc8c1895.png'),
    weapon("Crusader's Crossbow", ['Medic'], '9c40cbf4c4717626e8e02f0fcfbea19d.png'),
    weapon('Grenade Launcher', ['Demoman'], 'e62935cf06f65a239f52541c2fd9472f.png'),
  ];
  return {
    mode: 'demo',
    host,
    kind: 'tracker',
    id: 'sample',
    tracker: 'sample',
    preferredPlayer: 1,
    players: [{ player: 1, game }],
    names: new Map([[1, 'RED Team Server']]),
    catalog,
    buffWeapons: new Map(weapons.map((entry) => [entry.name, entry])),
    itemNames,
    slots: [],
    totals: [],
    received: [{ player: 1, items: receivedIds.map((id) => [id]) }],
    checks: [{ player: 1, locations: checked }],
    demoSlotData: {
      missions: catalog.map((entry) => entry.pop_file),
      start_mission: catalog[0].pop_file,
      goal: 'final_boss',
      goal_mission: catalog[3].pop_file,
      mission_ticket_importance: 'progression',
      tracker: { starting_items: [] },
    },
  };
}

function mission(id: number, pop: string, name: string, difficulty: string, waves: number) {
  const locations: Objective[] = Array.from({ length: waves }, (_, index) => ({
    id: 7_442_000_000 + id * 100 + index + 1,
    name: `${name} Wave ${index + 1}`,
    kind: 'wave_cleared',
    wave: index + 1,
  }));
  locations.push(
    { id: 7_442_000_000 + id * 100 + 90, name: `${name} Tank`, kind: 'tank_destroyed' },
    { id: 7_442_000_000 + id * 100 + 91, name: `${name} Giant`, kind: 'giant_killed' },
    { id: 7_442_000_000 + id * 100 + 99, name: `${name} Complete`, kind: 'mission_cleared' },
  );
  return { pop_file: pop, name, difficulty, locations };
}

function weapon(name: string, classes: readonly string[], filename: string): Weapon {
  return {
    name,
    classes,
    icon: `assets/tf2/items/${filename}`,
  };
}
