import { TestBed } from '@angular/core/testing';
import { create } from '@bufbuild/protobuf';
import { Code, ConnectError } from '@connectrpc/connect';
import { Observable, Subject, of, throwError } from 'rxjs';
import { beforeEach, describe, expect, it } from 'vitest';

import { LauncherCommands } from '@app/server/launcher-commands';
import { Frame, LauncherStream } from '@app/server/launcher-stream';
import { GameUpdate } from '@app/settings/components/game-update';
import { SnapshotSchema } from '@gen/tf2ap/launcher/v1/launcher_pb';
import { StreamMessageSchema } from '@gen/tf2ap/launcher/v1/stream_pb';

/**
 * The update block draws what the launcher says about TF2 and presses one
 * button. What matters: the build and the newer-build warning come off the
 * stream, the button waits while something else installs, and a refusal or a
 * failure is said where the button is.
 */
describe('the TF2 update', () => {
  let frames: Subject<Frame>;
  let answer: () => Observable<void>;
  let pressed: number;

  function snapshot(over: Parameters<typeof create<typeof SnapshotSchema>>[1]): void {
    frames.next(
      create(StreamMessageSchema, {
        body: { case: 'snapshot', value: create(SnapshotSchema, over) },
      }),
    );
  }

  function render(): HTMLElement {
    const fixture = TestBed.createComponent(GameUpdate);
    fixture.detectChanges();
    return fixture.nativeElement as HTMLElement;
  }

  function button(host: HTMLElement): HTMLButtonElement {
    const found = host.querySelector('button');
    if (found === null) {
      throw new Error('no Update TF2 button');
    }
    return found;
  }

  beforeEach(() => {
    frames = new Subject<Frame>();
    pressed = 0;
    answer = () => of(undefined);
    TestBed.configureTestingModule({
      providers: [
        { provide: LauncherStream, useValue: { frames: () => frames } },
        {
          provide: LauncherCommands,
          useValue: {
            updateGame: () => {
              pressed++;
              return answer();
            },
          },
        },
      ],
    });
  });

  it('shows the installed build and that Steam has a newer one', () => {
    const fixture = TestBed.createComponent(GameUpdate);
    snapshot({ gameBuild: '16234567', gameUpdateAvailable: true });
    fixture.detectChanges();
    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Installed build: 16234567');
    expect(text).toContain('Steam has a newer TF2 build');
    expect(text).toContain('Stops the server if it is running, updates TF2, then starts it again.');
  });

  it('says the build is unknown when the launcher could not read it', () => {
    snapshot({});
    expect(render().textContent).toContain('Installed build: unknown');
  });

  it('presses once and waits while an install or update runs', () => {
    const fixture = TestBed.createComponent(GameUpdate);
    snapshot({});
    fixture.detectChanges();
    const host = fixture.nativeElement as HTMLElement;
    button(host).click();
    expect(pressed).toBe(1);

    snapshot({ busy: true, activity: 'downloading TF2: 40%' });
    fixture.detectChanges();
    expect(button(host).disabled).toBe(true);
    expect(host.textContent).toContain('downloading TF2: 40%');
  });

  it('shows a refusal where the button is', () => {
    answer = () =>
      throwError(
        () =>
          new ConnectError('another install or update is already running', Code.FailedPrecondition),
      );
    const fixture = TestBed.createComponent(GameUpdate);
    snapshot({});
    fixture.detectChanges();
    const host = fixture.nativeElement as HTMLElement;
    button(host).click();
    fixture.detectChanges();
    expect(host.textContent).toContain('another install or update is already running');
  });

  it('shows why the last update failed', () => {
    const fixture = TestBed.createComponent(GameUpdate);
    snapshot({ gameUpdateError: 'SteamCMD left app 232250 at state 0x6' });
    fixture.detectChanges();
    expect((fixture.nativeElement as HTMLElement).textContent).toContain(
      'The TF2 update failed: SteamCMD left app 232250 at state 0x6',
    );
  });
});
