import { describe, it, expect } from "vitest";
import { youtubeIdentity, genericIdentity, youtubeIdentityFromState } from "./identity";

describe("youtubeIdentity", () => {
  it("reads the v param on /watch", () => {
    expect(youtubeIdentity("https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=5s")?.external_id).toBe("dQw4w9WgXcQ");
  });
  it("reads youtu.be short links", () => {
    expect(youtubeIdentity("https://youtu.be/dQw4w9WgXcQ")?.external_id).toBe("dQw4w9WgXcQ");
  });
  it("reads /shorts/ and /embed/", () => {
    expect(youtubeIdentity("https://www.youtube.com/shorts/abc123")?.external_id).toBe("abc123");
    expect(youtubeIdentity("https://www.youtube.com/embed/xyz789")?.external_id).toBe("xyz789");
  });
  it("returns the youtube source_kind", () => {
    expect(youtubeIdentity("https://youtu.be/x")?.source_kind).toBe("youtube");
  });
  it("returns null for non-video youtube pages and junk", () => {
    expect(youtubeIdentity("https://www.youtube.com/feed/subscriptions")).toBeNull();
    expect(youtubeIdentity("not a url")).toBeNull();
  });
});

describe("genericIdentity", () => {
  it("uses origin+path+search, dropping the fragment", () => {
    expect(genericIdentity("https://example.com/video/7?a=1#frag")?.external_id).toBe("https://example.com/video/7?a=1");
  });
  it("returns the web source_kind", () => {
    expect(genericIdentity("https://example.com/x")?.source_kind).toBe("web");
  });
  it("returns null for junk", () => {
    expect(genericIdentity("nope")).toBeNull();
  });
});

describe("youtubeIdentityFromState", () => {
  it("prefers the dataset video id over the URL", () => {
    const id = youtubeIdentityFromState("VIDID", "https://www.youtube.com/watch?v=URLID");
    expect(id?.external_id).toBe("VIDID");
    expect(id?.source_kind).toBe("youtube");
  });

  it("falls back to the URL when the id is empty or absent", () => {
    expect(youtubeIdentityFromState("", "https://www.youtube.com/watch?v=URLID")?.external_id).toBe("URLID");
    expect(youtubeIdentityFromState(undefined, "https://www.youtube.com/watch?v=URLID")?.external_id).toBe("URLID");
  });

  it("returns null when neither id nor a video URL is present", () => {
    expect(youtubeIdentityFromState(undefined, "https://www.youtube.com/feed/subscriptions")).toBeNull();
  });
});
