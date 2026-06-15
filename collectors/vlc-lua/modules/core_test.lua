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

print(string.format("%d tests, %d failures", tests, failures))
os.exit(failures == 0 and 0 or 1)
