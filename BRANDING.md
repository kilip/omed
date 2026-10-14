# Omed brand guidelines

Style: **modern, glassmorphism, compact, cool**. Component base: Ant Design. Primary color: **ocean blue**.

Related files:

- `apps/web/app/theme/tokens.css`: CSS variables + glass utilities
- `apps/web/app/theme/antd.ts`: `getAntdTheme(mode)` for `ConfigProvider`

## 1. Principles

1. **Cool, not stark.** Neutrals always lean blue-slate, never pure gray or warm brown.
2. **Glass in moderation.** Blur is for surfaces (cards, modals, sider, header), not for everything.
3. **Compact but breathable.** Small controls, 4/8/12/16 spacing, a clear hierarchy. Dense never means cramped.
4. **One accent per screen.** Ocean blue is only for the main action and active states.
5. **Legible first, pretty second.** Small text must pass AA contrast (4.5:1).

## 2. Color

### Brand and action

| Token | Light | Dark | Used for |
| --- | --- | --- | --- |
| `--omed-brand` | `#2478DB` | `#6AA8F5` | Logo, illustrations, large accents. Not small text |
| `--omed-primary` | `#0B63CE` | `#6AA8F5` | Primary buttons, links, focus, active menu item |
| `--omed-primary-hover` | `#0A56B3` | `#8BBCF8` | Hover |
| `--omed-primary-active` | `#084A98` | `#4F93E6` | Pressed |
| `--omed-on-primary` | `#FFFFFF` | `#101620` | Text on primary buttons |
| `--omed-tint` | `rgba(11,99,206,.10)` | `rgba(106,168,245,.16)` | Selected item background, soft badges |
| `--omed-tint-text` | `#0A428F` | `#BBD8FB` | Text on tint |

> `#2478DB` (brand) has 4.39:1 contrast against white, below AA for small text. So actions use `#0B63CE` (5.7:1), while `#2478DB` stays as the identity color. In dark mode brand and primary are the same color.

### Cool neutrals

| Token | Light | Dark |
| --- | --- | --- |
| `--omed-bg` | `#F1F5FB` | `#101620` |
| `--omed-text` | `#14233A` | `#E4EDF9` |
| `--omed-text-secondary` | 68% text | 68% text |
| `--omed-text-muted` | 45% text | 42% text |
| `--omed-border` | `rgba(20,35,58,.12)` | `rgba(228,237,249,.14)` |
| `--omed-border-soft` | `rgba(20,35,58,.07)` | `rgba(228,237,249,.08)` |

### Semantic

| Role | Light | Dark |
| --- | --- | --- |
| Success | `#4F7F52` | `#7FB382` |
| Warning | `#B7791F` | `#E0B04F` |
| Danger | `#B83A4B` | `#E0707F` |
| Info | `#0F766E` | `#4FD1C5` |

Danger leans rose on purpose so it doesn't get confused with the primary blue. Info is teal (not blue) for the same reason: a blue info message would read like an action.

## 3. Glass

| Property | Light | Dark |
| --- | --- | --- |
| Fill | `rgba(255,255,255,.55)` | `rgba(255,255,255,.06)` |
| Fill strong (overlays, header, sider) | `rgba(255,255,255,.72)` | `rgba(255,255,255,.10)` |
| Edge (1px border) | `rgba(255,255,255,.65)` | `rgba(255,255,255,.12)` |
| Blur | 16px + saturate 140% | same |
| Shadow | `0 4px 24px rgba(20,60,120,.08)` | `0 4px 28px rgba(0,0,0,.35)` |

Rules:

- Glass needs a colorful backdrop. The page background already has blue/teal blobs (`--omed-page-bg`); don't replace it with a flat color.
- At most **2 stacked glass layers**. Glass inside glass isn't blurred again (handled in the CSS).
- Long text, tables, and heavy forms go on dense surfaces (fill strong), not thin glass.
- Falls back to solid colors automatically when the browser doesn't support blur or the user enables `prefers-reduced-transparency`.

## 4. Typography

- Font: **Inter** (already loaded in `root.tsx`), system sans fallback.
- Base 14px. Two weights only: 400 and 500.

| Role | Size |
| --- | --- |
| Heading 1 | 24 |
| Heading 2 | 20 |
| Heading 3 | 16 |
| Heading 4 | 14 (weight 500) |
| Body, labels | 14 |
| Tables, dense lists | 13 |
| Caption, metadata | 12 |

Sentence case for all labels, buttons, and titles. No ALL CAPS.

## 5. Density and shape

- Ant Design uses `compactAlgorithm` + 14px font. Use `size="small"` on tables and forms by default, not only when the data is dense.
- Spacing scale: **4, 8, 12, 16, 24**. Card padding 12 (16 for modals and drawers), gap between cards 12.
- Radius: small 6 (tags, checkboxes), default 10 (buttons, inputs, small cards), large 14 (cards, modals, drawers).
- Header height 44px. Sider width 148px.
- Table cells: 5px vertical padding, 12px horizontal.

## 6. Component guidance

| Component | Rule |
| --- | --- |
| Button | One `type="primary"` per screen. The rest are default or text. No shadow. `size="small"` |
| Menu / sider | Active item: `tint` background + primary text, no active bar |
| Table | Transparent header, `tint` on hover, `border-soft` borders, no border on the last row |
| Card | Glass comes from the CSS. Don't set a background color manually |
| KPI card | Label (12) above value (20, weight 500). An optional sparkline in `primary` at 55% opacity |
| Modal / Drawer | Fill strong so forms stay readable |
| Tag / badge | `tint` + `tint-text`, not solid colors. Neutral states use `border-soft` fill + secondary text |
| Form | Labels on top, `size="small"`, errors use danger |
| Empty state | Short invitation + one action button, friendly tone |

## 7. Voice and copy

- Warm and clear, like talking to a helpful teammate. No jargon.
- Buttons: verb first, 1–3 words ("Add member", "Save changes").
- Errors: what happened, then what to do. No exclamation marks.

## 8. Do and don't

| Do | Don't |
| --- | --- |
| Use tokens (`var(--omed-*)` or the antd theme) | Hardcode hex values in components |
| Ocean blue for the main action and active states | Ocean blue as decoration everywhere |
| Glass on surfaces, solid for dense content | Three or more stacked glass layers |
| Check contrast in both light and dark | Small `brand`-colored text on light backgrounds |
| Blue-slate neutrals | Warm browns or pure grays mixed into the theme |
| Teal for info, rose for danger | Blue for info or danger, which reads as an action |

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