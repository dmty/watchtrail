import { build } from "esbuild";

await build({
  entryPoints: ["src/background.ts"],
  bundle: true,
  format: "iife",
  outdir: "dist",
  target: "chrome110",
  logLevel: "info",
});
