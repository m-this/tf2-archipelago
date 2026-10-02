import { Injectable, Signal, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { scan } from 'rxjs';

import { Frame, LauncherStream, isLost } from '@app/server/launcher-stream';
import { LogLine, ServerStatus, Session, Snapshot } from '@gen/tf2ap/launcher/v1/launcher_pb';

// The launcher keeps this many lines and no more. Holding the same number here
// means a tab left open all evening costs a bounded amount of memory, and it is
// the same bound on both sides so neither shows what the other dropped.
const logLinesMax = 20_000;

/** State is one reading of the launcher, folded from every frame so far. */
interface State {
  readonly snapshot: Snapshot | undefined;
  readonly logs: readonly LogLine[];
  readonly lost: boolean;
}

const nothingYet: State = { snapshot: undefined, logs: [], lost: false };

/**
 * LauncherStore is the launcher as signals. One stream feeds it and every
 * screen reads it, so no two screens can draw a different answer.
 */
@Injectable({ providedIn: 'root' })
export class LauncherStore {
  private readonly state = toSignal(inject(LauncherStream).frames().pipe(scan(apply, nothingYet)), {
    initialValue: nothingYet,
  });

  /** connected is false before the first frame and after the stream gives up. */
  readonly connected = computed(() => this.state().snapshot !== undefined && !this.state().lost);
  readonly lost = computed(() => this.state().lost);

  readonly title = this.field((s) => s.title, '');
  readonly slot = this.field((s) => s.slot, '');
  readonly status = this.field((s) => s.status, ServerStatus.UNSPECIFIED);
  readonly running = this.field((s) => s.running, false);
  readonly busy = this.field((s) => s.busy, false);
  readonly activity = this.field((s) => s.activity, '');
  readonly room = this.field((s) => s.room, '');
  readonly trackerRoomUrl = this.field((s) => s.trackerRoomUrl, '');
  readonly join = this.field((s) => s.join, '');
  readonly joinUrl = this.field((s) => s.joinUrl, '');
  readonly mission = this.field((s) => s.mission, '');
  readonly drawnBots = this.field((s) => s.drawnBots, '');
  readonly itemServer = this.field((s) => s.itemServer, '');
  readonly notice = this.field((s) => s.notice, '');
  readonly noticeSeq = this.field((s) => s.noticeSeq, 0n);
  readonly sessionError = this.field((s) => s.sessionError, '');
  readonly restartNeeded = this.field((s) => s.restartNeeded, false);
  readonly managedExternally = this.field((s) => s.managedExternally, false);
  readonly gameBuild = this.field((s) => s.gameBuild, '');
  readonly gameUpdateAvailable = this.field((s) => s.gameUpdateAvailable, false);
  readonly gameUpdateError = this.field((s) => s.gameUpdateError, '');

  readonly bots = this.field((s) => s.bots, []);
  readonly missionPool = this.field((s) => s.missionPool, []);
  readonly session = this.field<Session | undefined>((s) => s.session, undefined);
  readonly unlocks = computed(() => this.session()?.unlocks ?? []);
  readonly missions = computed(() => this.session()?.missions ?? []);
  readonly health = computed(() => this.session()?.health);

  readonly logs = computed(() => this.state().logs);

  /** screen is the settings rows, absent while the settings are closed. */
  readonly screen = computed(() => this.state().snapshot?.form);
  readonly screenPage = this.field((s) => s.formPage, '');

  private field<T>(read: (snapshot: Snapshot) => T, absent: T): Signal<T> {
    return computed(() => {
      const snapshot = this.state().snapshot;
      return snapshot === undefined ? absent : read(snapshot);
    });
  }
}

/**
 * apply folds one frame into the state. A snapshot replaces everything, and
 * the log too when it carries lines: the first frame does, and so does a
 * resync for a browser that fell behind. A line is appended, oldest dropped at
 * the bound.
 */
function apply(state: State, frame: Frame): State {
  if (isLost(frame)) {
    return { ...state, lost: true };
  }
  const body = frame.body;
  if (body.case === 'snapshot') {
    const logs = body.value.logs.length > 0 ? body.value.logs : state.logs;
    return { snapshot: body.value, logs: bounded(logs), lost: false };
  }
  if (body.case === 'line') {
    return { ...state, logs: bounded([...state.logs, body.value]) };
  }
  return state;
}

function bounded(logs: readonly LogLine[]): readonly LogLine[] {
  return logs.length <= logLinesMax ? logs : logs.slice(logs.length - logLinesMax);
}
