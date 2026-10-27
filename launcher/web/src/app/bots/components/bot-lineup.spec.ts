import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { create } from '@bufbuild/protobuf';
import { Observable, of } from 'rxjs';
import { beforeEach, describe, expect, it } from 'vitest';

import { BotLineup } from '@app/bots/components/bot-lineup';
import { LauncherStore } from '@app/server/launcher-store';
import { SettingsStore } from '@app/settings/settings-store';
import { FieldSchema } from '@gen/tf2ap/launcher/v1/form_pb';

/**
 * Save this team used to say "Saved the current seats." the moment it was
 * pressed, while the launcher refused a team with no name (apw-fjv8). What
 * the box says now is what the launcher answered, and a team with no name is
 * not sent at all.
 */
describe('saving a lineup', () => {
  let typed: ReturnType<typeof signal<string>>;
  let answer: () => Observable<string>;
  let dispatched: string[];

  async function render(): Promise<{ host: HTMLElement; settle: () => Promise<void> }> {
    const fixture = TestBed.createComponent(BotLineup);
    fixture.autoDetectChanges();
    await fixture.whenStable();
    return { host: fixture.nativeElement as HTMLElement, settle: () => fixture.whenStable() };
  }

  function saveButton(host: HTMLElement): HTMLButtonElement {
    const found = [...host.querySelectorAll('button')].find((one) =>
      /Save as new|Update/.test(one.textContent ?? ''),
    );
    if (found === undefined) {
      throw new Error('no Save button');
    }
    return found;
  }

  async function press(): Promise<string> {
    const { host, settle } = await render();
    saveButton(host).click();
    await settle();
    return host.querySelector('.feedback')?.textContent ?? '';
  }

  beforeEach(() => {
    typed = signal('');
    dispatched = [];
    answer = () => of('');
    const nameField = create(FieldSchema, { id: 'bots.team_name' });
    TestBed.configureTestingModule({
      providers: [
        {
          provide: SettingsStore,
          useValue: {
            open: () => true,
            field: (id: string) => (id === 'bots.team_name' ? nameField : undefined),
            fieldsMatching: () => [],
            value: (id: string) => (id === 'bots.team_name' ? typed() : ''),
            change: (_id: string, value: string) => typed.set(value),
            openSettings: () => of(undefined),
            dispatch: (id: string) => {
              dispatched.push(id);
              return answer();
            },
          },
        },
        { provide: LauncherStore, useValue: { running: () => true, connected: () => true } },
      ],
    });
  });

  it('does not send a team with no name', async () => {
    expect(saveButton((await render()).host).disabled).toBe(true);
    typed.set('   ');
    expect(await press()).not.toContain('Saved');
    expect(dispatched).toEqual([]);
  });

  it('says what the launcher refused instead of claiming a save', async () => {
    typed.set('Pyro rush');
    answer = () => of('name the team first');
    const said = await press();
    expect(dispatched).toEqual(['bots.save_team']);
    expect(said).toContain('name the team first');
    expect(said).not.toContain('Saved');
  });

  it('names the team it saved once the launcher has answered', async () => {
    typed.set(' Pyro rush ');
    expect(await press()).toContain('Saved the team as Pyro rush.');
  });
});
