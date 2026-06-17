#!/usr/bin/env bash
# Pack a self-hosted .crx for the enterprise-policy force-install path (no dev
# mode, no Web Store). Reuses `npm run pack` to stage, then drives Chrome's
# packer. Keep the generated key.pem stable — it fixes the extension ID.
set -euo pipefail

CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
DIR="$PWD/pkg/watchtrail-extension"
KEY="$PWD/key.pem"

[ -x "$CHROME" ] || { echo "Chrome not found at: $CHROME (override with CHROME=...)" >&2; exit 1; }

npm run pack

if [ -f "$KEY" ]; then
  "$CHROME" --pack-extension="$DIR" --pack-extension-key="$KEY"
else
  "$CHROME" --pack-extension="$DIR"
  echo "First pack: generated pkg/watchtrail-extension.pem — move it to ./key.pem and keep it."
  echo "  mv pkg/watchtrail-extension.pem key.pem"
fi

echo "crx at pkg/watchtrail-extension.crx"
echo "Get the extension ID from chrome://extensions (load the .crx once), then put it in updates.xml + the policy."
