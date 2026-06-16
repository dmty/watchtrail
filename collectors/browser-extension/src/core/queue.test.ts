import { describe, it, expect } from "vitest";
import { enqueue, drainBatch, ack, MAX_QUEUE } from "./queue";
import type { WatchEvent } from "./event";

const ev = (id: string): WatchEvent => ({
  v: 1, event_id: id, type: "start", occurred_at: "t", source_kind: "web", media: { external_id: "x" },
});

describe("queue", () => {
  it("enqueue appends without dropping under the cap", () => {
    const { queue, dropped } = enqueue([], ev("a"));
    expect(queue.map((e) => e.event_id)).toEqual(["a"]);
    expect(dropped).toBe(0);
  });

  it("drainBatch returns the head and its ids; ack removes them", () => {
    const q = [ev("a"), ev("b"), ev("c")];
    const { batch, ids } = drainBatch(q, 2);
    expect(ids).toEqual(["a", "b"]);
    expect(batch.length).toBe(2);
    expect(ack(q, ids).map((e) => e.event_id)).toEqual(["c"]);
  });

  it("caps at MAX_QUEUE by dropping oldest", () => {
    const big: WatchEvent[] = [];
    for (let i = 0; i < MAX_QUEUE; i++) big.push(ev("e" + i));
    const { queue, dropped } = enqueue(big, ev("new"));
    expect(dropped).toBe(1);
    expect(queue.length).toBe(MAX_QUEUE);
    expect(queue[0].event_id).toBe("e1");
    expect(queue[MAX_QUEUE - 1].event_id).toBe("new");
  });
});
