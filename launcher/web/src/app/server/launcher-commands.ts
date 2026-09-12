import { Injectable, inject } from '@angular/core';
import { Client, createClient } from '@connectrpc/connect';
import { Observable, from } from 'rxjs';

import { LAUNCHER_TRANSPORT } from '@app/transport/connect-transport';
import { FilesService } from '@gen/tf2ap/launcher/v1/files_pb';
import { FileTarget } from '@gen/tf2ap/launcher/v1/files_pb';
import { LauncherService } from '@gen/tf2ap/launcher/v1/launcher_pb';

/**
 * Every button the launcher answers. One place, so a screen presses a named
 * thing rather than assembling a request.
 *
 * Each method returns an Observable of the answer. Nothing here waits: the
 * state a press changes arrives on the stream, not in the reply, because a
 * second browser watching has to see the same thing.
 */
@Injectable({ providedIn: 'root' })
export class LauncherCommands {
  private readonly launcher: Client<typeof LauncherService> = createClient(
    LauncherService,
    inject(LAUNCHER_TRANSPORT),
  );

  private readonly files: Client<typeof FilesService> = createClient(
    FilesService,
    inject(LAUNCHER_TRANSPORT),
  );

  start(): Observable<void> {
    return this.done(this.launcher.start({}));
  }

  stop(): Observable<void> {
    return this.done(this.launcher.stop({}));
  }

  restart(): Observable<void> {
    return this.done(this.launcher.restart({}));
  }

  quit(): Observable<void> {
    return this.done(this.launcher.quit({}));
  }

  sendRcon(command: string): Observable<void> {
    return this.done(this.launcher.sendRcon({ command }));
  }

  setMission(popFile: string): Observable<void> {
    return this.done(this.launcher.setMission({ popFile }));
  }

  /** resumeMission loads a mission and starts it at a wave, with the money the
      game gives for having won every wave before it. */
  resumeMission(popFile: string, wave: number): Observable<void> {
    return this.done(this.launcher.resumeMission({ popFile, wave }));
  }

  approveFunnel(): Observable<{ approvalUrl: string; message: string }> {
    return from(this.launcher.approveFunnel({}));
  }

  /** showFile answers with the path it opened, which is what to say when the
      desktop had no handler for it. */
  showFile(target: FileTarget): Observable<{ path: string }> {
    return from(this.files.showFile({ target }));
  }

  /** debugBundle is a stream of chunks: the first message names the file and
      the ones after it carry bytes. */
  debugBundle(): Observable<{ filename: string; chunk: Uint8Array }> {
    return from(this.files.downloadDebugBundle({}));
  }

  /** listFolder is what a Browse row picks from: a browser has no file dialog,
      so the launcher reads the folder and the picker draws it. */
  listFolder(path: string): Observable<{ path: string; parent: string; folders: string[] }> {
    return from(this.files.listFolder({ path }));
  }

  /** done drops an empty answer. Every one of these replies carries no fields:
      what the press changed arrives on the stream, not here. */
  private done(call: Promise<object>): Observable<void> {
    return from(call.then(() => undefined));
  }
}
