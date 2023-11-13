import { defineConfig } from "vitest/dist/config.js";

export default defineConfig({
  test: {
    globals: true,
    setupFiles: ["dotenv/config"], //this line,
  },
});
