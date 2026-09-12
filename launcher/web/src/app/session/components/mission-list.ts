import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Subject, exhaustMap } from 'rxjs';

import { LauncherCommands } from '@app/server/launcher-commands';
import { LauncherStore } from '@app/server/launcher-store';
import { EmptyState } from '@app/ui/empty-state';
import { MissionRow, MissionTone, MissionsTable } from '@app/ui/missions-table';
import { SessionMission } from '@gen/tf2ap/launcher/v1/launcher_pb';

// What a state means, and the order the table sorts them in: what you can play
// now first, what you have done last.
const states: Record<string, number> = { unlocked: 0, locked: 1, elsewhere: 2, played: 3 };

/**
 * The run's missions and what has happened to each.
 *
 * Played and cleared are not the same thing. Another world's !collect sends
 * every check it still holds, so the room can hold a mission's check that this
 * server never played, and a table saying only "cleared" would tell a player
 * they had finished a mission they had never loaded.
 */
@Component({
  selector: 'app-mission-list',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [EmptyState, MissionsTable],
  templateUrl: './mission-list.html',
  styleUrl: './mission-list.scss',
})
export class MissionList {
  private readonly store = inject(LauncherStore);
  private readonly commands = inject(LauncherCommands);

  readonly running = computed(() => this.store.running());

  /** The rows in state order, which is what the table keeps for the State column. */
  readonly rows = computed<MissionRow[]>(() => {
    const playing = this.store.mission();
    const running = this.running();
    return this.store
      .missions()
      .map((mission) => ({ mission, ...describe(mission) }))
      .toSorted(
        (left, right) =>
          left.order - right.order || left.mission.name.localeCompare(right.mission.name),
      )
      .map(({ mission, state, tone }) => ({
        key: mission.popFile,
        name: mission.name,
        map: mission.map,
        tier: mission.tier,
        source: mission.source,
        waves: mission.waves,
        loadout: mission.loadout,
        status: state,
        tone,
        badge: false,
        mods: '',
        on: mission.unlocked,
        disabled: false,
        dim: !mission.unlocked,
        playing: mission.popFile === playing,
        play: mission.unlocked && running ? (mission.played ? 'Replay' : 'Play') : '',
        // Every wave of the mission, so a team can go straight to the one they
        // want. Offered only where it would do something: the mission has to
        // be playable now and the server up.
        waveChoices:
          mission.unlocked && running && mission.waves > 1
            ? Array.from({ length: mission.waves }, (_, index) => index + 1)
            : [],
        // On the wave they got to, which is the one they came back for.
        waveStart: Math.min(mission.waveReached + 1, Math.max(mission.waves, 1)),
      }));
  });

  /** What the multiworld has to do with this list, in one line. */
  readonly multiworldLine = computed(() => {
    const missions = this.store.missions();
    if (missions.length === 0) {
      return '';
    }
    const unlocked = missions.filter((mission) => mission.unlocked).length;
    const played = missions.filter((mission) => mission.played).length;
    return `${unlocked} of ${missions.length} unlocked, ${played} played`;
  });

  readonly switchHint = computed(() => {
    if (!this.running()) {
      return 'Start the server to load a mission.';
    }
    const play =
      'Play next loads the mission on the running server. Anyone on it is sent to the new map.';
    if (!this.rows().some((row) => row.waveChoices.length > 0)) {
      return play;
    }
    return (
      play + ' Picking a wave starts the mission there, with the money for the waves before it.'
    );
  });

  readonly choose = new Subject<string>();
  readonly resume = new Subject<{ key: string; wave: number }>();

  constructor() {
    this.choose
      .pipe(
        exhaustMap((popFile) => this.commands.setMission(popFile)),
        takeUntilDestroyed(),
      )
      .subscribe();
    this.resume
      .pipe(
        exhaustMap((asked) => this.commands.resumeMission(asked.key, asked.wave)),
        takeUntilDestroyed(),
      )
      .subscribe();
  }
}

function describe(mission: SessionMission): { state: string; tone: MissionTone; order: number } {
  if (mission.played) {
    return { state: 'played', tone: 'good', order: states['played'] };
  }
  if (mission.cleared) {
    return { state: 'cleared elsewhere', tone: 'info', order: states['elsewhere'] };
  }
  if (mission.unlocked) {
    return { state: 'unlocked', tone: 'accent', order: states['unlocked'] };
  }
  return { state: 'locked', tone: 'neutral', order: states['locked'] };
}
