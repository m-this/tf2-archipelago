import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  output,
  signal,
} from '@angular/core';

import { SettingsStore } from '@app/settings/settings-store';
import { Button } from '@app/ui/button';
import { EmptyState } from '@app/ui/empty-state';
import { MissionRow, MissionsTable } from '@app/ui/missions-table';
import { SearchBox } from '@app/ui/search-box';

/**
 * The mission pool: twenty-six rows and more with the community packs on, so it
 * is the missions table with a tick per row and a search over it, rather than
 * a row of settings per mission.
 *
 * Compatibility is the launcher's word, not a guess made here. A row that is
 * not Ready stays visible with the launcher's directions, but cannot enter the
 * pool until its archive and server-mod requirements are met.
 */
@Component({
  selector: 'app-mission-table',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [Button, EmptyState, MissionsTable, SearchBox],
  templateUrl: './mission-table.html',
  styleUrl: './mission-table.scss',
})
export class MissionTable {
  readonly store = inject(SettingsStore);

  readonly typed = output<{ id: string; value: string }>();

  readonly filter = signal('');

  private readonly all = computed<MissionRow[]>(() =>
    this.store
      .missionPool()
      .map((pool) => ({ pool, on: this.store.value(pool.field) === 'true' }))
      .toSorted(
        (left, right) =>
          Number(right.pool.compatibility === 'Ready') -
            Number(left.pool.compatibility === 'Ready') ||
          left.pool.name.localeCompare(right.pool.name),
      )
      .map(({ pool, on }) => ({
        key: pool.field,
        name: pool.name,
        map: pool.map,
        tier: pool.tier,
        source: pool.source,
        waves: wavesOf(pool.waves),
        loadout: '',
        status: pool.compatibility,
        tone: pool.compatibility === 'Ready' ? 'good' : 'warn',
        badge: true,
        mods: pool.mods,
        on,
        disabled: pool.disabled,
        dim: !on,
        playing: false,
        play: '',
        waveChoices: [],
        waveStart: 1,
      })),
  );

  readonly rows = computed(() => {
    const needle = this.filter().trim().toLowerCase();
    return this.all().filter((row) => needle === '' || matches(row, needle));
  });

  readonly chosen = computed(() => this.all().filter((row) => row.on).length);

  /** Ticks every row on screen, not every row there is: the search is part of
      what the player meant by "all". */
  setShown(on: boolean): void {
    for (const row of this.rows()) {
      if (row.on !== on) {
        this.typed.emit({ id: row.key, value: on ? 'true' : 'false' });
      }
    }
  }

  flip(key: string): void {
    const row = this.all().find((one) => one.key === key);
    if (row !== undefined) {
      this.typed.emit({ id: key, value: row.on ? 'false' : 'true' });
    }
  }
}

function matches(row: MissionRow, needle: string): boolean {
  return (
    row.name.toLowerCase().includes(needle) ||
    row.map.toLowerCase().includes(needle) ||
    row.source.toLowerCase().includes(needle) ||
    row.status.toLowerCase().includes(needle)
  );
}

/** The wave count out of "1–6". The launcher writes the range; the number at
    the end of it is what a mission is long or short by. */
function wavesOf(waves: string): number {
  return Number(/\d+$/.exec(waves)?.[0] ?? 0);
}
