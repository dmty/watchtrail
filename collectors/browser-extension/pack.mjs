import { cpSync, copyFileSync, mkdirSync, rmSync } from "node:fs";
import { execFileSync } from "node:child_process";

// Build the bundles (reuses build.mjs: esbuild + popup.html copy), then stage a
// clean loadable extension (manifest.json + dist/ only — no src/node_modules)
// and zip it store-ready, with manifest.json at the archive root.
await import("./build.mjs");

const STAGE = "pkg/watchtrail-extension";
rmSync("pkg", { recursive: true, force: true });
mkdirSync(STAGE, { recursive: true });
copyFileSync("manifest.json", `${STAGE}/manifest.json`);
cpSync("dist", `${STAGE}/dist`, { recursive: true });
cpSync("icons", `${STAGE}/icons`, {
  recursive: true,
  filter: (src) => !src.endsWith(".svg"),
});

execFileSync("zip", ["-r", "-q", "../watchtrail-extension.zip", "."], {
  cwd: STAGE,
});

console.log(
  "Staged   pkg/watchtrail-extension/      (load-unpacked or --pack-extension source)",
);
console.log(
  "Zipped   pkg/watchtrail-extension.zip   (Chrome Web Store upload)",
);
