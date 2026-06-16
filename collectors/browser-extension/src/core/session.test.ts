import { describe, it, expect } from "vitest";
import { newSession, step } from "./session";

describe("session step", () => {
  it("emits start once, then resume on later play", () => {
    let s = newSession();
    let r = step(s, "play", 1000);
    expect(r.type).toBe("start");
    s = r.state;
    r = step(s, "pause", 2000);
    expect(r.type).toBe("pause");
    s = r.state;
    r = step(s, "play", 3000);
    expect(r.type).toBe("resume");
  });

  it("throttles timeupdate progress to >= 30s", () => {
    let s = step(newSession(), "play", 1000).state;
    expect(step(s, "timeupdate", 5000).type).toBeNull(); // <30s since start
    const r = step(s, "timeupdate", 31001);
    expect(r.type).toBe("progress");
    s = r.state;
    expect(step(s, "timeupdate", 40000).type).toBeNull(); // <30s since last progress
  });

  it("ended and hide emit stop and clear started", () => {
    let s = step(newSession(), "play", 0).state;
    const ended = step(s, "ended", 100);
    expect(ended.type).toBe("stop");
    expect(ended.state.started).toBe(false);

    s = step(newSession(), "play", 0).state;
    expect(step(s, "hide", 100).type).toBe("stop");
  });

  it("ignores pause/seek before any play", () => {
    expect(step(newSession(), "pause", 0).type).toBeNull();
    expect(step(newSession(), "seeked", 0).type).toBeNull();
  });

  it("emits seek while playing", () => {
    const s = step(newSession(), "play", 0).state;
    expect(step(s, "seeked", 1000).type).toBe("seek");
  });

  it("emits progress at exactly the 30s boundary", () => {
    const s = step(newSession(), "play", 1000).state; // lastProgress = 1000
    expect(step(s, "timeupdate", 31000).type).toBe("progress"); // exactly +30000, >= boundary
  });
});
