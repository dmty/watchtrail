import { describe, it, expect } from "vitest";
import { probeCore } from "./probe";

describe("probeCore", () => {
  it("returns not-configured when coreUrl is empty", async () => {
    const r = await probeCore("", "");
    expect(r.state).toBe("not-configured");
  });

  it("returns not-configured when coreUrl is whitespace", async () => {
    const r = await probeCore("   ", "");
    expect(r.state).toBe("not-configured");
  });

  it("returns reachable on 2xx", async () => {
    const fetchFn = async () => new Response("ok", { status: 200 });
    const r = await probeCore("http://localhost:8765", "", { fetchFn: fetchFn as typeof fetch });
    expect(r).toEqual({ state: "reachable", status: 200 });
  });

  it("returns unreachable on non-2xx", async () => {
    const fetchFn = async () => new Response("nope", { status: 401 });
    const r = await probeCore("http://localhost:8765", "x", { fetchFn: fetchFn as typeof fetch });
    expect(r.state).toBe("unreachable");
  });

  it("returns unreachable on network error", async () => {
    const fetchFn = async () => { throw new Error("connection refused"); };
    const r = await probeCore("http://localhost:8765", "", { fetchFn: fetchFn as typeof fetch });
    expect(r).toEqual({ state: "unreachable", reason: "connection refused" });
  });

  it("sends bearer token when present", async () => {
    let seenAuth: string | null = null;
    const fetchFn = async (_url: RequestInfo | URL, init?: RequestInit) => {
      seenAuth = new Headers(init?.headers).get("authorization");
      return new Response("ok", { status: 200 });
    };
    await probeCore("http://localhost:8765", "abc", { fetchFn: fetchFn as typeof fetch });
    expect(seenAuth).toBe("Bearer abc");
  });
});
