import { index, type RouteConfigEntry, route } from "@react-router/dev/routes";

export default [
  route("sys", "modules/sys/layout.tsx", [
    index("modules/sys/routes/overview.tsx"),
    route("workspaces", "modules/sys/routes/workspaces.tsx"),
    route("members", "modules/sys/routes/members.tsx"),
    route("invitations", "modules/sys/routes/invitations.tsx"),
  ]),
] satisfies RouteConfigEntry[];
