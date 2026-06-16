import { build } from "esbuild";
import { copyFileSync } from "node:fs";

await build({
  entryPoints: ["src/background.ts", "src/content.ts", "src/popup.ts"],
  bundle: true,
  format: "iife",
  outdir: "dist",
  target: "chrome110",
  logLevel: "info",
});

copyFileSync("src/popup.html", "dist/popup.html");
