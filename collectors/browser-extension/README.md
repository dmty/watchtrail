# WatchTrail — Browser extension collector

A Chrome/Chromium MV3 extension that records what you watch in the browser and
posts it to your local WatchTrail core. Local-only: events go to `127.0.0.1`
and nowhere else.

## Build

```
cd collectors/browser-extension
npm install
npm run build
```

This produces `dist/` (the loadable extension output).

## Load in Chrome

1. Open `chrome://extensions`.
2. Enable **Developer mode** (top right).
3. **Load unpacked** → select the `collectors/browser-extension/` directory
   (the one containing `manifest.json`).

## Package (no Developer mode)

`npm run pack` builds and stages a clean extension (just `manifest.json` + `dist/`)
into `pkg/watchtrail-extension/` and zips it to `pkg/watchtrail-extension.zip`.
Modern Chrome refuses to install a plain `.crx` outside the Web Store, so there
are two real ways to run without the Developer-mode toggle:

### A. Self-host + enterprise policy (local, free — recommended for personal use)

Force-install your own `.crx` via Chrome's `ExtensionSettings` policy. No store,
nothing leaves the machine, and it removes the "disable developer extensions"
startup nag.

1. `npm run make-crx` — packs `pkg/watchtrail-extension.crx` and, on first run,
   a key. Move that key to `key.pem` and keep it (it fixes the extension ID);
   re-run `make-crx` to repack with the same ID.
2. Load the `.crx` once at `chrome://extensions` to read the **extension ID**.
3. Fill in `updates.xml`: the `appid` (extension ID) and the absolute `codebase`
   path to the `.crx`. Bump its `version` (and `manifest.json`'s) on each update.
4. Set the policy (macOS), then relaunch Chrome and confirm at `chrome://policy`:

   ```
   defaults write com.google.Chrome ExtensionSettings -dict-add "<EXTENSION_ID>" \
     '{"installation_mode":"normal_installed","update_url":"file:///ABSOLUTE/PATH/TO/collectors/browser-extension/updates.xml"}'
   ```

   (Windows: the `Software\Policies\Google\Chrome\ExtensionSettings` registry
   key; Linux: a JSON policy under `/etc/opt/chrome/policies/managed/`.)

### B. Chrome Web Store, unlisted (simplest install, auto-updates — $5 one-time)

Upload `pkg/watchtrail-extension.zip` to the Web Store dashboard, set visibility
to **Unlisted** (installable by link, not searchable) or **Private** (your
Workspace org only). One-time $5 developer registration plus a review pass.

`pkg/`, `*.crx`, and `*.pem` are gitignored — never commit the private key.

## Configure

Click the WatchTrail toolbar icon:

- **Enabled** — master on/off.
- **Core URL** — defaults to `http://127.0.0.1:8765` (the core's default HTTP
  address; change it if you run the core elsewhere).
- **Token** — paste the same value as the core's `token` config / `WATCHTRAIL_TOKEN`.
  Leave blank if the core runs without a token (loopback-only).
- **Track this site** — adds the current site to the allowlist for the generic
  `<video>` adapter (asks for that site's permission). **Stop** removes it.

YouTube is tracked out of the box; other sites are tracked only after you add
them.

## What it tracks

- **YouTube** (`youtube` source): video id identity, title, channel, duration.
- **Allowlisted sites** (`web` source): any HTML5 `<video>`, page-URL identity.

Both appear in the dashboard alongside VLC, with `youtube` / `web` / `vlc`
source badges.

## Miniplayer and fullscreen

YouTube identity comes from `data-wt-video-id` on `<html>`, written by a
MAIN-world probe that reads the player's real `video_id`. This means playback
is tracked continuously across watch-page ↔ miniplayer ↔ fullscreen transitions
without missing a `start` or `stop`. The page URL is only a fallback when the
probe hasn't run yet or on non-YouTube pages.

## Tests

```
npm test
```

Runs the vitest suite over the pure `core/` modules (identity, session/throttle,
queue, flush) and config.

## Limitations

- Chrome/Chromium only (Firefox is a future port; the code is structured for it).
- The generic adapter's identity is the page URL minus the fragment — coarse by
  design; sites that reuse one URL for many videos will collapse them.
- MV3 may terminate the background worker; the event queue is persisted to
  `chrome.storage.local` and flushed on the next event, alarm tick, or startup,
  so nothing is lost while the core is briefly unreachable.
- A hard browser crash mid-video may drop the final `stop`; the core's
  sessionizer closes the session on timeout anyway.
