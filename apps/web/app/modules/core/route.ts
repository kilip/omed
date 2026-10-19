import {
  index,
  prefix,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

export default prefix("core", [
  index("modules/core/routes/dashboard.tsx"),
  route("workspaces/new", "modules/core/routes/workspace-new.tsx"),
  route("members", "modules/core/routes/members.tsx"),
  route("members/new", "modules/core/routes/member-new.tsx"),
  route("members/:id", "modules/core/routes/member-edit.tsx"),
  route("invitations", "modules/core/routes/invitations.tsx"),
]) satisfies RouteConfigEntry[];
