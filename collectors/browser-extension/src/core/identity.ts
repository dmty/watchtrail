export interface Identity {
  source_kind: "youtube" | "web";
  external_id: string;
  meta?: Record<string, unknown>;
}

export function youtubeIdentity(rawUrl: string): Identity | null {
  let u: URL;
  try {
    u = new URL(rawUrl);
  } catch {
    return null;
  }
  let id: string | null = null;
  if (u.hostname === "youtu.be") {
    id = u.pathname.slice(1).split("/")[0] || null;
  } else if (u.pathname === "/watch") {
    id = u.searchParams.get("v");
  } else {
    const m = u.pathname.match(/^\/(shorts|embed)\/([^/]+)/);
    if (m) id = m[2];
  }
  if (!id) return null;
  return { source_kind: "youtube", external_id: id };
}

export function youtubeIdentityFromState(
  videoId: string | null | undefined,
  url: string,
): Identity | null {
  if (videoId) return { source_kind: "youtube", external_id: videoId };
  return youtubeIdentity(url);
}

export function genericIdentity(rawUrl: string): Identity | null {
  let u: URL;
  try {
    u = new URL(rawUrl);
  } catch {
    return null;
  }
  return { source_kind: "web", external_id: u.origin + u.pathname + u.search };
}
