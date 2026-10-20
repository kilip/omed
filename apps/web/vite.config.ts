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
  optimizeDeps: {
    include: [
      "@ant-design/icons",
      "@tanstack/react-query",
      "antd",
      "antd/locale/en_US",
      "antd/locale/id_ID",
      "dayjs",
      "i18next",
      "openapi-fetch",
      "react",
      "react-dom",
      "react-i18next",
      "react-router",
    ],
  },
  server: {
    port: 3000,
  },
});
