import {
  index,
  layout,
  type RouteConfig,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

// Each module declares its own routes in `modules/<name>/route.ts`.
const moduleRoutes = import.meta.glob<{ default: RouteConfigEntry[] }>(
  "./modules/*/route.ts",
  { eager: true },
);

export default [
  route("login", "routes/login.tsx"),
  route("invite/:token", "routes/invite.tsx"),
  layout("routes/shell.tsx", [
    index("routes/launcher.tsx"),
    ...Object.values(moduleRoutes).flatMap((m) => m.default),
  ]),
] satisfies RouteConfig;
