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

return M
