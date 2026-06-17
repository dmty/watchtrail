import type { Identity } from "./identity";

export type EventType = "start" | "progress" | "pause" | "resume" | "stop" | "seek";

export interface MediaInfo {
  external_id: string;
  kind?: string;
  title?: string;
  url_or_path?: string;
  duration_seconds?: number;
  language?: string;
}

export interface WatchEvent {
  v: 1;
  event_id: string;
  type: EventType;
  occurred_at: string;
  source_kind: string;
  source_instance?: string;
  media: MediaInfo;
  position_seconds?: number;
  meta?: Record<string, unknown>;
}

export interface BuildInput {
  type: EventType;
  identity: Identity;
  position_seconds?: number;
  title?: string;
  duration_seconds?: number;
  language?: string;
  url_or_path?: string;
  kind?: string;
  meta?: Record<string, unknown>;
  source_instance?: string;
}

export function buildEvent(
  input: BuildInput,
  now: () => string = () => new Date().toISOString(),
  newId: () => string = () => crypto.randomUUID(),
): WatchEvent {
  const media: MediaInfo = { external_id: input.identity.external_id };
  if (input.kind !== undefined) media.kind = input.kind;
  if (input.title !== undefined) media.title = input.title;
  if (input.url_or_path !== undefined) media.url_or_path = input.url_or_path;
  if (input.duration_seconds !== undefined) media.duration_seconds = input.duration_seconds;
  if (input.language !== undefined) media.language = input.language;

  const ev: WatchEvent = {
    v: 1,
    event_id: newId(),
    type: input.type,
    occurred_at: now(),
    source_kind: input.identity.source_kind,
    media,
  };
  if (input.position_seconds !== undefined) ev.position_seconds = input.position_seconds;
  if (input.source_instance !== undefined) ev.source_instance = input.source_instance;
  const meta = input.meta ?? input.identity.meta;
  if (meta !== undefined) ev.meta = meta;
  return ev;
}
