import { flatRoutes } from "remix-flat-routes";

/** @type {import('@remix-run/dev').AppConfig} */
export default {
  ignoredRouteFiles: ["**/.*"],
  // appDirectory: "app",
  // assetsBuildDirectory: "public/build",
  // publicPath: "/build/",
  // serverBuildPath: "build/index.js",
  routes: async (defineRoutes) => {
    const dirs = ["pkg/core/routes", "pkg/auth/routes"];

    return flatRoutes(dirs, defineRoutes);
  },
  serverDependenciesToBundle: [/^@radix-ui.*/, /^@omed.*/, /^lodash.*/],
};
