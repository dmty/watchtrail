import type { EventType } from "./event";

export type NativeKind = "play" | "pause" | "seeked" | "ended" | "timeupdate" | "hide";

export const PROGRESS_INTERVAL_MS = 30_000;

export interface SessionState {
  started: boolean;
  lastProgressMs: number;
}

export function newSession(): SessionState {
  return { started: false, lastProgressMs: 0 };
}

export function step(
  state: SessionState,
  native: NativeKind,
  nowMs: number,
): { state: SessionState; type: EventType | null } {
  switch (native) {
    case "play":
      if (!state.started) {
        return { state: { started: true, lastProgressMs: nowMs }, type: "start" };
      }
      return { state, type: "resume" };
    case "pause":
      return state.started ? { state, type: "pause" } : { state, type: null };
    case "seeked":
      return state.started ? { state, type: "seek" } : { state, type: null };
    case "ended":
    case "hide":
      return state.started
        ? { state: { ...state, started: false }, type: "stop" }
        : { state, type: null };
    case "timeupdate":
      if (state.started && nowMs - state.lastProgressMs >= PROGRESS_INTERVAL_MS) {
        return { state: { ...state, lastProgressMs: nowMs }, type: "progress" };
      }
      return { state, type: null };
  }
}

export function switchMedia(
  state: SessionState,
  currentId: string | null,
  newId: string,
  nowMs: number,
): { state: SessionState; currentId: string; close: EventType | null } {
  if (currentId === newId) return { state, currentId: newId, close: null };
  const close = state.started ? step(state, "hide", nowMs).type : null;
  return { state: newSession(), currentId: newId, close };
}
