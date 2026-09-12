export const defaultHost = 'https://archipelago.gg';

export interface RoomStatus {
  readonly tracker?: string;
  readonly players?: readonly (readonly string[])[];
}

export interface ParsedSource {
  readonly host: string;
  readonly kind: 'room' | 'tracker' | 'unknown';
  readonly id: string;
  readonly player?: number;
}

export interface ResolvedSource extends Omit<ParsedSource, 'kind'> {
  readonly kind: 'room' | 'tracker';
  readonly tracker: string;
  readonly room?: RoomStatus;
}

export function parseSource(input: string): ParsedSource {
  const raw = input.trim();
  if (raw.length === 0) throw new Error('Paste a room or tracker link first.');
  if (!/^https?:\/\//i.test(raw)) {
    return { host: defaultHost, kind: 'unknown', id: compactId(raw) };
  }
  const url = new URL(raw);
  const parts = url.pathname.split('/').filter(Boolean);
  const roomAt = parts.indexOf('room');
  const trackerAt = parts.indexOf('tracker');
  if (roomAt >= 0 && parts[roomAt + 1] !== undefined) {
    return { host: url.origin, kind: 'room', id: compactId(parts[roomAt + 1]) };
  }
  if (trackerAt >= 0 && parts[trackerAt + 1] !== undefined) {
    const player = Number(parts[trackerAt + 3]);
    return {
      host: url.origin,
      kind: 'tracker',
      id: compactId(parts[trackerAt + 1]),
      ...(Number.isFinite(player) && player > 0 ? { player } : {}),
    };
  }
  throw new Error('Use an Archipelago room or tracker URL.');
}

export function compactId(value: string): string {
  const id = value.trim();
  if (!/^[A-Za-z0-9_-]+$/.test(id)) {
    throw new Error('That does not look like an Archipelago ID.');
  }
  return id;
}
