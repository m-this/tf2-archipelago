import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';

import { LauncherStore } from '@app/server/launcher-store';
import { Chip } from '@app/ui/chip';
import { EmptyState } from '@app/ui/empty-state';
import { Panel } from '@app/ui/panel';

/**
 * The filters, fixed rather than read off the data.
 *
 * The kinds are a closed set the bridge names, and a chip that appears only
 * once something of that kind arrives is a chip that moves the other chips
 * along under the player's finger. "All" is first and is where the screen
 * starts.
 */
const kinds = [
  { key: '', label: 'All' },
  { key: 'Class', label: 'Classes' },
  { key: 'Weapon slot', label: 'Weapon slots' },
  { key: 'Mission', label: 'Missions' },
  { key: 'Weapon buff', label: 'Weapon buffs' },
  { key: 'Server lever', label: 'Server levers' },
] as const;

/**
 * Everything the run has handed this slot. The bridge serves it by key, the
 * launcher turns the keys into the names a person uses, and this only groups
 * and counts them.
 */
@Component({
  selector: 'app-unlocks-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [Chip, EmptyState, Panel],
  templateUrl: './unlocks-page.html',
  styleUrl: './unlocks-page.scss',
})
export class UnlocksPage {
  private readonly store = inject(LauncherStore);

  readonly kinds = kinds;
  readonly chosen = signal('');

  readonly all = computed(() => this.store.unlocks());

  readonly summary = computed(() => {
    const held = this.all().length;
    return held === 1 ? '1 unlock so far.' : `${held} unlocks so far.`;
  });

  readonly rows = computed(() => {
    const kind = this.chosen();
    return this.all().filter((unlock) => kind === '' || unlock.kind === kind);
  });

  /** A buff held twice is the same buff at a higher level, which is what the
      column says. Anything held once has no level to report. */
  level(level: number): string {
    return level > 1 ? `×${level}` : '-';
  }
}
