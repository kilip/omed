# Omed brand guidelines

Style: **modern, glassmorphism, compact, warm**. Component base: Ant Design. Primary color: **terracotta**.

Related files:

- `apps/dash/app/theme/tokens.css`: CSS variables + glass utilities
- `apps/dash/app/theme/antd.ts`: `getAntdTheme(mode)` for `ConfigProvider`

## 1. Principles

1. **Warm, not cold.** Neutrals always lean brown/cream, never blue-gray.
2. **Glass in moderation.** Blur is for surfaces (cards, modals, sider, header), not for everything.
3. **Dense but breathable.** Compact: small controls, 4/8/12/16 spacing, with a clear hierarchy.
4. **One accent per screen.** Terracotta is only for the main action and active states.
5. **Legible first, pretty second.** Small text must pass AA contrast (4.5:1).

## 2. Color

### Brand and action

| Token | Light | Dark | Used for |
| --- | --- | --- | --- |
| `--omed-brand` | `#C2603A` | `#E08A66` | Logo, illustrations, large accents. Not small text |
| `--omed-primary` | `#B85630` | `#E08A66` | Primary buttons, links, focus, active menu item |
| `--omed-primary-hover` | `#A94D29` | `#EBA283` | Hover |
| `--omed-primary-active` | `#94421F` | `#C9714D` | Pressed |
| `--omed-on-primary` | `#FFFFFF` | `#2A140B` | Text on primary buttons |
| `--omed-tint` | `rgba(194,96,58,.10)` | `rgba(224,138,102,.16)` | Selected item background, soft badges |
| `--omed-tint-text` | `#7A3519` | `#F4C4AE` | Text on tint |

> `#C2603A` (the original pick) has 4.18:1 contrast against white, below AA for small text. So actions use `#B85630` (4.76:1), while `#C2603A` stays as the identity color.

### Warm neutrals

| Token | Light | Dark |
| --- | --- | --- |
| `--omed-bg` | `#FAF6F0` | `#1C1713` |
| `--omed-text` | `#2B211B` | `#F3EAE2` |
| `--omed-text-secondary` | 68% text | 68% text |
| `--omed-text-muted` | 45% text | 42% text |
| `--omed-border` | `rgba(43,33,27,.12)` | `rgba(243,234,226,.14)` |
| `--omed-border-soft` | `rgba(43,33,27,.07)` | `rgba(243,234,226,.08)` |

### Semantic

| Role | Light | Dark |
| --- | --- | --- |
| Success | `#4F7F52` | `#7FB382` |
| Warning | `#B7791F` | `#E0B04F` |
| Danger | `#B83A4B` | `#E0707F` |
| Info | `#4A6FA5` | `#82A6D6` |

Danger leans rose on purpose so it doesn't get confused with terracotta.

## 3. Glass

| Property | Light | Dark |
| --- | --- | --- |
| Fill | `rgba(255,255,255,.55)` | `rgba(255,255,255,.06)` |
| Fill strong (overlays, header, sider) | `rgba(255,255,255,.72)` | `rgba(255,255,255,.10)` |
| Edge (1px border) | `rgba(255,255,255,.65)` | `rgba(255,255,255,.12)` |
| Blur | 16px + saturate 140% | same |
| Shadow | `0 4px 24px rgba(120,60,30,.08)` | `0 4px 28px rgba(0,0,0,.35)` |

Rules:

- Glass needs a colorful backdrop. The page background already has terracotta/amber blobs (`--omed-page-bg`); don't replace it with a flat color.
- At most **2 stacked glass layers**. Glass inside glass isn't blurred again (handled in the CSS).
- Long text, tables, and heavy forms go on dense surfaces (fill strong), not thin glass.
- Falls back to solid colors automatically when the browser doesn't support blur or the user enables `prefers-reduced-transparency`.

## 4. Typography

- Font: **Inter** (already loaded in `root.tsx`), system sans fallback.
- Base 15px. Two weights only: 400 and 500.

| Role | Size |
| --- | --- |
| Heading 1 | 28 |
| Heading 2 | 22 |
| Heading 3 | 18 |
| Heading 4 | 16 |
| Body, labels, tables | 15 |
| Caption, metadata | 13 |

Sentence case for all labels, buttons, and titles. No ALL CAPS.

## 5. Density and shape

- Ant Design uses `compactAlgorithm` + 15px font. Use `size="small"` on tables and forms when the data is dense.
- Spacing scale: **4, 8, 12, 16, 24**. Card padding 16, gap between cards 12.
- Radius: small 6 (tags, checkboxes), default 10 (buttons, inputs, small cards), large 14 (cards, modals, drawers).
- Header height 52px.

## 6. Component guidance

| Component | Rule |
| --- | --- |
| Button | One `type="primary"` per screen. The rest are default or text. No shadow |
| Menu / sider | Active item: `tint` background + primary text, no active bar |
| Table | Transparent header, `tint` on hover, `border-soft` borders |
| Card | Glass comes from the CSS. Don't set a background color manually |
| Modal / Drawer | Fill strong so forms stay readable |
| Tag / badge | `tint` + `tint-text`, not solid colors |
| Form | Labels on top, `size="small"`, errors use danger |
| Empty state | Short invitation + one action button, warm tone |

## 7. Voice and copy

- Warm and clear, like talking to a fellow parish volunteer. No jargon.
- Buttons: verb first, 1–3 words ("Add member", "Save changes").
- Errors: what happened, then what to do. No exclamation marks.

## 8. Do and don't

| Do | Don't |
| --- | --- |
| Use tokens (`var(--omed-*)` or the antd theme) | Hardcode hex values in components |
| Terracotta for the main action and active states | Terracotta as decoration everywhere |
| Glass on surfaces, solid for dense content | Three or more stacked glass layers |
| Check contrast in both light and dark | Small `brand`-colored text on light backgrounds |
| Warm neutrals | Cold grays mixed into the theme |

## 9. Usage

```tsx
// app/root.tsx
import { ConfigProvider } from "antd";
import { useEffect, useMemo, useState } from "react";
import { getAntdTheme, type ThemeMode } from "~/theme/antd";
import "~/theme/tokens.css";

function useThemeMode(): ThemeMode {
  const [mode, setMode] = useState<ThemeMode>("light");
  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => setMode(mq.matches ? "dark" : "light");
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, []);
  useEffect(() => {
    document.documentElement.dataset.theme = mode;
  }, [mode]);
  return mode;
}

export default function App() {
  const mode = useThemeMode();
  const theme = useMemo(() => getAntdTheme(mode), [mode]);
  return (
    <ConfigProvider theme={theme}>
      <Outlet />
    </ConfigProvider>
  );
}
```

Notes:

- A manual light/dark toggle just changes `mode` and `data-theme`. Both must stay in sync.
- `apps/dash/app/app.css` currently only has `@import "tailwindcss"`. Import `tokens.css` after it so the variables are available.
- `react-router.config.ts` is still `ssr: true`, while the README says SPA. If it stays SSR, Ant Design needs `@ant-design/cssinjs` setup to avoid a style flash.
