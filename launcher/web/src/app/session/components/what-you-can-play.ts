import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { appLink } from '@app/routing/app-routes';
import { LauncherStore } from '@app/server/launcher-store';
import { Panel } from '@app/ui/panel';
import { mercenaries } from '@app/ui/tf2-art';

/**
 * What the run has handed this slot, at a glance: which mercenaries you may
 * play, how many weapon slots and missions you hold, and the last thing that
 * arrived. The whole list is a click away on the Unlocks screen.
 */
@Component({
  selector: 'app-what-you-can-play',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [Panel, RouterLink],
  template: `
    <app-panel heading="What you can play">
      <a panelAside [routerLink]="unlocks">Full list &rarr;</a>

      <div>
        <div class="caption">Classes &middot; {{ classesLine() }}</div>
        <div class="classes">
          @for (mercenary of classes(); track mercenary.name) {
            <div
              class="mercenary"
              [class.held]="mercenary.held"
              [attr.title]="
                mercenary.held
                  ? mercenary.name + ' is unlocked'
                  : mercenary.name + ' is still locked'
              "
            >
              {{ mercenary.name }}
            </div>
          }
        </div>
      </div>

      <div class="counts">
        <div class="count">
          <div class="caption">Weapon slots</div>
          <div class="figure">{{ slotsLine() }}</div>
        </div>
        <div class="count">
          <div class="caption">Missions</div>
          <div class="figure">{{ missionsLine() }}</div>
        </div>
      </div>

      <div class="latest">
        Latest: <span>{{ latest() }}</span>
      </div>
    </app-panel>
  `,
  styleUrl: './what-you-can-play.scss',
})
export class WhatYouCanPlay {
  private readonly store = inject(LauncherStore);

  readonly unlocks = appLink.unlocks();

  private readonly held = computed(() => {
    const byKind = new Map<string, string[]>();
    for (const unlock of this.store.unlocks()) {
      byKind.set(unlock.kind, [...(byKind.get(unlock.kind) ?? []), unlock.name]);
    }
    return byKind;
  });

  readonly classes = computed(() => {
    const unlocked = new Set(this.held().get('Class') ?? []);
    return mercenaries.map((name) => ({ name, held: unlocked.has(name) }));
  });

  readonly classesLine = computed(
    () => `${this.classes().filter((one) => one.held).length} of ${mercenaries.length}`,
  );

  readonly slotsLine = computed(() => count(this.held().get('Weapon slot')));

  readonly missionsLine = computed(() => {
    const missions = this.store.missions();
    if (missions.length === 0) {
      return count(this.held().get('Mission'));
    }
    return `${missions.filter((mission) => mission.unlocked).length} of ${missions.length}`;
  });

  readonly latest = computed(() => {
    const unlocks = this.store.unlocks();
    const last = unlocks[unlocks.length - 1];
    return last === undefined ? 'nothing yet' : `${last.name} (${last.kind.toLowerCase()})`;
  });
}

function count(names: string[] | undefined): string {
  const held = names?.length ?? 0;
  return held === 1 ? '1 held' : `${held} held`;
}
