# WatchTrail

A lightweight, extensible system for tracking what you watch — across local players (starting with VLC) and streaming services (starting with YouTube via a browser extension) — into a single local history you own, with dashboards on top.

> **Codename:** `watchtrail` is a working name. Rename freely; nothing in the design depends on it.

## What this is

Most "watch history" lives trapped inside individual apps: VLC forgets, YouTube's history is buried and platform-locked, and nothing unifies them. WatchTrail is a small local-first service that ingests **watch events** from many **sources** through one normalized pipeline, stores them in a single database you control, and exposes a dashboard and query API over the result.

## Design priorities (in order)

1. **Low steady-state cost.** Event-driven (push), not polling. Near-zero idle footprint. A single static Go binary plus a file database.
2. **Extensibility.** Adding a new source (a player, a streaming service, a podcast app) should be a small, well-defined unit of work that touches no core logic.
3. **Local-first & ownership.** Your history is a file on your disk. Sync is optional and designed-for, not required.
4. **Incremental.** Each milestone is independently useful. VLC logging works before the dashboard exists; the dashboard works before the browser extension exists.

## Current decisions

| Area | Decision |
|------|----------|
| Platform | Cross-platform from day one (macOS / Linux / Windows) |
| Backend | Single Go binary, stdlib-first |
| Storage | SQLite now, with a sync seam designed into the schema |
| Dashboard | Go server-rendered (htmx + templates) — no JS build step |
| Ingestion | Canonical HTTP/JSON; optional ultra-light TCP line protocol for the VLC Lua collector |
| First sources | VLC (Lua extension) → then browser extension (YouTube + generic) |

## Status

Planning — no code yet. Design documentation, ADRs, the roadmap, and implementation
plans are kept locally (outside version control) and are not part of this repository
yet. Code lands milestone by milestone (VLC ingestion first); docs graduate into the
repo as the surfaces they describe become real.
