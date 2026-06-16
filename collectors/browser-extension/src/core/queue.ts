import type { WatchEvent } from "./event";

export const MAX_QUEUE = 1000;

export function enqueue(q: WatchEvent[], event: WatchEvent): { queue: WatchEvent[]; dropped: number } {
  const next = [...q, event];
  if (next.length > MAX_QUEUE) {
    const dropped = next.length - MAX_QUEUE;
    return { queue: next.slice(dropped), dropped };
  }
  return { queue: next, dropped: 0 };
}

export function drainBatch(q: WatchEvent[], max: number): { batch: WatchEvent[]; ids: string[] } {
  const batch = q.slice(0, max);
  return { batch, ids: batch.map((e) => e.event_id) };
}

export function ack(q: WatchEvent[], ids: string[]): WatchEvent[] {
  const set = new Set(ids);
  return q.filter((e) => !set.has(e.event_id));
}
