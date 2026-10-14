import { reactRouter } from "@react-router/dev/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import istanbul from "vite-plugin-istanbul";

export default defineConfig({
  plugins: [
    tailwindcss(),
    reactRouter(),
    istanbul({
      include: ["app/**"],
      exclude: ["node_modules", "test/"],
      extension: [".js", ".ts", ".tsx"],
      requireEnv: true,
    }),
  ],
  resolve: {
    tsconfigPaths: true,
  },
  build: {
    sourcemap: true,
  },
  server: {
    port: 3000,
  },
});
