import { defineConfig } from "tsdown";

export default defineConfig({
  deps: {
    neverBundle: [
      "drizzle-kit",
      /^drizzle-kit\//,
      "@electric-sql/pglite",
      "drizzle-orm/pglite",
    ],
  },
  dts: { build: true, incremental: true },
  format: ["esm"],
  entry: ["./src/index.ts"],
  treeshake: true,
});
