import { index, type RouteConfigEntry, route } from "@react-router/dev/routes";

export default [
  route("core", "modules/core/layout.tsx", [
    index("modules/core/routes/dashboard.tsx"),
    route("workspaces", "modules/core/routes/workspaces.tsx"),
    route("members", "modules/core/routes/members.tsx"),
    route("invitations", "modules/core/routes/invitations.tsx"),
  ]),
] satisfies RouteConfigEntry[];
