import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed, toObservable } from '@angular/core/rxjs-interop';
import { Subject, concatMap, exhaustMap, filter, map, tap } from 'rxjs';

import { LauncherStore } from '@app/server/launcher-store';
import { SettingsStore } from '@app/settings/settings-store';
import { Button } from '@app/ui/button';
import { Field } from '@gen/tf2ap/launcher/v1/form_pb';

/** One seat, as the editor draws it: who plays it and what they carry. */
interface Seat {
  readonly number: number;
  readonly label: string;
  readonly playedBy: Field;
  readonly carries: Field | undefined;
  readonly called: Field | undefined;
}

/**
 * The team: whether RED is filled and to how many, a class and a loadout for
 * each seat, and the teams somebody named and kept.
 *
 * Every one of these is a row form already declares, so this picks them out of
 * the model by id rather than inventing a second way to say who plays seat two.
 * A change goes to the launcher immediately and reaches a running server on the
 * bot's next respawn, with no restart: saveplan calls that a Team change and
 * botlive turns it into console commands.
 */
@Component({
  selector: 'app-bot-lineup',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [Button],
  templateUrl: './bot-lineup.html',
  styleUrl: './bot-lineup.scss',
})
export class BotLineup {
  private readonly settings = inject(SettingsStore);
  private readonly launcher = inject(LauncherStore);

  readonly open = computed(() => this.settings.open());
  readonly notRunning = computed(() => !this.launcher.running());
  readonly feedback = signal('');

  readonly fill = computed(() => this.settings.field('bots.fill'));
  readonly teamSize = computed(() => this.settings.field('bots.team_size'));
  readonly chat = computed(() => this.settings.field('bots.upgrades_chat'));
  readonly savedTeams = computed(() => this.settings.field('bots.team_preset'));
  readonly teamName = computed(() => this.settings.field('bots.team_name'));

  readonly seats = computed<Seat[]>(() => {
    const classes = this.settings.fieldsMatching('bots.seat.');
    const seats: Seat[] = [];
    for (const played of classes.filter((one) => one.id.endsWith('.class'))) {
      const number = seats.length + 1;
      seats.push({
        number,
        label: `Seat ${number}`,
        playedBy: played,
        carries: classes.find((one) => one.id === played.id.replace('.class', '.loadout')),
        called: classes.find((one) => one.id === played.id.replace('.class', '.name')),
      });
    }
    return seats;
  });

  /** How many seats name a class, for the line under the grid. */
  readonly named = computed(
    () => this.seats().filter((seat) => this.value(seat.playedBy) !== '').length,
  );

  /** What Save says, which depends on whether the typed name is already one of
      the saved lineups. */
  readonly saveLabel = computed(() => {
    const typed = this.value(this.teamName()).trim().toLowerCase();
    const known = (this.savedTeams()?.options ?? []).some(
      (option) => option.label.toLowerCase() === typed && typed !== '',
    );
    return known ? 'Update' : 'Save as new';
  });

  /** The name Save would keep the seats under, empty when there is none to
      send: the launcher refuses a team with no name. */
  readonly typedName = computed(() => this.value(this.teamName()).trim());

  readonly fired = new Subject<{ id: string; said: string }>();
  readonly answered = new Subject<{ id: string; value: string; said: string }>();

  constructor() {
    this.answered
      .pipe(
        tap((change) => {
          this.settings.change(change.id, change.value);
          this.feedback.set(change.said);
        }),
        takeUntilDestroyed(),
      )
      .subscribe();

    // The seats are rows of the settings draft, so the draft is opened the
    // moment this finds it closed, and again after a Save closes it.
    toObservable(computed(() => this.launcher.connected() && !this.open()))
      .pipe(
        filter((closed) => closed),
        exhaustMap(() => this.settings.openSettings('Bots')),
        takeUntilDestroyed(),
      )
      .subscribe();

    this.fired
      .pipe(
        concatMap((press) =>
          this.settings
            .dispatch(press.id)
            .pipe(map((refusal) => (refusal === '' ? press.said : refusal))),
        ),
        tap((said) => this.feedback.set(said)),
        takeUntilDestroyed(),
      )
      .subscribe();
  }

  set(field: Field | undefined, value: string, said = ''): void {
    if (field !== undefined) {
      this.answered.next({ id: field.id, value, said });
    }
  }

  setSeatClass(seat: Seat, value: string): void {
    const chosen = seat.playedBy.options.find((option) => option.value === value);
    this.set(seat.playedBy, value, `${seat.label} → ${chosen?.label ?? value}.`);
  }

  setSeatLoadout(seat: Seat, value: string): void {
    this.set(seat.carries, value, `${seat.label} carries ${labelOf(seat.carries, value)}.`);
  }

  setSeatName(seat: Seat, value: string): void {
    const said =
      value === ''
        ? `${seat.label} draws its name from the pool.`
        : `${seat.label} is called ${value}.`;
    this.set(seat.called, value, said);
  }

  loadTeam(value: string): void {
    const field = this.savedTeams();
    if (value !== '') {
      this.set(field, value, `Loaded ${labelOf(field, value)}. Bots switch on their next respawn.`);
    }
  }

  saveTeam(): void {
    const name = this.typedName();
    if (name !== '') {
      this.fired.next({ id: 'bots.save_team', said: `Saved the team as ${name}.` });
    }
  }

  removeTeam(): void {
    const name = this.typedName();
    if (name !== '') {
      this.fired.next({ id: 'bots.remove_team', said: `Removed the team ${name}.` });
    }
  }

  value(field: Field | undefined): string {
    return field === undefined ? '' : this.settings.value(field.id);
  }

  on(field: Field | undefined): boolean {
    return this.value(field) === 'true';
  }
}

function labelOf(field: Field | undefined, value: string): string {
  return field?.options.find((option) => option.value === value)?.label ?? value;
}
