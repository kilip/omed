import path from "node:path";
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    coverage: {
      provider: "istanbul",
      reporter: ["text", "lcov"],
    },
  },
  resolve: {
    alias: {
      "~": path.resolve(import.meta.dirname, "./app"),
    },
  },
});
