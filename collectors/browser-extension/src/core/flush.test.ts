import { describe, it, expect } from "vitest";
import { flushOnce } from "./flush";
import type { WatchEvent } from "./event";

const ev = (id: string): WatchEvent => ({
  v: 1, event_id: id, type: "start", occurred_at: "t", source_kind: "web", media: { external_id: "x" },
});

describe("flushOnce", () => {
  it("acks the batch on success and persists the remainder", async () => {
    let saved: WatchEvent[] | null = null;
    const res = await flushOnce({
      loadQueue: async () => [ev("a"), ev("b")],
      saveQueue: async (q) => { saved = q; },
      post: async () => true,
      batchMax: 10,
    });
    expect(res.sent).toBe(2);
    expect(saved).toEqual([]);
  });

  it("keeps the queue and does not save on failure", async () => {
    let saved = false;
    const res = await flushOnce({
      loadQueue: async () => [ev("a")],
      saveQueue: async () => { saved = true; },
      post: async () => false,
      batchMax: 10,
    });
    expect(res.sent).toBe(0);
    expect(res.kept).toBe(1);
    expect(saved).toBe(false);
  });

  it("is a no-op on an empty queue", async () => {
    const res = await flushOnce({
      loadQueue: async () => [],
      saveQueue: async () => { throw new Error("should not save"); },
      post: async () => { throw new Error("should not post"); },
      batchMax: 10,
    });
    expect(res).toEqual({ sent: 0, kept: 0 });
  });
});
