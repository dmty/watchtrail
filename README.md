# WatchTrail

A local-first media-watch history tool. WatchTrail receives normalized "watch events" from any source (VLC, a browser extension, a script) and stores them in a single SQLite database you own.

**What it solves:** most media apps track nothing, or track on someone else's server. WatchTrail gives you one append-only log of what you watched, when, and how far you got — on your own disk, in a format that will outlast any app.

**Design priorities:**
- One static binary + one file — no daemon, no Docker, no cloud account required.
- Push, not poll — collectors send events; the server is idle otherwise.
- Adding a new source requires no changes to core logic.
- Raw events are stored verbatim so derived data (sessions, stats) can be recomputed as the logic improves.

---

## Requirements

Go 1.26+ to build. No cgo. Runs on macOS, Linux, and Windows.

---

## Build

```sh
go build -o watchtrail ./cmd/watchtrail
```

---

## Run

```sh
watchtrail serve
watchtrail serve -config watchtrail.toml
```

`-config` defaults to `watchtrail.toml`. A missing config file is fine — built-in defaults apply.

On startup, the server logs the HTTP and TCP addresses it is listening on. It binds loopback only. Shutdown cleanly with `SIGINT` or `SIGTERM`.

---

## Configuration

All keys are optional. Environment variables override the file.

| Key | Default | Env override | Description |
|-----|---------|--------------|-------------|
| `http_addr` | `127.0.0.1:8765` | `WATCHTRAIL_HTTP_ADDR` | HTTP ingest bind address |
| `tcp_addr` | `127.0.0.1:8766` | `WATCHTRAIL_TCP_ADDR` | TCP line-protocol bind address |
| `token` | `""` (auth disabled) | `WATCHTRAIL_TOKEN` | Shared bearer / handshake token |
| `db_path` | `watchtrail.db` | `WATCHTRAIL_DB_PATH` | SQLite database file |
| `session_gap_seconds` | `1800` | — | Reserved — future sessionizer |
| `completion_threshold` | `0.9` | — | Reserved — future sessionizer |
| `progress_cadence_seconds` | `30` | — | Reserved — collector cadence hint |

**Minimal `watchtrail.toml`:**

```toml
db_path = "~/.local/share/watchtrail/history.db"
token   = "change-me"
```

---

## Sending events

### HTTP (canonical)

```
POST http://127.0.0.1:8765/ingest
Authorization: Bearer <token>
Content-Type: application/json

<single event object, or a JSON array for a batch>
```

Responses:
- `202 Accepted` — single event stored
- `200 OK` with `{"accepted": n}` — batch stored
- `400` — invalid or unsupported event
- `401` — bad token

Re-sending the same `event_id` is a safe no-op (idempotent).

### TCP line protocol (for lightweight collectors)

Connect to `tcp_addr`. Write one JSON event per line, newline-terminated.

If a token is configured, the **first line must be the token** (handshake). Every subsequent line is a JSON event. Same pipeline as HTTP — validation, dedup, and storage are identical.

---

## The canonical event (v1)

One JSON object per event. Required fields:

| Field | Type | Description |
|-------|------|-------------|
| `v` | int | Protocol version — always `1` |
| `event_id` | UUID string | Collector-generated; the idempotency key |
| `type` | string | `start` / `progress` / `pause` / `resume` / `stop` / `seek` |
| `occurred_at` | ISO-8601 UTC | When the event happened |
| `source_kind` | string | e.g. `vlc`, `youtube` |
| `media.external_id` | string | Source-scoped media identity |

Optional: `source_instance`, `position_seconds`, `media.kind`, `media.title`, `media.url_or_path`, `media.duration_seconds`, `meta` (free-form object).

**Example:**

```json
{
  "v": 1,
  "event_id": "f3c1e0a2-1111-4abc-8def-000000000001",
  "type": "progress",
  "occurred_at": "2026-06-14T09:31:02Z",
  "source_kind": "vlc",
  "source_instance": "laptop-vlc",
  "media": {
    "external_id": "file:9a1f",
    "title": "Spirited Away",
    "duration_seconds": 7500
  },
  "position_seconds": 1342.0,
  "meta": { "rate": 1.0 }
}
```

---

## Project layout

```
cmd/watchtrail/     # binary entry point — the `serve` command
internal/
  config/           # config loading: defaults → TOML → env
  event/            # canonical event type, parsing, validation
  ingest/           # transport-agnostic pipeline + HTTP and TCP transports
  store/            # Repository interface, SQLite impl, embedded migrations
  ids/              # dependency-free UUID generator
```

**Data model:** three tables — `media_item` (deduplicated identity), `watch_event` (append-only facts; the source of truth), `watch_session` (derived; not yet populated). All rows carry UUID primary keys and `updated_at`/`deleted_at` so multi-device sync can be layered in later without a schema rewrite.

**Dependencies:** `modernc.org/sqlite` (pure-Go, no cgo) and `github.com/BurntSushi/toml`.

---

## Tests

```sh
go test ./...
```

---

## Status and roadmap

Working today: HTTP and TCP ingest, event validation, media identity deduplication, idempotent append-only storage, SQLite with embedded migrations.

Planned next (each independently useful, not yet built):

- **Sessionization** — derive watch sessions from raw events; replayable over all history
- **Read API + CLI** — query your history from the terminal
- **Dashboard** — server-rendered local web UI
- **VLC Lua collector** — sends events over TCP from inside VLC
- **Browser extension** — YouTube and other streaming sites

Because raw events are stored verbatim, the sessionizer and any future analytics can be recomputed over your full history whenever the logic improves.

---

## License

TBD
