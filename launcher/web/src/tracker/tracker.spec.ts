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
    expect(soldier).toHaveLength(1);
    expect(soldier[0]?.weapon).toBe('Air Strike');
    expect(soldier[0]?.total).toBe(6);
    expect(soldier[0]?.effects).toHaveLength(3);
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
    expect(element.querySelector('.equipment-state')?.textContent?.trim()).toBe('Unlocked');
    expect(element.querySelector('.tier')).not.toBeNull();
    expect(element.querySelector('app-badge span.accent')).not.toBeNull();
  });
});
