import { TestBed } from '@angular/core/testing';
import { create } from '@bufbuild/protobuf';
import { Subject } from 'rxjs';
import { beforeEach, describe, expect, it } from 'vitest';

import { Frame, LauncherStream } from '@app/server/launcher-stream';
import { LauncherStore } from '@app/server/launcher-store';
import { LogLineSchema, ServerStatus, SnapshotSchema } from '@gen/tf2ap/launcher/v1/launcher_pb';
import { StreamMessageSchema } from '@gen/tf2ap/launcher/v1/stream_pb';

/**
 * The store folds every frame into one reading of the launcher. What matters
 * here is what it does with a log: the first frame carries every line, and no
 * frame after it carries any, so a fold that took the newer empty list would
 * blank the player's log on the first redraw.
 */
describe('the launcher store', () => {
  let frames: Subject<Frame>;
  let store: LauncherStore;

  function snapshot(over: Parameters<typeof create<typeof SnapshotSchema>>[1]): Frame {
    return create(StreamMessageSchema, {
      body: { case: 'snapshot', value: create(SnapshotSchema, over) },
    });
  }

  function line(text: string): Frame {
    return create(StreamMessageSchema, {
      body: { case: 'line', value: create(LogLineSchema, { source: 'srcds', text }) },
    });
  }

  beforeEach(() => {
    frames = new Subject<Frame>();
    TestBed.configureTestingModule({
      providers: [{ provide: LauncherStream, useValue: { frames: () => frames } }],
    });
    store = TestBed.inject(LauncherStore);
  });

  it('is not connected until the first frame', () => {
    expect(store.connected()).toBe(false);
    expect(store.status()).toBe(ServerStatus.UNSPECIFIED);
  });

  it('takes the first frame whole', () => {
    frames.next(
      snapshot({
        title: 'Mann vs Archipelago',
        status: ServerStatus.RUNNING,
        running: true,
        activity: 'downloading archive-assets.zip: 25%',
        managedExternally: true,
        logs: [create(LogLineSchema, { source: 'srcds', text: 'a line from before' })],
      }),
    );
    expect(store.connected()).toBe(true);
    expect(store.title()).toBe('Mann vs Archipelago');
    expect(store.running()).toBe(true);
    expect(store.activity()).toBe('downloading archive-assets.zip: 25%');
    expect(store.managedExternally()).toBe(true);
    expect(store.logs()).toHaveLength(1);
  });

  it('reads the TF2 build and whether Steam has a newer one', () => {
    frames.next(
      snapshot({ gameBuild: '16234567', gameUpdateAvailable: true, gameUpdateError: 'stuck' }),
    );
    expect(store.gameBuild()).toBe('16234567');
    expect(store.gameUpdateAvailable()).toBe(true);
    expect(store.gameUpdateError()).toBe('stuck');
  });

  it('keeps the log when a redraw arrives without one', () => {
    frames.next(snapshot({ logs: [create(LogLineSchema, { text: 'kept' })] }));
    frames.next(snapshot({ status: ServerStatus.RUNNING }));

    expect(store.status()).toBe(ServerStatus.RUNNING);
    expect(store.logs()).toHaveLength(1);
    expect(store.logs()[0].text).toBe('kept');
  });

  // A browser that fell behind is sent the whole state again, log included:
  // the lines it missed are in there, and the ones it holds are a prefix.
  it('takes the log again when a resync carries one', () => {
    frames.next(snapshot({ logs: [create(LogLineSchema, { text: 'first' })] }));
    frames.next(
      snapshot({
        logs: [create(LogLineSchema, { text: 'first' }), create(LogLineSchema, { text: 'missed' })],
      }),
    );

    expect(store.logs().map((row) => row.text)).toEqual(['first', 'missed']);
  });

  it('appends a line as it happens', () => {
    frames.next(snapshot({}));
    frames.next(line('the server said something'));
    frames.next(line('and then something else'));

    expect(store.logs().map((row) => row.text)).toEqual([
      'the server said something',
      'and then something else',
    ]);
  });

  // A tab left open all evening must cost a bounded amount of memory, and the
  // bound is the launcher's own so neither side shows what the other dropped.
  // A tab left open all evening must cost a bounded amount of memory, and the
  // bound is the launcher's own so neither side shows what the other dropped.
  // The full log arrives in one frame, which is how it really comes: the lines
  // after it are the few that happened since.
  it('holds no more lines than the launcher does, oldest first out', () => {
    const full = Array.from({ length: 20_000 }, (_, i) =>
      create(LogLineSchema, { text: `line ${i}` }),
    );
    frames.next(snapshot({ logs: full }));
    expect(store.logs()).toHaveLength(20_000);

    frames.next(line('one more'));
    expect(store.logs()).toHaveLength(20_000);
    expect(store.logs()[0].text).toBe('line 1');
    expect(store.logs()[19_999].text).toBe('one more');
  });

  it('says the launcher is unreachable once the stream gives up', () => {
    frames.next(snapshot({ status: ServerStatus.RUNNING }));
    expect(store.lost()).toBe(false);

    frames.next({ lost: true });
    expect(store.lost()).toBe(true);
    expect(store.connected()).toBe(false);
  });
});
