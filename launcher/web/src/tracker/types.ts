import { BadgeTone } from '@app/ui/badge';
import { Mercenary } from '@app/ui/tf2-art';

export interface Objective {
  readonly id: number;
  readonly name: string;
  readonly kind: string;
  readonly wave?: number;
}

export interface Mission {
  readonly pop_file: string;
  readonly name: string;
  readonly difficulty: string;
  readonly locations: readonly Objective[];
}

export interface Weapon {
  readonly name: string;
  readonly classes: readonly string[];
  readonly icon?: string;
  readonly aliases?: readonly string[];
}

export interface Player {
  readonly player: number;
  readonly game: string;
}

export interface SlotData {
  readonly missions?: readonly string[];
  readonly start_mission?: string;
  readonly goal?: string;
  readonly goal_mission?: string;
  readonly missionsanity_target?: number;
  readonly mission_ticket_importance?: string;
  readonly starting_items?: readonly (number | string)[];
  readonly tracker?: {
    readonly starting_items?: readonly (number | string)[];
    readonly missions?: readonly Mission[];
  };
}

export interface ReceivedRow {
  readonly player: number;
  readonly items: readonly (readonly number[] | { readonly item: number })[];
}

export interface ChecksRow {
  readonly player: number;
  readonly locations: readonly number[];
}

export interface SlotRow {
  readonly player: number;
  readonly slot_data: SlotData;
}

export interface TotalRow {
  readonly player: number;
  readonly total_locations: number;
}

export interface TrackerSource {
  readonly mode: 'live' | 'demo';
  readonly host: string;
  readonly kind: 'room' | 'tracker';
  readonly id: string;
  readonly tracker: string;
  readonly preferredPlayer?: number;
  readonly players: readonly Player[];
  readonly names: ReadonlyMap<number, string>;
  readonly catalog: readonly Mission[];
  readonly buffWeapons: ReadonlyMap<string, Weapon>;
  readonly itemNames: ReadonlyMap<number, string>;
  readonly slots: readonly SlotRow[];
  readonly totals: readonly TotalRow[];
  readonly received: readonly ReceivedRow[];
  readonly checks: readonly ChecksRow[];
  readonly demoSlotData?: SlotData;
}

export interface MissionView extends Mission {
  readonly complete: boolean;
  readonly locked: boolean;
  readonly finalBoss: boolean;
  readonly progress: number;
}

export interface ClassView {
  readonly name: Mercenary;
  readonly unlocked: boolean;
}

export interface UnlockView {
  readonly kind: string;
  readonly name: string;
  readonly count: number;
  readonly tone: BadgeTone;
}

export interface BuffEffect {
  readonly effect: string;
  readonly count: number;
}

export interface BuffWeaponView {
  readonly weapon: string;
  readonly icon?: string;
  readonly aliases: readonly string[];
  readonly effects: readonly BuffEffect[];
  readonly total: number;
}

export interface TrackerView {
  readonly player: number;
  readonly playerName: string;
  readonly missions: readonly MissionView[];
  readonly checked: ReadonlySet<number>;
  readonly owned: ReadonlyMap<string, number>;
  readonly classes: readonly ClassView[];
  readonly classCount: number;
  readonly unlocks: readonly UnlockView[];
  readonly slotCount: number;
  readonly grapplingHook: boolean;
  readonly completed: number;
  readonly total: number;
  readonly goalTitle: string;
  readonly goalDescription: string;
  readonly goalPercent: number;
  readonly oldInventory: boolean;
}
