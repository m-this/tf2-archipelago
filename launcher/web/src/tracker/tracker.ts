import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  ElementRef,
  OnDestroy,
  afterNextRender,
  computed,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { finalize } from 'rxjs';
import { Badge } from '@app/ui/badge';
import { Button } from '@app/ui/button';
import { EmptyState } from '@app/ui/empty-state';
import { Logo } from '@app/ui/logo';
import { Panel } from '@app/ui/panel';
import { grapplingHookIcon, mercenaryIcons } from '@app/ui/tf2-art';
import { buffsFor, buildView } from './model';
import { hideBrokenImage, initialLocation, objectiveLabel, rememberSource } from './presentation';
import { TrackerSourceClient } from './tracker-source';
import { ClassView, TrackerSource } from './types';
@Component({
  selector: 'app-tracker',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [Badge, Button, EmptyState, Logo, Panel],
  templateUrl: './tracker.html',
})
export class Tracker implements OnDestroy {
  private readonly client = inject(TrackerSourceClient);
  private readonly destroyRef = inject(DestroyRef);
  private readonly roomInput = viewChild<ElementRef<HTMLInputElement>>('roomInput');
  private readonly buffDialog = viewChild.required<ElementRef<HTMLDialogElement>>('buffDialog');
  private timer: ReturnType<typeof setInterval> | undefined;

  readonly input = signal('');
  readonly source = signal<TrackerSource | undefined>(undefined);
  readonly player = signal(0);
  readonly busy = signal(false);
  readonly refreshing = signal(false);
  readonly status = signal('No room loaded');
  readonly live = signal(false);
  readonly message = signal(
    'Room links provide player names. Tracker links and compact IDs work too.',
  );
  readonly error = signal(false);
  readonly selectedClass = signal<ClassView | undefined>(undefined);

  readonly view = computed(() => {
    const source = this.source();
    const player = this.player();
    return source === undefined || player === 0 ? undefined : buildView(source, player);
  });
  readonly players = computed(() => {
    const source = this.source();
    return (source?.players ?? []).map((row) => ({
      id: Number(row.player),
      name: source?.names.get(Number(row.player)) ?? `TF2 slot ${row.player}`,
    }));
  });
  readonly classBuffs = computed(() => {
    const chosen = this.selectedClass();
    const view = this.view();
    const source = this.source();
    return chosen === undefined || view === undefined || source === undefined
      ? []
      : buffsFor(chosen.name, view.owned, source.buffWeapons);
  });
  readonly buffLevels = computed(() =>
    this.classBuffs().reduce((sum, weapon) => sum + weapon.total, 0),
  );

  readonly mercenaryIcons = mercenaryIcons;
  readonly grapplingHookIcon = grapplingHookIcon;
  readonly slots = ['Primary', 'Secondary', 'Melee'] as const;
  readonly powerSegments = Array.from({ length: 10 }, (_, index) => index + 1);
  readonly objectiveLabel = objectiveLabel;
  readonly hideBrokenImage = hideBrokenImage;
  constructor() {
    afterNextRender(() => this.openInitialURL());
  }

  ngOnDestroy(): void {
    this.stopRefreshing();
  }

  load(): void {
    this.busy.set(true);
    this.error.set(false);
    this.message.set('Finding the room and TF2 slot…');
    this.status.set('Connecting…');
    this.client
      .load(this.input())
      .pipe(
        finalize(() => this.busy.set(false)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: (source) => {
          this.show(source);
          rememberSource(source);
          this.scheduleRefresh();
        },
        error: (error: Error) => {
          this.status.set('Could not load room');
          this.live.set(false);
          this.error.set(true);
          this.message.set(`Could not load that tracker: ${error.message}`);
        },
      });
  }

  showSample(): void {
    this.stopRefreshing();
    const source = this.client.sample();
    this.show(source);
    history.replaceState(null, '', `${location.pathname}?demo=1`);
  }

  changeRoom(): void {
    this.stopRefreshing();
    this.source.set(undefined);
    this.player.set(0);
    this.status.set('No room loaded');
    this.live.set(false);
    this.error.set(false);
    this.message.set('Room links provide player names. Tracker links and compact IDs work too.');
    history.replaceState(null, '', location.pathname);
    setTimeout(() => this.roomInput()?.nativeElement.focus());
  }

  refresh(): void {
    const source = this.source();
    if (source?.mode !== 'live' || this.refreshing()) return;
    this.refreshing.set(true);
    this.client
      .refresh(source)
      .pipe(
        finalize(() => this.refreshing.set(false)),
        takeUntilDestroyed(this.destroyRef),
      )
      .subscribe({
        next: (updated) => {
          this.source.set(updated);
          this.status.set('Updated just now');
          this.live.set(true);
        },
        error: () => {
          this.status.set('Refresh failed');
          this.live.set(false);
        },
      });
  }

  choosePlayer(event: Event): void {
    this.player.set(Number((event.target as HTMLSelectElement).value));
  }

  openBuffs(classView: ClassView): void {
    this.selectedClass.set(classView);
    this.buffDialog().nativeElement.showModal();
  }

  closeBuffs(): void {
    this.buffDialog().nativeElement.close();
  }

  private show(source: TrackerSource): void {
    const choices = source.players.map((row) => Number(row.player));
    const preferred = source.preferredPlayer ?? choices[0] ?? 0;
    this.source.set(source);
    this.player.set(choices.includes(preferred) ? preferred : (choices[0] ?? 0));
    this.status.set(source.mode === 'demo' ? 'Sample data' : 'Tracker connected');
    this.live.set(source.mode === 'live');
  }

  private scheduleRefresh(): void {
    this.stopRefreshing();
    this.timer = setInterval(() => this.refresh(), 60_000);
  }

  private stopRefreshing(): void {
    if (this.timer !== undefined) clearInterval(this.timer);
    this.timer = undefined;
  }

  private openInitialURL(): void {
    const initial = initialLocation(location.search);
    if (initial.demo) {
      this.showSample();
      return;
    }
    if (initial.input !== undefined) {
      this.input.set(initial.input);
      this.load();
    }
  }
}
