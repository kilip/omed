import {
  index,
  prefix,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

export default prefix("cms", [
  index("modules/cms/routes/index.tsx"),
  route("posts", "modules/cms/routes/posts.tsx"),
  route("posts/new", "modules/cms/routes/post-new.tsx"),
  route("posts/:id", "modules/cms/routes/post.tsx"),
  route("pages", "modules/cms/routes/pages.tsx"),
  route("pages/new", "modules/cms/routes/page-new.tsx"),
  route("pages/:id", "modules/cms/routes/page.tsx"),
  route("media", "modules/cms/routes/media.tsx"),
]) satisfies RouteConfigEntry[];
