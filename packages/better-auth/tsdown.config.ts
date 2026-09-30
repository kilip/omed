import { defineConfig } from "tsdown";

export default defineConfig({
  deps: {
    neverBundle: [
      "drizzle-kit",
      "@electric-sql/pglite",
      "drizzle-orm/pglite",
      "@tursodatabase/database",
      "@libsql/client",
    ],
  },
  dts: { build: true, incremental: true },
  format: ["esm"],
  entry: ["./src/index.ts"],
  treeshake: true,
});
