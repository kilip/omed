import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";
import "dotenv/config";

export default defineConfig({
  plugins: [react()],
  test: {
    //environment: "jsdom",
  },
  resolve: {
    tsconfigPaths: true,
  },
});
