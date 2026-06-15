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

return M
