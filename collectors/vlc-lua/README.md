# WatchTrail — VLC collector

A VLC **interface module** (Lua) that reports what you watch in VLC to a running
WatchTrail core. It samples playback every 30 s and pushes watch events over the
core's localhost TCP line protocol. Targets **VLC 3.x**.

## Files

- `watchtrail.lua` — the interface module (the VLC binding).
- `modules/core.lua` — pure logic (identity hash, event encoder, state machine).
- `modules/core_test.lua` — unit tests (run with luajit).

## Install

Copy both files into VLC's per-user Lua directories (no admin rights needed):

| OS | `watchtrail.lua` -> | `core.lua` -> |
|----|--------------------|--------------|
| Linux | `~/.local/share/vlc/lua/intf/` | `~/.local/share/vlc/lua/intf/modules/` |
| macOS | `~/Library/Application Support/org.videolan.vlc/lua/intf/` | `.../org.videolan.vlc/lua/intf/modules/` |
| Windows | `%APPDATA%\vlc\lua\intf\` | `%APPDATA%\vlc\lua\intf\modules\` |

Example (macOS):

```sh
DST="$HOME/Library/Application Support/org.videolan.vlc/lua/intf"
mkdir -p "$DST/modules"
cp watchtrail.lua "$DST/"
cp modules/core.lua "$DST/modules/"
```

## Enable

Start VLC with the interface module (replace the port/token to match your core):

```sh
vlc --extraintf luaintf --lua-intf watchtrail \
    --lua-config "watchtrail={port=8766,token='',instance='laptop-vlc'}"
```

To enable it permanently, set `extraintf=luaintf` and `lua-intf=watchtrail` in
VLC Preferences -> Show all -> Interface -> Extra interface modules.

### Config keys (`--lua-config "watchtrail={...}"`)

| Key | Default | Meaning |
|-----|---------|---------|
| `port` | `8766` | core TCP line-listener port |
| `token` | `""` | TCP handshake token (sent as the first line when set) |
| `instance` | `"vlc"` | `source_instance` stamped on every event |
| `interval` | `30` | seconds between playback samples |

## Test

Unit tests (pure logic):

```sh
cd modules && luajit core_test.lua   # exit 0 = pass
```

Manual end-to-end:

1. Build and run the core: `go build -o watchtrail ./cmd/watchtrail && ./watchtrail serve`
   (use a config with `token=""` for this smoke test).
2. Install the two files (above) and start VLC with the `--lua-config "watchtrail={port=8766}"` flags.
3. Play a file for ~1 minute; pause and resume; stop.
4. Run `./watchtrail recent` — the VLC session should appear with the right
   title, a non-zero watched time, and a completion mark if you watched >=90%.

## Notes / limitations

- VLC opened before the core is fine: events buffer to
  `<vlc-userdir>/watchtrail-queue.jsonl` and flush on the next reachable tick
  (capped at 10 000 lines, newest kept).
- A hard VLC crash may drop the final `stop`; the core's sessionizer closes the
  session on its gap timeout anyway.
- VLC 4.x (which moves to the `vlc.player.*` API) is not yet supported; only the
  `read_snapshot` function in `watchtrail.lua` would need a small shim.
