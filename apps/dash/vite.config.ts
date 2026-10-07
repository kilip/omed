import { reactRouter } from "@react-router/dev/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [tailwindcss(), reactRouter()],
  optimizeDeps: {
    include: [
      "react",
      "react/jsx-runtime",
      "react/jsx-dev-runtime",
      "react-dom",
      "react-dom/client",
      "react-router",
      "react-router/dom",
      "antd",
      "@ant-design/icons",
      "antd/locale/en_US",
      "antd/locale/id_ID",
      "better-auth/client/plugins",
      "better-auth/react",
      "i18next",
      "react-i18next",
      "i18next-browser-languagedetector",
      "@t3-oss/env-core",
      "zod",
    ],
  },
  resolve: {
    tsconfigPaths: true,
  },
});
