-- WatchTrail VLC interface module (VLC 3.x). The bug-prone logic lives in
-- modules/core.lua; this file is the thin binding to VLC.
-- Install: copy this file to VLC's lua/intf/ and modules/core.lua to lua/intf/modules/.
-- Enable: vlc --extraintf luaintf --lua-intf watchtrail --lua-config "watchtrail={port=8766}"
local core = require "core"

-- config: VLC populates the global `config` table from --lua-config.
local cfg      = config or {}
local PORT     = cfg.port or 8766
local TOKEN    = cfg.token or ""
local INSTANCE = cfg.instance or "vlc"
local INTERVAL = (cfg.interval or 30) * 1000000 -- microseconds for mwait
local QUEUE    = vlc.config.userdatadir() .. "/watchtrail-queue.jsonl"
local QUEUE_CAP = 10000

-- new_id: unique per event, generated once (re-sent unchanged on retry so the
-- core dedups by event_id).
math.randomseed(os.time())
local counter = 0
local function new_id()
  counter = counter + 1
  return string.format("vlc-%x-%x-%x", os.time(), counter, math.random(0, 0xffffff))
end
local opts = { new_id = new_id, source_instance = INSTANCE }

-- read_snapshot: the only VLC-3.x-specific code. Returns plain values for core.step.
local function read_snapshot()
  local snap = { status = "stopped", occurred_at = os.date("!%Y-%m-%dT%H:%M:%SZ") }
  local item = vlc.input.item()
  if not item then return snap end
  snap.status = vlc.playlist.status() or "stopped"
  snap.uri = item:uri()
  snap.name = item:name()
  snap.duration = item:duration() -- seconds (float), < 0 if unknown
  local input = vlc.object.input()
  if input then
    local t = vlc.var.get(input, "time") -- microseconds
    if t then snap.position = t / 1000000 end
  end
  return snap
end

local function read_queue()
  local lines = {}
  local f = io.open(QUEUE, "r")
  if f then
    for line in f:lines() do if #line > 0 then lines[#lines + 1] = line end end
    f:close()
  end
  return lines
end

local function write_queue(lines)
  local start = 1
  if #lines > QUEUE_CAP then start = #lines - QUEUE_CAP + 1 end -- keep newest
  local f = io.open(QUEUE, "w")
  if f then
    for i = start, #lines do f:write(lines[i] .. "\n") end
    f:close()
  end
end

-- push: one connection drains the disk backlog + the current lines. The queue is
-- cleared only if every byte was sent; on connect or send failure the current
-- lines are appended to the preserved backlog and retried next tick (the core
-- dedups by event_id, so re-sends are harmless).
local function push(lines)
  local fd = vlc.net.connect_tcp("127.0.0.1", PORT)
  if fd and fd >= 0 then
    local ok = true
    local function send_line(s)
      if not ok then return end
      local n = vlc.net.send(fd, s)
      if not n or n < #s then ok = false end
    end
    if TOKEN ~= "" then send_line(TOKEN .. "\n") end
    for _, l in ipairs(read_queue()) do send_line(l .. "\n") end
    for _, l in ipairs(lines) do send_line(l .. "\n") end
    vlc.net.close(fd)
    if ok then
      write_queue({}) -- everything delivered
      return
    end
  end
  -- connect failed, or a send failed mid-stream: buffer current lines onto the
  -- existing backlog (capped, newest kept) and retry on a later tick.
  local q = read_queue()
  for _, l in ipairs(lines) do q[#q + 1] = l end
  write_queue(q)
end

-- main loop
local state = {}
while true do
  local ok, snap = pcall(read_snapshot)
  if ok and snap then
    local events
    events, state = core.step(state, snap, opts)
    if #events > 0 then
      local lines = {}
      for _, ev in ipairs(events) do lines[#lines + 1] = core.json_encode(ev) end
      pcall(push, lines)
    end
  end
  vlc.misc.mwait(vlc.misc.mdate() + INTERVAL)
end
