import {
  index,
  prefix,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

export default prefix("agent", [
  index("modules/agent/routes/index.tsx"),
  route("providers", "modules/agent/routes/providers.tsx"),
]) satisfies RouteConfigEntry[];
