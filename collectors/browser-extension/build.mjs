import { build } from "esbuild";
import { copyFileSync } from "node:fs";

await build({
  entryPoints: ["src/background.ts", "src/content.ts", "src/popup.ts", "src/youtube-audio.ts"],
  bundle: true,
  format: "iife",
  outdir: "dist",
  target: "chrome111",
  logLevel: "info",
});

copyFileSync("src/popup.html", "dist/popup.html");
