// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import react from "@astrojs/react";

// https://astro.build/config
export default defineConfig({
  integrations: [
    starlight({
      title: "Omed",
      tableOfContents: false,
      pagefind: false,
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/kilip/omed",
        },
      ],
      sidebar: [
        {
          label: "Guides",
          items: [
            // Each item here is one entry in the navigation menu.
            { label: "Example Guide", slug: "guides/example" },
          ],
        },
        {
          label: "Reference",
          autogenerate: { directory: "reference" },
        },
      ],
    }),
    react(),
  ],
});
