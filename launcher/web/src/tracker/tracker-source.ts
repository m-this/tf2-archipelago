import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, catchError, defer, forkJoin, map, of, switchMap, throwError } from 'rxjs';

import { sampleSource } from './sample';
import { ResolvedSource, RoomStatus, compactId, defaultHost, parseSource } from './source-address';
import {
  ChecksRow,
  Mission,
  Player,
  ReceivedRow,
  SlotRow,
  TotalRow,
  TrackerSource,
  Weapon,
} from './types';

const game = 'Team Fortress 2 Mann vs Machine';

interface StaticTracker {
  readonly player_game?: readonly Player[];
  readonly player_locations_total?: readonly TotalRow[];
  readonly datapackage?: DataPackageReferences;
}

interface PackageReference {
  readonly checksum?: string;
  readonly version?: number;
}

interface DataPackageReferences {
  readonly games?: Readonly<Record<string, PackageReference>>;
  readonly [name: string]:
    PackageReference | Readonly<Record<string, PackageReference>> | undefined;
}

interface LiveTracker {
  readonly player_items_received?: readonly ReceivedRow[];
  readonly player_checks_done?: readonly ChecksRow[];
}

interface DataPackage {
  readonly games?: Readonly<Record<string, GamePackage>>;
  readonly item_name_to_id?: Readonly<Record<string, number>>;
}

interface GamePackage {
  readonly item_name_to_id?: Readonly<Record<string, number>>;
}

interface MissionCatalog {
  readonly missions?: readonly Mission[];
}

interface WeaponCatalog {
  readonly weapons?: readonly Weapon[];
}

@Injectable({ providedIn: 'root' })
export class TrackerSourceClient {
  private readonly http = inject(HttpClient);

  load(input: string): Observable<TrackerSource> {
    return defer(() => this.resolve(input)).pipe(
      switchMap((resolved) =>
        forkJoin({
          resolved: of(resolved),
          staticData: this.http.get<StaticTracker>(
            `${resolved.host}/api/static_tracker/${resolved.tracker}`,
          ),
          slots: this.http.get<readonly SlotRow[]>(
            `${resolved.host}/api/slot_data_tracker/${resolved.tracker}`,
          ),
          live: this.http.get<LiveTracker>(`${resolved.host}/api/tracker/${resolved.tracker}`),
          catalog: this.catalog(),
        }),
      ),
      switchMap((loaded) => {
        const checksum = datapackageChecksum(loaded.staticData.datapackage);
        if (checksum === undefined) {
          return throwError(() => new Error('The room does not contain the TF2 MvM datapackage.'));
        }
        return this.http
          .get<DataPackage>(`${loaded.resolved.host}/api/datapackage/${checksum}`)
          .pipe(map((gamePackage) => this.source(loaded, gamePackage)));
      }),
    );
  }

  refresh(source: TrackerSource): Observable<TrackerSource> {
    if (source.mode !== 'live') return of(source);
    return this.http.get<LiveTracker>(`${source.host}/api/tracker/${source.tracker}`).pipe(
      map((live) => ({
        ...source,
        received: live.player_items_received ?? [],
        checks: live.player_checks_done ?? [],
      })),
    );
  }

  sample(): TrackerSource {
    return sampleSource(defaultHost);
  }

  private resolve(input: string): Observable<ResolvedSource> {
    const parsed = parseSource(input);
    if (parsed.kind === 'tracker') {
      return of({ ...parsed, kind: 'tracker', tracker: parsed.id });
    }
    return this.http.get<RoomStatus>(`${parsed.host}/api/room_status/${parsed.id}`).pipe(
      map((room) => {
        if (room.tracker === undefined) {
          throw new Error('This room does not have tracking enabled.');
        }
        return { ...parsed, kind: 'room' as const, room, tracker: compactId(room.tracker) };
      }),
      catchError((error: Error) =>
        parsed.kind === 'room'
          ? throwError(() => error)
          : of({ ...parsed, kind: 'tracker' as const, tracker: parsed.id }),
      ),
    );
  }

  private catalog(): Observable<{ missions: readonly Mission[]; weapons: readonly Weapon[] }> {
    const at = (base: string) =>
      forkJoin({
        missions: this.http
          .get<MissionCatalog>(`${base}/missions.json`)
          .pipe(map((catalog) => catalog.missions ?? [])),
        weapons: this.http
          .get<WeaponCatalog>(`${base}/weapon_classes.json`)
          .pipe(map((catalog) => catalog.weapons ?? [])),
      });
    return at('./data').pipe(
      catchError(() =>
        at('https://raw.githubusercontent.com/m-this/tf2-archipelago/main/apworld/tf2_mvm/data'),
      ),
    );
  }

  private source(
    loaded: {
      resolved: ResolvedSource;
      staticData: StaticTracker;
      slots: readonly SlotRow[];
      live: LiveTracker;
      catalog: { missions: readonly Mission[]; weapons: readonly Weapon[] };
    },
    envelope: DataPackage,
  ): TrackerSource {
    const players = (loaded.staticData.player_game ?? []).filter((row) => row.game === game);
    if (players.length === 0) {
      throw new Error('This room has no Team Fortress 2 Mann vs Machine slot.');
    }
    const names = new Map<number, string>();
    loaded.resolved.room?.players?.forEach((player, index) => {
      if (player[0] !== undefined) names.set(index + 1, player[0]);
    });
    const gamePackage = envelope.games?.[game] ?? envelope;
    return {
      mode: 'live',
      host: loaded.resolved.host,
      kind: loaded.resolved.kind,
      id: loaded.resolved.id,
      tracker: loaded.resolved.tracker,
      preferredPlayer: loaded.resolved.player,
      players,
      names,
      catalog: loaded.catalog.missions,
      buffWeapons: new Map(loaded.catalog.weapons.map((weapon) => [weapon.name, weapon])),
      itemNames: new Map(
        Object.entries(gamePackage.item_name_to_id ?? {}).map(([name, id]) => [Number(id), name]),
      ),
      slots: loaded.slots,
      totals: loaded.staticData.player_locations_total ?? [],
      received: loaded.live.player_items_received ?? [],
      checks: loaded.live.player_checks_done ?? [],
    };
  }
}

export function datapackageChecksum(
  datapackage: DataPackageReferences | undefined,
): string | undefined {
  const reference = (datapackage?.games ?? datapackage)?.[game];
  const checksum = reference?.checksum;
  return typeof checksum === 'string' ? checksum : undefined;
}
