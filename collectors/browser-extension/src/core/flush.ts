import type { WatchEvent } from "./event";
import { drainBatch, ack } from "./queue";

export interface FlushDeps {
  loadQueue: () => Promise<WatchEvent[]>;
  saveQueue: (q: WatchEvent[]) => Promise<void>;
  post: (batch: WatchEvent[]) => Promise<boolean>;
  batchMax: number;
}

export async function flushOnce(deps: FlushDeps): Promise<{ sent: number; kept: number }> {
  const q = await deps.loadQueue();
  if (q.length === 0) return { sent: 0, kept: 0 };
  const { batch, ids } = drainBatch(q, deps.batchMax);
  const ok = await deps.post(batch);
  if (!ok) return { sent: 0, kept: q.length };
  const remaining = ack(q, ids);
  await deps.saveQueue(remaining);
  return { sent: batch.length, kept: remaining.length };
}
