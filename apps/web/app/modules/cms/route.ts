import {
  index,
  prefix,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

export default prefix("cms", [
  index("modules/cms/routes/index.tsx"),
  route("posts", "modules/cms/routes/posts.tsx"),
  route("pages", "modules/cms/routes/pages.tsx"),
  route("media", "modules/cms/routes/media.tsx"),
]) satisfies RouteConfigEntry[];
