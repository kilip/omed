import { defineConfig } from "tsdown";

export default defineConfig({
  deps: {
    neverBundle: ["drizzle-kit", "@electric-sql/pglite"],
  },
  dts: { build: true, incremental: true },
  format: ["esm"],
  entry: ["./src/index.ts"],
  treeshake: true,
});
