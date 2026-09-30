import type { Config } from "@react-router/dev/config";

export default {
  // Config options...
  // Server-side render by default, to enable SPA mode set this to `false`
  ssr: false,
  // Disable prerendering — pure SPA served by nginx, no static shell needed
  prerender: false,
} satisfies Config;
