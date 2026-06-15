package.path = "./?.lua;" .. package.path
local M = require("core")

local tests, failures = 0, 0
local function check(cond, msg)
  tests = tests + 1
  if not cond then failures = failures + 1; print("FAIL: " .. tostring(msg)) end
end
local function eq(got, want, msg)
  check(got == want, (msg or "eq") .. "  got=" .. tostring(got) .. " want=" .. tostring(want))
end

-- hash: deterministic djb2-32, lowercase 8-hex. Golden vectors (hand-computed,
-- verified with luajit): djb2("")=5381=0x00001505 ; djb2("a")=177670=0x0002b606
eq(M.hash(""), "00001505", "hash empty")
eq(M.hash("a"), "0002b606", "hash a")
check(#M.hash("file:///some/long/path.mkv") == 8, "hash is 8 hex chars")
check(M.hash("x") == M.hash("x"), "hash deterministic")
check(M.hash("x") ~= M.hash("y"), "hash distinguishes inputs")

-- external_id: scheme prefix + hash of the full uri
eq(M.external_id("file:///x.mkv"), "file:" .. M.hash("file:///x.mkv"), "external_id file")
eq(M.external_id("http://h/v"), "url:" .. M.hash("http://h/v"), "external_id url")

-- json_encode: structure + escaping. Keys are sorted for deterministic output.
local prog = {
  v = 1, type = "progress", source_kind = "vlc", source_instance = "laptop",
  occurred_at = "2026-06-15T20:00:30Z", position_seconds = 1342,
  media = { external_id = "file:9a1f" }, meta = {},
}
local j = M.json_encode(prog)
check(j:sub(1, 1) == "{" and j:sub(-1) == "}", "json is an object")
for _, sub in ipairs({
  '"v":1', '"type":"progress"', '"source_kind":"vlc"', '"source_instance":"laptop"',
  '"occurred_at":"2026-06-15T20:00:30Z"', '"position_seconds":1342',
  '"media":{"external_id":"file:9a1f"}', '"meta":{}',
}) do
  check(j:find(sub, 1, true) ~= nil, "json contains " .. sub)
end

-- escaping: quote, backslash, newline
local esc = M.json_encode({ title = 'a"b\\c\nd' })
check(esc:find('"a\\"b\\\\c\\nd"', 1, true) ~= nil, "json escapes quote/backslash/newline")

-- nested media object with multiple fields, and a fractional number
local st = M.json_encode({ media = { external_id = "file:x", kind = "video", duration_seconds = 7500 }, position_seconds = 1342.5 })
check(st:find('"media":{', 1, true) ~= nil, "nested media is an object")
check(st:find('"duration_seconds":7500', 1, true) ~= nil, "integer field no decimal")
check(st:find('"position_seconds":1342.5', 1, true) ~= nil, "fractional field preserved")

-- non-finite numbers must not produce invalid JSON
eq(M.json_encode({ x = math.huge }), '{"x":null}', "infinity -> null")
eq(M.json_encode({ x = -math.huge }), '{"x":null}', "-infinity -> null")
eq(M.json_encode({ x = 0/0 }), '{"x":null}', "nan -> null")

print(string.format("%d tests, %d failures", tests, failures))
os.exit(failures == 0 and 0 or 1)
