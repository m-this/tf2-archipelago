import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Subject, exhaustMap, tap } from 'rxjs';

import { BotLineup } from '@app/bots/components/bot-lineup';
import { BotCardDeck } from '@app/bots/components/bot-card-deck';
import { ClassTable } from '@app/bots/components/class-table';
import { SettingsStore } from '@app/settings/settings-store';
import { LauncherStore } from '@app/server/launcher-store';
import { Button } from '@app/ui/button';
import { Notice } from '@app/ui/notice';
import { Panel } from '@app/ui/panel';

/**
 * The Bot Switcher: who holds RED's seats, and which classes the mod may draw
 * for the ones nobody named. The same two editors the settings draw on the
 * Bots page, because a change here is a change there: both write the draft.
 *
 * Apply is what writes it. The seats are edited in the settings draft, and a
 * draft is nothing until it is saved: a whole lineup was built here once and
 * was gone when the tab was opened again, because nothing on this page said
 * so or offered to keep it.
 */
@Component({
  selector: 'app-bots-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [BotCardDeck, BotLineup, Button, ClassTable, Notice, Panel],
  template: `
    <app-panel heading="Bot cards">
      <app-bot-card-deck />
    </app-panel>
    <app-panel heading="Bot Switcher">
      <app-bot-lineup />
    </app-panel>

    <app-panel heading="Classes the mod may draw">
      <app-class-table />
    </app-panel>

    <footer>
      @if (refusal()) {
        <app-notice tone="bad">{{ refusal() }}</app-notice>
      }
      <span class="dirty" [class.pending]="dirty()">{{ dirtyLabel() }}</span>
      <app-button tone="ghost" [disabled]="!dirty()" (press)="discard.next()">Discard</app-button>
      <app-button tone="go" [disabled]="!dirty()" (press)="apply.next()">Apply</app-button>
    </footer>
  `,
  styleUrl: './bots-page.scss',
})
export class BotsPage {
  private readonly store = inject(SettingsStore);
  private readonly attached = inject(LauncherStore).managedExternally;

  readonly dirty = computed(() => this.store.dirty());
  readonly refusal = signal('');
  readonly applied = signal(false);

  readonly dirtyLabel = computed(() => {
    if (this.attached()) {
      return this.dirty()
        ? 'Not saved yet. Apply updates the live bot team now.'
        : this.applied()
          ? 'Applied to the running server.'
          : 'Edit a bot or loadout, then Apply to update the running server.';
    }
    if (this.dirty()) {
      return 'Not applied yet. Apply writes the lineup and the bots switch on their next respawn.';
    }
    return this.applied() ? 'Applied.' : 'Nothing to apply';
  });

  readonly apply = new Subject<void>();
  readonly discard = new Subject<void>();

  constructor() {
    this.apply
      .pipe(
        exhaustMap(() => this.store.save(false)),
        tap((refusal) => {
          this.refusal.set(refusal);
          this.applied.set(refusal === '');
        }),
        takeUntilDestroyed(),
      )
      .subscribe();
    this.discard
      .pipe(
        exhaustMap(() => this.store.cancel()),
        takeUntilDestroyed(),
      )
      .subscribe();
  }
}
