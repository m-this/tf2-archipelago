import { TestBed } from '@angular/core/testing';
import { create } from '@bufbuild/protobuf';
import { Subject } from 'rxjs';
import { beforeEach, describe, expect, it } from 'vitest';

import { Frame, LauncherStream } from '@app/server/launcher-stream';
import { MissionList } from '@app/session/components/mission-list';
import { SnapshotSchema } from '@gen/tf2ap/launcher/v1/launcher_pb';
import { StreamMessageSchema } from '@gen/tf2ap/launcher/v1/stream_pb';

/**
 * With no missions the page says why when the launcher knows: the bridge did
 * not answer, or it could not get onto the room. Otherwise it keeps the
 * generic line.
 */
describe('the Missions empty state', () => {
  let frames: Subject<Frame>;

  function snapshot(over: Parameters<typeof create<typeof SnapshotSchema>>[1]): void {
    frames.next(
      create(StreamMessageSchema, {
        body: { case: 'snapshot', value: create(SnapshotSchema, over) },
      }),
    );
  }

  function render(over: Parameters<typeof create<typeof SnapshotSchema>>[1]): string {
    const fixture = TestBed.createComponent(MissionList);
    fixture.detectChanges();
    snapshot(over);
    fixture.detectChanges();
    return (fixture.nativeElement as HTMLElement).textContent;
  }

  beforeEach(() => {
    frames = new Subject<Frame>();
    TestBed.configureTestingModule({
      providers: [{ provide: LauncherStream, useValue: { frames: () => frames } }],
    });
  });

  it('keeps the generic line when there is no error', () => {
    const text = render({ session: { health: { lastError: '' } } });
    expect(text).toContain('No missions yet.');
    expect(text).not.toContain('Not on the multiworld');
  });

  it('shows the reason the bridge is not on the room', () => {
    const text = render({
      session: { health: { lastError: 'slot "guebor" refused: invalid slot' } },
    });
    expect(text).toContain('Not on the multiworld: slot "guebor" refused: invalid slot');
    expect(text).not.toContain('No missions yet.');
  });

  it('shows that the bridge is not answering', () => {
    expect(render({ sessionError: 'connection refused' })).toContain(
      'The launcher cannot reach the bridge: connection refused',
    );
  });
});
