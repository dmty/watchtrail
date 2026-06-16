import { describe, it, expect } from "vitest";
import { buildEvent } from "./event";
import type { Identity } from "./identity";

const yt: Identity = { source_kind: "youtube", external_id: "abc123" };

describe("buildEvent", () => {
  it("assembles a canonical start event, omitting absent optionals", () => {
    const ev = buildEvent(
      { type: "start", identity: yt, position_seconds: 0, title: "T", duration_seconds: 120 },
      () => "2026-06-16T10:00:00.000Z",
      () => "uuid-1",
    );
    expect(ev).toEqual({
      v: 1,
      event_id: "uuid-1",
      type: "start",
      occurred_at: "2026-06-16T10:00:00.000Z",
      source_kind: "youtube",
      media: { external_id: "abc123", title: "T", duration_seconds: 120 },
      position_seconds: 0,
    });
  });

  it("carries identity meta when the input has none", () => {
    const idy: Identity = { source_kind: "youtube", external_id: "x", meta: { channel: "C" } };
    const ev = buildEvent({ type: "progress", identity: idy }, () => "t", () => "id");
    expect(ev.meta).toEqual({ channel: "C" });
  });

  it("input meta overrides identity meta", () => {
    const idy: Identity = { source_kind: "web", external_id: "x", meta: { channel: "C" } };
    const ev = buildEvent({ type: "stop", identity: idy, meta: { foo: 1 } }, () => "t", () => "id");
    expect(ev.meta).toEqual({ foo: 1 });
  });
});
