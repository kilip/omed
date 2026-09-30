import { defineProject } from "vitest/config";
import "dotenv/config";

export default defineProject({
  resolve: {
    tsconfigPaths: true,
  },
});
