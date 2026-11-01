import {
  index,
  prefix,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

export default prefix("agent", [
  index("modules/agent/routes/index.tsx"),
  route("providers", "modules/agent/routes/providers.tsx"),
  route("providers/new", "modules/agent/routes/provider-new.tsx"),
  route("providers/:id/edit", "modules/agent/routes/provider-edit.tsx"),
]) satisfies RouteConfigEntry[];
