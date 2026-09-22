import { defineConfig } from "vitest/config";

export default defineConfig({
  //plugins: [react()],
  test: {
    //environment: "jsdom",
  },
  resolve: {
    tsconfigPaths: true,
  },
});
