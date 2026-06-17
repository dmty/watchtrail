-- Pure WatchTrail collector logic: identity hashing, event encoding, and the
-- playback state machine. No vlc.*/io/net here, so it runs identically on VLC's
-- vanilla Lua 5.1 and on luajit, and is unit-testable in isolation.
-- Arithmetic only (no bitwise ops) — VLC's Lua 5.1 has no bit library.
local M = {}

-- hash returns a deterministic 8-char hex djb2-32 hash of s.
function M.hash(s)
  local h = 5381
  for i = 1, #s do
    h = (h * 33 + s:byte(i)) % 4294967296
  end
  return string.format("%08x", h)
end

-- external_id maps a VLC uri to a source-scoped identity: file uris get a
-- "file:" prefix, anything else "url:". The full uri is preserved in url_or_path.
function M.external_id(uri)
  local prefix = "url:"
  if uri:sub(1, 7) == "file://" then prefix = "file:" end
  return prefix .. M.hash(uri)
end

local function escape_str(s)
  s = s:gsub("\\", "\\\\"):gsub('"', '\\"')
  s = s:gsub("\n", "\\n"):gsub("\r", "\\r"):gsub("\t", "\\t")
  s = s:gsub("[%z\1-\31]", function(c) return string.format("\\u%04x", c:byte()) end)
  return '"' .. s .. '"'
end

local function encode(v)
  local t = type(v)
  if t == "string" then
    return escape_str(v)
  elseif t == "boolean" then
    return v and "true" or "false"
  elseif t == "number" then
    if v ~= v or v == math.huge or v == -math.huge then return "null" end
    if v == math.floor(v) and v < 1e15 and v > -1e15 then
      return string.format("%d", v)
    end
    return (string.format("%.6f", v):gsub("0+$", ""):gsub("%.$", ""))
  elseif t == "table" then
    local keys = {}
    for k in pairs(v) do keys[#keys + 1] = k end
    table.sort(keys)
    local parts = {}
    for _, k in ipairs(keys) do
      parts[#parts + 1] = escape_str(k) .. ":" .. encode(v[k])
    end
    return "{" .. table.concat(parts, ",") .. "}"
  end
  return "null"
end

-- json_encode serializes a canonical event table to a JSON object string. Object
-- keys are sorted so the output is deterministic (handy for tests and stable on
-- the wire). Handles nested objects (the media sub-table).
M.json_encode = encode

-- mk_event builds one canonical v1 event. full_media is true only for "start",
-- which must carry title/duration so the core can create the media item.
local function mk_event(etype, snap, opts, full_media)
  local media = { external_id = M.external_id(snap.uri) }
  if full_media then
    media.kind = "video"
    media.title = snap.name
    media.url_or_path = snap.uri
    if snap.duration and snap.duration > 0 then
      media.duration_seconds = math.floor(snap.duration + 0.5)
    end
  end
  if snap.language and snap.language ~= "" then
    media.language = snap.language
  end
  local meta = {}
  if snap.language and snap.language ~= "" then
    meta.audio_language_raw = snap.language
  end
  if snap.audio_language_label and snap.audio_language_label ~= "" then
    meta.audio_language_label = snap.audio_language_label
  end
  local ev = {
    v = 1,
    event_id = opts.new_id(),
    type = etype,
    occurred_at = snap.occurred_at,
    source_kind = "vlc",
    source_instance = opts.source_instance,
    media = media,
    meta = meta,
  }
  if snap.position ~= nil then ev.position_seconds = snap.position end
  return ev
end

-- step maps a playback snapshot (plus prior state) to zero or more events and
-- the updated state. Gap-only boundaries with soft stop: see the design.
function M.step(state, snap, opts)
  state = state or {}
  local events = {}
  local status = snap.status or "stopped"

  if status == "playing" then
    if snap.uri ~= state.last_uri then
      events[#events + 1] = mk_event("start", snap, opts, true)
      state.last_uri = snap.uri
    elseif state.last_status == "paused" then
      events[#events + 1] = mk_event("resume", snap, opts, false)
    else
      events[#events + 1] = mk_event("progress", snap, opts, false)
    end
  elseif status == "paused" then
    if state.last_status == "playing" and state.last_uri then
      events[#events + 1] = mk_event("pause", snap, opts, false)
    end
  else -- stopped
    if state.last_uri and state.last_status ~= "stopped" then
      local stopsnap = { uri = state.last_uri, position = snap.position, occurred_at = snap.occurred_at }
      events[#events + 1] = mk_event("stop", stopsnap, opts, false)
      state.last_uri = nil
    end
  end

  state.last_status = status
  return events, state
end

return M
