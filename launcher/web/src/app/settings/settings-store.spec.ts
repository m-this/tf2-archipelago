import { TestBed } from '@angular/core/testing';
import { Code, ConnectError, Transport } from '@connectrpc/connect';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { SettingsStore } from '@app/settings/settings-store';
import { LAUNCHER_TRANSPORT } from '@app/transport/connect-transport';

/**
 * The launcher owns the answers, so what the store owns is the timing: when a
 * keystroke is told to the launcher, and what Save does about the ones that
 * have not been told yet.
 */
describe('the settings store', () => {
  let sent: { field: string; value: string }[];
  let order: string[];
  let saves: number;
  let store: SettingsStore;

  beforeEach(() => {
    vi.useFakeTimers();
    sent = [];
    order = [];
    saves = 0;

    // A transport that records what each call carried. Enough to see the order
    // and the timing, which is the whole of what is under test here.
    const transport = {
      unary: (
        method: { name: string },
        _s: unknown,
        _t: unknown,
        _h: unknown,
        message: unknown,
      ) => {
        const body = message as { change?: { field: string; value: string } };
        if (method.name === 'ChangeSetting' && body.change) {
          if (body.change.value === 'refused') {
            return Promise.reject(new ConnectError('not that one', Code.InvalidArgument));
          }
          sent.push({ field: body.change.field, value: body.change.value });
          order.push(`change ${body.change.field}=${body.change.value}`);
        }
        if (method.name === 'SaveSettings') {
          saves++;
          order.push('save');
        }
        if (method.name === 'DispatchAction') {
          order.push('press');
        }
        return Promise.resolve({ message: { saved: true, refusal: '' }, stream: false });
      },
      stream: () => Promise.reject(new Error('no streams here')),
    } as unknown as Transport;

    TestBed.configureTestingModule({
      providers: [{ provide: LAUNCHER_TRANSPORT, useValue: transport }],
    });
    store = TestBed.inject(SettingsStore);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('waits for the typing to stop before telling the launcher', () => {
    store.change('rewards.traps', '4');
    store.change('rewards.traps', '42');
    expect(sent).toHaveLength(0);

    vi.advanceTimersByTime(300);
    expect(sent).toEqual([{ field: 'rewards.traps', value: '42' }]);
  });

  it('lets two rows answered in the same breath both land', () => {
    store.change('rewards.traps', '42');
    store.change('run.install_root', '/games/tf2');

    vi.advanceTimersByTime(300);
    expect(sent.map((one) => one.field).sort()).toEqual(['rewards.traps', 'run.install_root']);
  });

  it('shows what was typed while the launcher has not caught up', () => {
    store.change('rewards.traps', '42');
    expect(store.value('rewards.traps')).toBe('42');
  });

  // Pressing Save inside the quarter second used to write the value from before
  // the word was finished: the player watched their own typing be discarded by
  // the button meant to keep it. What matters is the order, not the count: the
  // answer has to reach the launcher before the file is written.
  it('sends what is still in flight before it saves', async () => {
    store.change('rewards.traps', '42');
    const done = store.save(false).toPromise();

    await vi.runAllTimersAsync();
    await done;

    expect(order.indexOf('save')).toBeGreaterThan(0);
    expect(order.slice(0, order.indexOf('save'))).toContain('change rewards.traps=42');
    expect(saves).toBe(1);
  });

  // One refused keystroke used to end the subscription every row shares:
  // nothing typed after it reached the launcher, and nobody was told why.
  it('shows a refusal and still sends the next answer', async () => {
    store.change('rewards.traps', 'refused');
    await vi.advanceTimersByTimeAsync(300);
    expect(store.refusal()).toBe('not that one');

    store.change('rewards.traps', '42');
    await vi.advanceTimersByTimeAsync(300);
    expect(sent).toEqual([{ field: 'rewards.traps', value: '42' }]);
    expect(store.refusal()).toBe('');
  });

  // Discard drops the launcher's draft. What was typed and not yet sent has to
  // go with it, or the next Save writes back what was just thrown away.
  it('forgets what was typed when the draft is discarded', async () => {
    store.change('rewards.traps', '42');
    store.cancel().subscribe();
    expect(store.dirty()).toBe(false);

    const done = store.save(false).toPromise();
    await vi.runAllTimersAsync();
    await done;
    expect(sent).toHaveLength(0);
    expect(saves).toBe(1);
  });

  // A name typed and its button pressed inside the quarter second used to reach
  // the launcher after the press, which saved nothing and said to name it first.
  it('sends what is still in flight before it presses a button', async () => {
    store.change('loadout.name', 'pop');
    const done = store.dispatch('loadout.save').toPromise();

    await vi.runAllTimersAsync();
    await done;

    expect(order.indexOf('press')).toBeGreaterThan(0);
    expect(order.slice(0, order.indexOf('press'))).toContain('change loadout.name=pop');
  });

  it('has nothing to save until something is answered', () => {
    expect(store.dirty()).toBe(false);
    store.change('rewards.traps', '42');
    expect(store.dirty()).toBe(true);
  });
});
