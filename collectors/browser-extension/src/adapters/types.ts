import type { Identity } from "../core/identity";

export interface MediaDetails {
  title?: string;
  duration_seconds?: number;
  url_or_path?: string;
  kind?: string;
  meta?: Record<string, unknown>;
}

export interface Adapter {
  /** True when this adapter applies to the current page. */
  matches(): boolean;
  /** The current media's identity, or null if the page has none. */
  identity(): Identity | null;
  /** Per-video metadata read live from the element/page. */
  details(video: HTMLVideoElement): MediaDetails;
}
