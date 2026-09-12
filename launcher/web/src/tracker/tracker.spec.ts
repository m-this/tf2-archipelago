import { provideHttpClient } from '@angular/common/http';
import { TestBed } from '@angular/core/testing';

import { buffsFor, buildView } from './model';
import { sampleSource } from './sample';
import { parseSource } from './source-address';
import { Tracker } from './tracker';
import { datapackageChecksum } from './tracker-source';

describe('tracker source', () => {
  it('accepts room URLs, tracker URLs and compact IDs', () => {
    expect(parseSource('https://archipelago.gg/room/room_123')).toEqual({
      host: 'https://archipelago.gg',
      kind: 'room',
      id: 'room_123',
    });
    expect(parseSource('https://archipelago.gg/tracker/track-1/0/7')).toEqual({
      host: 'https://archipelago.gg',
      kind: 'tracker',
      id: 'track-1',
      player: 7,
    });
    expect(parseSource('compact_id')).toEqual({
      host: 'https://archipelago.gg',
      kind: 'unknown',
      id: 'compact_id',
    });
  });

  it('rejects strings that cannot be Archipelago IDs', () => {
    expect(() => parseSource('not an id')).toThrow('Archipelago ID');
  });

  it('reads both public tracker datapackage shapes', () => {
    const game = 'Team Fortress 2 Mann vs Machine';
    expect(datapackageChecksum({ [game]: { checksum: 'f3f00d5', version: 0 } })).toBe('f3f00d5');
    expect(datapackageChecksum({ games: { [game]: { checksum: 'f3f00d5' } } })).toBe('f3f00d5');
  });
});

describe('tracker view', () => {
  const source = sampleSource('https://archipelago.gg');
  const view = buildView(source, 1);

  it('combines starting state and received inventory', () => {
    expect(view.playerName).toBe('RED Team Server');
    expect(view.slotCount).toBe(2);
    expect(view.classCount).toBe(5);
    expect(view.grapplingHook).toBe(true);
    expect(view.missions[0]?.locked).toBe(false);
    expect(view.missions.at(-1)?.locked).toBe(true);
  });

  it('groups compatible buffs by weapon', () => {
    const soldier = buffsFor('Soldier', view.owned, source.buffWeapons);
    expect(soldier).toHaveLength(2);
    expect(soldier[0]?.weapon).toBe('Air Strike');
    expect(soldier[0]?.total).toBe(6);
    expect(soldier[0]?.effects).toHaveLength(3);
  });

  it('shows the shared melee pool for every class', () => {
    for (const className of [
      'Scout',
      'Soldier',
      'Pyro',
      'Demoman',
      'Heavy',
      'Engineer',
      'Medic',
      'Sniper',
      'Spy',
    ] as const) {
      const melee = buffsFor(className, view.owned, source.buffWeapons).find(
        (weapon) => weapon.weapon === 'All-Class Melee',
      );
      expect(melee?.total).toBe(3);
      expect(melee?.icon).toContain('cb6c1fb553e24bdf3d885d5c07a9a1cc.png');
    }
  });

  it('presents legacy Saxxy rewards as the shared melee pool', () => {
    const owned = new Map([['Weapon Buff: Saxxy — +10% damage', 2]]);
    const weapons = new Map([
      [
        'Saxxy',
        {
          name: 'Saxxy',
          classes: ['Scout'],
          icon: 'assets/tf2/items/legacy-saxxy.png',
        },
      ],
    ]);
    expect(buffsFor('Scout', owned, weapons)).toEqual([
      {
        weapon: 'All-Class Melee',
        icon: 'assets/tf2/items/legacy-saxxy.png',
        aliases: [],
        effects: [{ effect: '+10% damage', count: 2 }],
        total: 2,
      },
    ]);
  });

  it('folds equivalent reskin rewards into their base weapon cards', () => {
    const scout = buffsFor('Scout', view.owned, source.buffWeapons);
    const basher = scout.find((weapon) => weapon.weapon === 'Boston Basher');
    expect(basher?.total).toBe(2);
    expect(basher?.aliases).toContain('Three-Rune Blade');

    const demoman = buffsFor('Demoman', view.owned, source.buffWeapons);
    const booties = demoman.find((weapon) => weapon.weapon === "Ali Baba's Wee Booties");
    expect(booties?.total).toBe(1);
    expect(booties?.aliases).toContain('Bootlegger');
  });
});

describe('tracker screen', () => {
  it('renders the sample through the shared Angular components', () => {
    TestBed.configureTestingModule({ imports: [Tracker], providers: [provideHttpClient()] });
    const fixture = TestBed.createComponent(Tracker);
    const element = fixture.nativeElement as HTMLElement;
    fixture.detectChanges();

    const sample = [...element.querySelectorAll<HTMLButtonElement>('button')].find((button) =>
      button.textContent?.includes('View sample run'),
    );
    sample?.click();
    TestBed.tick();

    expect(element.querySelector('app-logo')).not.toBeNull();
    expect(element.querySelectorAll('app-panel').length).toBeGreaterThan(3);
    expect(element.textContent).toContain('RED Team Server');
    expect(element.textContent).toContain('Grappling Hook');
    expect(element.textContent).toContain('All-Class Melee');
    expect(element.querySelector('.equipment-state')?.textContent?.trim()).toBe('Unlocked');
    expect(element.querySelector('.tier')).not.toBeNull();
    expect(element.querySelector('app-badge span.accent')).not.toBeNull();
  });
});
