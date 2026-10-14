import { index, type RouteConfigEntry, route } from "@react-router/dev/routes";

export default [
  route("user", "modules/user/layout.tsx", [
    index("modules/user/routes/users.tsx"),
  ]),
] satisfies RouteConfigEntry[];
