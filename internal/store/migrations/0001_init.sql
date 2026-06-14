CREATE TABLE media_item (
    id               TEXT PRIMARY KEY,
    source_kind      TEXT NOT NULL,
    external_id      TEXT NOT NULL,
    identity_key     TEXT NOT NULL UNIQUE,
    kind             TEXT NOT NULL DEFAULT 'unknown',
    title            TEXT,
    url_or_path      TEXT,
    duration_seconds INTEGER,
    metadata         TEXT,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL,
    deleted_at       TEXT
);

CREATE TABLE watch_event (
    id               TEXT PRIMARY KEY,
    media_item_id    TEXT NOT NULL REFERENCES media_item(id),
    source_kind      TEXT NOT NULL,
    source_instance  TEXT,
    type             TEXT NOT NULL,
    position_seconds REAL,
    occurred_at      TEXT NOT NULL,
    received_at      TEXT NOT NULL,
    session_id       TEXT,
    raw              TEXT NOT NULL
);

CREATE TABLE watch_session (
    id                   TEXT PRIMARY KEY,
    media_item_id        TEXT NOT NULL REFERENCES media_item(id),
    source_kind          TEXT NOT NULL,
    source_instance      TEXT,
    started_at           TEXT NOT NULL,
    ended_at             TEXT NOT NULL,
    watched_seconds      INTEGER NOT NULL DEFAULT 0,
    max_position_seconds REAL,
    completed            INTEGER NOT NULL DEFAULT 0,
    event_count          INTEGER NOT NULL DEFAULT 0,
    created_at           TEXT NOT NULL,
    updated_at           TEXT NOT NULL,
    deleted_at           TEXT
);

CREATE INDEX idx_watch_event_media_occurred ON watch_event(media_item_id, occurred_at);
CREATE INDEX idx_watch_event_session        ON watch_event(session_id);
CREATE INDEX idx_watch_session_started      ON watch_session(started_at);
CREATE INDEX idx_watch_session_media        ON watch_session(media_item_id);
