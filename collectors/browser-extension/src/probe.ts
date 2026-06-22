export type ProbeResult =
  | { state: "not-configured" }
  | { state: "reachable"; status: number }
  | { state: "unreachable"; reason: string };

export interface ProbeOptions {
  timeoutMs?: number;
  fetchFn?: typeof fetch;
}

export async function probeCore(
  coreUrl: string,
  token: string,
  opts: ProbeOptions = {},
): Promise<ProbeResult> {
  const trimmed = coreUrl.trim();
  if (trimmed === "") return { state: "not-configured" };

  const timeoutMs = opts.timeoutMs ?? 1500;
  const f = opts.fetchFn ?? fetch;
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), timeoutMs);

  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  try {
    const url = trimmed.replace(/\/+$/, "") + "/healthz";
    const res = await f(url, { method: "GET", headers, signal: ctrl.signal });
    if (res.status >= 200 && res.status < 300) {
      return { state: "reachable", status: res.status };
    }
    return { state: "unreachable", reason: `HTTP ${res.status}` };
  } catch (e) {
    const reason = e instanceof Error ? e.message : String(e);
    return { state: "unreachable", reason };
  } finally {
    clearTimeout(timer);
  }
}
