import { ChangeDetectionStrategy, Component, computed, input, output, signal } from '@angular/core';

import { Badge } from '@app/ui/badge';
import { Button } from '@app/ui/button';
import { Tier } from '@app/ui/tier';

export type MissionTone = 'good' | 'info' | 'accent' | 'neutral' | 'warn';

/**
 * One mission as either screen draws it. Play and Settings show the same
 * mission with a different question beside it: whether it is in the pool, and
 * whether it can be played now. Both are here, and each screen turns on the
 * column that asks its question.
 */
export interface MissionRow {
  readonly key: string;
  readonly name: string;
  readonly map: string;
  readonly tier: string;
  readonly source: string;
  readonly waves: number;
  /** A word under the name: the loadout the mission wants, where it wants one. */
  readonly loadout: string;
  /** What the screen says about the mission: unlocked, played, Ready, a reason. */
  readonly status: string;
  readonly tone: MissionTone;
  /** Drawn as a pill rather than coloured words: a reason is one, a state is not. */
  readonly badge: boolean;
  readonly mods: string;
  /** The pool tick, where the table has one. */
  readonly on: boolean;
  /** The pool tick cannot change until its compatibility requirement is met. */
  readonly disabled: boolean;
  /** Dimmed: out of the pool, or not unlocked. */
  readonly dim: boolean;
  /** The row the server is on now. */
  readonly playing: boolean;
  /** The Play button, where the table has one: empty means none on this row. */
  readonly play: string;

  /** The waves the picker beside Play offers, empty where it offers none. */
  readonly waveChoices: readonly number[];

  /** The wave the picker starts on: the one the team got to, or the first. */
  readonly waveStart: number;
}

export type MissionColumn = 'on' | 'name' | 'map' | 'tier' | 'source' | 'waves' | 'status' | 'mods';

const tiers: Record<string, number> = { normal: 0, intermediate: 1, advanced: 2, expert: 3 };

/**
 * The missions table, one component for both screens.
 *
 * Sorting is here because it is the same on both: the player asks for the pool
 * together or the unlocked ones first, and a column header answers. What the
 * status column means and what sorts under it is the caller's, given in the
 * row as `order`-free text: the caller sorts status by handing rows in the
 * order it wants and the table keeps that order for ties.
 */
@Component({
  selector: 'app-missions-table',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [Badge, Button, Tier],
  templateUrl: './missions-table.html',
  styleUrl: './missions-table.scss',
})
export class MissionsTable {
  readonly rows = input.required<readonly MissionRow[]>();
  readonly pool = input(false);
  readonly mods = input(false);
  readonly playable = input(false);
  readonly initialSort = input<MissionColumn>('name');

  readonly toggled = output<string>();
  readonly chosen = output<string>();
  readonly resumed = output<{ key: string; wave: number }>();

  readonly sortBy = signal<MissionColumn | undefined>(undefined);
  readonly ascending = signal(true);

  readonly columns = computed<{ key: MissionColumn; label: string }[]>(() => [
    ...(this.pool() ? [{ key: 'on' as const, label: 'In pool' }] : []),
    { key: 'name', label: 'Mission' },
    { key: 'map', label: 'Map' },
    { key: 'tier', label: 'Tier' },
    { key: 'source', label: 'Archive' },
    { key: 'waves', label: 'Waves' },
    { key: 'status', label: this.pool() ? 'Compatibility' : 'State' },
    ...(this.mods() ? [{ key: 'mods' as const, label: 'Server mod' }] : []),
  ]);

  readonly key = computed(() => this.sortBy() ?? this.initialSort());

  readonly sorted = computed(() => {
    const key = this.key();
    const direction = this.ascending() ? 1 : -1;
    return this.rows().toSorted((left, right) => direction * compare(left, right, key));
  });

  sortOn(key: MissionColumn): void {
    if (this.key() === key) {
      this.ascending.set(!this.ascending());
      return;
    }
    this.sortBy.set(key);
    this.ascending.set(true);
  }
}

/**
 * Three kinds of column, and text order is wrong for two of them. The tick and
 * the state are what the player came to sort by, and a tier is a ladder, not a
 * word. Everything else is text, with the name breaking ties so the order is
 * the same twice.
 */
function compare(left: MissionRow, right: MissionRow, key: MissionColumn): number {
  if (key === 'on') {
    return Number(left.on) - Number(right.on) || left.name.localeCompare(right.name);
  }
  if (key === 'waves') {
    return left.waves - right.waves || left.name.localeCompare(right.name);
  }
  if (key === 'tier') {
    return rank(left.tier) - rank(right.tier) || left.name.localeCompare(right.name);
  }
  if (key === 'status') {
    // The caller handed the rows in status order; a stable sort keeps it.
    return 0;
  }
  return left[key].localeCompare(right[key]) || left.name.localeCompare(right.name);
}

function rank(tier: string): number {
  return tiers[tier.toLowerCase()] ?? 9;
}
