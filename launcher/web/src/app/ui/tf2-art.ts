/** TF2 artwork shared by the launcher and the public campaign tracker. */
export const mercenaries = [
  'Scout',
  'Soldier',
  'Pyro',
  'Demoman',
  'Heavy',
  'Engineer',
  'Medic',
  'Sniper',
  'Spy',
] as const;

export type Mercenary = (typeof mercenaries)[number];

export const mercenaryIcons: Readonly<Record<Mercenary, string>> = {
  Scout: 'assets/tf2/classes/Class_scoutred.png',
  Soldier: 'assets/tf2/classes/Class_soldierred.png',
  Pyro: 'assets/tf2/classes/Class_pyrored.png',
  Demoman: 'assets/tf2/classes/Class_demored.png',
  Heavy: 'assets/tf2/classes/Class_heavyred.png',
  Engineer: 'assets/tf2/classes/Class_engired.png',
  Medic: 'assets/tf2/classes/Class_medicred.png',
  Sniper: 'assets/tf2/classes/Class_sniperred.png',
  Spy: 'assets/tf2/classes/Class_spyred.png',
};

export const grapplingHookIcon = 'assets/tf2/items/eadba6f0e8dc08e8e734a4454705b006.png';
