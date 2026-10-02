import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Subject, exhaustMap, map, of, tap } from 'rxjs';

import { LauncherCommands } from '@app/server/launcher-commands';
import { LauncherStore } from '@app/server/launcher-store';
import { orRefusal } from '@app/transport/refusal';
import { Button } from '@app/ui/button';
import { Notice } from '@app/ui/notice';

/**
 * The installed TF2 build and the button that updates it. Steam only tells a
 * running server that it is behind, so "a newer build is out" is the launcher
 * passing that on. The button waits while anything else is installing: the
 * launcher refuses a second install or update anyway.
 */
@Component({
  selector: 'app-game-update',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [Button, Notice],
  template: `
    <div class="row">
      <div class="words">
        <span class="label">Team Fortress 2</span>
        <p>{{ buildLine() }}</p>
        <p>Stops the server if it is running, updates TF2, then starts it again.</p>
      </div>
      <app-button tone="secondary" [disabled]="busy()" (press)="update.next()">
        Update TF2
      </app-button>
    </div>
    @if (available()) {
      <app-notice tone="warn">
        Steam has a newer TF2 build. Players on the new build cannot join until you update.
      </app-notice>
    }
    @if (busy() && activity()) {
      <app-notice tone="info">
        <span class="activity"><span class="activity-dot"></span>{{ activity() }}</span>
      </app-notice>
    }
    @if (refusal()) {
      <app-notice tone="bad">{{ refusal() }}</app-notice>
    } @else if (failure()) {
      <app-notice tone="bad">The TF2 update failed: {{ failure() }}</app-notice>
    }
  `,
  styles: `
    :host {
      display: flex;
      flex-direction: column;
      gap: var(--gap-sm);
      padding: var(--gap) 0;
    }

    .row {
      display: flex;
      align-items: start;
      justify-content: space-between;
      gap: var(--gap-lg);
    }

    .label {
      font-weight: 600;
    }

    p {
      margin: 2px 0 0;
      font-size: var(--text-sm);
      color: var(--text-dim);
    }

    .activity {
      display: inline-flex;
      align-items: center;
      gap: var(--gap-sm);
    }

    .activity-dot {
      width: 9px;
      height: 9px;
      flex: none;
      border-radius: 50%;
      background: currentcolor;
      animation: pulse-dot 1.2s ease-in-out infinite;
    }
  `,
})
export class GameUpdate {
  private readonly store = inject(LauncherStore);
  private readonly commands = inject(LauncherCommands);

  readonly busy = this.store.busy;
  readonly activity = this.store.activity;
  readonly available = this.store.gameUpdateAvailable;
  readonly failure = this.store.gameUpdateError;
  readonly buildLine = computed(() => {
    const build = this.store.gameBuild();
    return build === '' ? 'Installed build: unknown' : `Installed build: ${build}`;
  });

  /** What the launcher refused the last press for: an update or install
      already running, or a server Docker Compose runs. */
  readonly refusal = signal('');

  readonly update = new Subject<void>();

  constructor() {
    this.update
      .pipe(
        tap(() => this.refusal.set('')),
        exhaustMap(() =>
          this.commands.updateGame().pipe(
            map(() => ''),
            orRefusal((refusal) => of(refusal)),
          ),
        ),
        tap((refusal) => this.refusal.set(refusal)),
        takeUntilDestroyed(),
      )
      .subscribe();
  }
}
