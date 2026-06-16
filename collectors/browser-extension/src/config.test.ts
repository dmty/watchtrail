import { describe, it, expect } from "vitest";
import { withDefaults, DEFAULT_CONFIG } from "./config";

describe("withDefaults", () => {
  it("fills the core default URL", () => {
    expect(withDefaults({}).coreUrl).toBe("http://127.0.0.1:8765");
  });
  it("overlays partials onto defaults", () => {
    expect(withDefaults({ token: "x" })).toEqual({ ...DEFAULT_CONFIG, token: "x" });
    expect(withDefaults({ enabled: false }).enabled).toBe(false);
  });
});
