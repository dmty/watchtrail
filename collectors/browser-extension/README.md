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
