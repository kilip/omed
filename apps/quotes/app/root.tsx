import { Theme } from "@radix-ui/themes";
import cssRadixHref from "@radix-ui/themes/styles.css";
import type { LinksFunction } from "@remix-run/node";
import { Outlet } from "@remix-run/react";
import Document from "./pkg/core/components/Document";
import cssTailwindHref from "./pkg/core/styles/tailwind.css";

export const links: LinksFunction = () => [
  { rel: "stylesheet", href: cssTailwindHref },
  { rel: "stylesheet", href: cssRadixHref },
];

export default function App() {
  return (
    <Document>
      <Theme>
        <Outlet />
      </Theme>
    </Document>
  );
}
