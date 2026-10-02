import { NgComponentOutlet } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  Type,
  computed,
  inject,
  input,
  linkedSignal,
  signal,
} from '@angular/core';
import { takeUntilDestroyed, toObservable, toSignal } from '@angular/core/rxjs-interop';
import { applyEach, disabled, form } from '@angular/forms/signals';
import { RouterLink } from '@angular/router';
import { Subject, concatMap, from, of, switchMap, tap } from 'rxjs';

import { appLink } from '@app/routing/app-routes';
import { GameUpdate } from '@app/settings/components/game-update';
import { MissionTable } from '@app/settings/components/mission-table';
import { SettingsRow } from '@app/settings/components/settings-row';
import { SECTION_RENDERERS } from '@app/settings/section-renderers';
import { SettingsActions } from '@app/settings/settings-actions';
import { SettingsStore } from '@app/settings/settings-store';
import { slugOf } from '@app/settings/slug';
import { EmptyState } from '@app/ui/empty-state';
import { Notice } from '@app/ui/notice';
import { Field } from '@gen/tf2ap/launcher/v1/form_pb';

/** Placed is a row and the section it was found in. */
interface Placed {
  readonly section: string;
  readonly field: Field;
}

/** Section is one tab of a page: its own group of rows, or a page filed under it. */
interface Section {
  readonly title: string;
  readonly slug: string;
}

/**
 * One row of the form the page binds to. locked travels in the model rather
 * than being looked up by index: the schema reads the row it is applied to, and
 * a row that moved would otherwise take another row's answer.
 */
interface Row {
  id: string;
  value: string;
  locked: boolean;
}

// The Missions page draws its pool as a table, so the tick per mission and the
// two buttons that tick them all are the table's and not rows as well.
const poolRow = /^missions\.pool[._]/;

/**
 * One settings page. Its sections are tabs across the top: the groups its rows
 * were declared in, then every page filed under it. One section is on screen
 * at a time, so a page with sixty rows is four screens of fifteen rather than
 * one long scroll. A search flattens them: a setting the player cannot name
 * the section of is the one they are searching for.
 *
 * A section a domain draws itself arrives through SECTION_RENDERERS; the rest
 * are the rows. Either way the form is signal forms over every row of the
 * page, and what a row means still belongs to the launcher.
 */
@Component({
  selector: 'app-settings-tab-page',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    EmptyState,
    GameUpdate,
    MissionTable,
    NgComponentOutlet,
    Notice,
    RouterLink,
    SettingsRow,
  ],
  templateUrl: './settings-tab-page.html',
  styleUrl: './settings-tab-page.scss',
})
export class SettingsTabPage {
  readonly store = inject(SettingsStore);
  private readonly actions = inject(SettingsActions);
  private readonly renderers = inject(SECTION_RENDERERS);

  readonly settingsTab = input('');
  readonly settingsSection = input('');
  readonly link = appLink;

  readonly tab = computed(() => this.store.tab(this.settingsTab()));
  readonly title = computed(() => this.tab()?.title ?? '');
  readonly intro = computed(() => this.tab()?.intro ?? '');
  readonly missionTabs = computed(() => this.settingsTab() === 'missions');
  readonly gameServerTab = computed(() => this.settingsTab() === 'game-server');

  /** The rows of the page, in order, each with the section it belongs to. */
  readonly rows = computed<Placed[]>(() => {
    const tab = this.tab();
    if (tab === undefined) {
      return [];
    }
    const own = tab.fields.map((field) => ({ section: field.group || tab.title, field }));
    const under = this.store
      .tabs()
      .filter((candidate) => candidate.under === tab.title)
      .flatMap((nested) => nested.fields.map((field) => ({ section: nested.title, field })));
    return [...own, ...under].filter(({ field }) => !poolRow.test(field.id));
  });

  readonly sections = computed<Section[]>(() => {
    const seen: Section[] = [];
    for (const { section } of this.rows()) {
      if (!seen.some((one) => one.title === section)) {
        seen.push({ title: section, slug: slugOf(section) });
      }
    }
    return seen;
  });

  readonly tabbed = computed(() => this.sections().length > 1);
  readonly current = computed<Section | undefined>(
    () =>
      this.sections().find((one) => one.slug === this.settingsSection()) ?? this.sections().at(0),
  );
  readonly searching = computed(() => this.store.search().trim() !== '');

  /** What is drawn as rows: the section on screen, or every match of a search. */
  readonly shown = computed(() =>
    this.rows().filter(
      ({ section, field }) =>
        this.store.matches(field) && (this.searching() || section === this.current()?.title),
    ),
  );

  /** The component a domain registered for the section on screen, if any. */
  readonly custom = toSignal(
    toObservable(computed(() => `${this.settingsTab()}/${this.current()?.slug ?? ''}`)).pipe(
      switchMap((key) => {
        const renderer = this.renderers.find((one) => one.key === key);
        return renderer === undefined ? of<Type<object> | null>(null) : from(renderer.load());
      }),
    ),
    { initialValue: null },
  );

  /** Which page of how many, so the player knows how much is left. */
  readonly stepLabel = computed(() => {
    const pages = this.store.topTabs();
    const here = pages.findIndex((tab) => slugOf(tab.title) === this.settingsTab());
    return here < 0 ? '' : `Section ${here + 1} of ${pages.length}`;
  });

  private readonly model = linkedSignal<Placed[], { rows: Row[] }>({
    source: () => this.rows(),
    computation: (placed) => ({
      rows: placed.map(({ field }) => ({
        id: field.id,
        value: this.store.value(field.id),
        locked: field.disabled,
      })),
    }),
  });

  /**
   * The form. Only `disabled` is a rule here: a Number's floor and ceiling are
   * on the input as min and max, and what a value means is form.Apply's on the
   * Go side, with the words the player reads. A second copy of that check here
   * would be a second place for it to be wrong.
   */
  readonly page = form(this.model, (path) => {
    applyEach(path.rows, (row) => {
      disabled(row.value, (context) => context.valueOf(row.locked));
    });
  });

  /** said is the last thing a button answered with: a path opened, a page to
      visit, a file saved. The launcher says the rest on the stream. */
  readonly said = signal('');
  readonly saidLink = computed(() => {
    const answer = this.said();
    return answer.startsWith('https://login.tailscale.com/') ? answer : '';
  });

  readonly fired = new Subject<string>();

  constructor() {
    this.fired
      .pipe(
        concatMap((id) => this.actions.press(id)),
        tap((said) => this.said.set(said)),
        takeUntilDestroyed(),
      )
      .subscribe();
  }

  answer(id: string, value: string): void {
    this.store.change(id, value);
  }

  value(id: string): string {
    return this.store.value(id);
  }

  indexOf(id: string): number {
    return this.rows().findIndex(({ field }) => field.id === id);
  }
}
