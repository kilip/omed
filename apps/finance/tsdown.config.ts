import { defineConfig } from "tsdown";

export default defineConfig({
  dts: { build: true, incremental: true },
  format: ["esm"],
  entry: ["./client/index.ts"],
  treeshake: true,
});
