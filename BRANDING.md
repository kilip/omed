# Omed brand guidelines

Style: **warm, flat, compact, modern** (Notion-like). Component base: Ant Design. Primary color: **cocoa**.

Related files:

- `apps/web/app/theme/tokens.css`: CSS variables + surface utilities
- `apps/web/app/theme/antd.ts`: `getAntdTheme(mode)` for `ConfigProvider`
- `apps/web/app/theme/theme-context.tsx`: `ThemeProvider` and `useTheme` (system, light, dark)
- `apps/web/app/theme/init-script.ts`: inline head script that prevents a theme flash on load

## 1. Principles

1. **Warm, not stark.** Neutrals always lean warm cream and cocoa, never pure gray or cool blue-slate.
2. **Flat surfaces.** Hairline borders separate surfaces. No blur, no gradients, no decorative shadows.
3. **Compact but breathable.** Small controls, 4/8/12/16 spacing, a clear hierarchy. Dense never means cramped.
4. **One accent per screen.** Cocoa is only for the main action and active states.
5. **Legible first, pretty second.** Small text must pass AA contrast (4.5:1).

## 2. Color

### Brand and action

| Token | Light | Dark | Used for |
| --- | --- | --- | --- |
| `--omed-brand` | `#9A5B48` | `#C99480` | Logo ring and identity accents |
| `--omed-brand-accent` | `#CFA04A` | `#DDB462` | Logo arc only. Not an action or status color |
| `--omed-primary` | `#9A5B48` | `#C99480` | Primary buttons, links, focus, active menu item |
| `--omed-primary-hover` | `#874D3C` | `#D6A793` | Hover |
| `--omed-primary-active` | `#74402F` | `#B98068` | Pressed |
| `--omed-on-primary` | `#FFFFFF` | `#1B1917` | Text on primary buttons |
| `--omed-tint` | `rgba(154,91,72,.10)` | `rgba(201,148,128,.16)` | Selected item background, soft badges |
| `--omed-tint-text` | `#6E3A2B` | `#EACBBE` | Text on tint |

> Brand and primary are the same color. `#9A5B48` has 5.3:1 contrast against white, so it passes AA for small text and no separate action color is needed.

### Warm neutrals

| Token | Light | Dark | Used for |
| --- | --- | --- | --- |
| `--omed-bg` | `#FBF9F6` | `#1B1917` | Page background |
| `--omed-surface` | `#FFFFFF` | `#242120` | Cards, header, sider, modals |
| `--omed-surface-muted` | `#F4F1EC` | `#2E2A28` | Light cards, hover background |
| `--omed-text` | `#2F2B26` | `#ECE8E3` | Body text |
| `--omed-text-secondary` | `#736B61` | `#A09890` | Supporting text |
| `--omed-text-muted` | `#8C8479` | `#7C746C` | Placeholders and metadata only |
| `--omed-border` | `#E9E4DC` | `#3D3835` | Default hairline |
| `--omed-border-soft` | `#F0EBE3` | `#322E2B` | Table rows, quiet dividers |

> `--omed-text-muted` is about 3.5:1 on the page background. Never use it for text the user must read.

### Semantic

| Role | Fill (light) | Text (light) | Fill and text (dark) |
| --- | --- | --- | --- |
| Success | `#5F8767` | `#476B4F` | `#82AD8A` |
| Warning | `#CFA04A` | `#8A6420` | `#DDB462` |
| Danger | `#BC4B44` | `#BC4B44` | `#DE7570` |
| Info | `#3F727A` | `#3F727A` | `#7FB1B6` |

Each role has a fill token (`--omed-success`) and a text token (`--omed-success-text`). Use the fill for dots, icons, and borders, and the text token for text.

- In light mode the success fill is 4.1:1 and the warning fill is 2.4:1 on white. Both are below AA, so they are never used as text.
- Danger and primary are both reddish. Never rely on color alone for a destructive action: pair it with a label ("Delete member") and an icon.
- Info is teal and not part of the cocoa family, so an info message never reads like an action.

### Logo and app icons

The mark is a ring with a quarter arc. Assets live in `apps/web/public`:

| Asset | Colors | Used for |
| --- | --- | --- |
| `logo-icon.svg`, `logo-icon-512.png` | Ring `#9A5B48`, arc `#CFA04A`, transparent | Light backgrounds |
| `favicon.svg` | Same as above, switches to the dark pair with `prefers-color-scheme` | Browser tab |
| `favicon-16x16.png`, `favicon-32x32.png`, `favicon.ico` | Same as `logo-icon.svg` | Fallback favicons |
| `icon-rounded.svg`, `icon-192.png`, `icon-512.png` | Ring `#C99480`, arc `#DDB462` on `#1B1917` | App icons with rounded corners |
| `icon-maskable-512.png`, `apple-touch-icon.png` | Same as `icon-rounded.svg`, full bleed | Maskable and iOS icons |

When the colors change, edit the three SVGs and regenerate the PNGs and `favicon.ico` from them, so the files never drift apart. The in-app `Logo` component reads `--omed-brand` and `--omed-brand-accent`, so it follows the active theme.

## 3. Surfaces

| Class | Fill | Used for |
| --- | --- | --- |
| `.surface` | `--omed-surface` | Header, sider, dense panels (tables and forms), login and invite cards |
| `.surface-muted` | `--omed-surface-muted` | Light cards such as launcher tiles |

Both have a 1px `--omed-border` hairline and no shadow or blur. The page background is the flat `--omed-bg`.

Rules:

- Long text, tables, and heavy forms go on `.surface` (white), not `.surface-muted`.
- At most two nested levels. A `.surface-muted` card sits on the page, and a `.surface` card sits inside only when it holds dense content.
- Use `Panel` (`strong` for dense content) instead of repeating the classes by hand.

## 4. Typography

- Font: **Inter** (already loaded in `root.tsx`), system sans fallback.
- Base 13px. Two weights only: 400 and 500.

| Role | Size |
| --- | --- |
| Heading 1 | 22 |
| Heading 2 | 18 |
| Heading 3 | 15 |
| Heading 4 | 13 (weight 500) |
| Body, labels | 13 |
| Tables, dense lists | 13 |
| Caption, metadata | 12 |

Sentence case for all labels, buttons, and titles. No ALL CAPS.

## 5. Density and shape

- Ant Design uses `compactAlgorithm` + 13px font, and `ConfigProvider` sets `componentSize="small"`. Use `size="small"` on tables and forms by default, not only when the data is dense.
- Spacing scale: **4, 8, 12, 16, 24**. Card and panel padding 10, gap between cards 8 to 12, auth cards 20.
- Radius: small 4 (tags, checkboxes), default 6 (buttons, inputs, small cards), large 8 (cards, modals, drawers).
- Header height 40px. Sider width 140px. Menu item height 26px.
- Table cells: 4px vertical padding, 10px horizontal.

## 6. Component guidance

| Component | Rule |
| --- | --- |
| Button | One `type="primary"` per screen. The rest are default or text. No shadow. `size="small"` |
| Menu / sider | Active item: `tint` background + primary text, no active bar. Hover: `surface-muted` |
| Table | Transparent header, `surface-muted` on hover, `tint` on selected rows, `border-soft` borders, no border on the last row |
| Card | Use `.surface` or `.surface-muted`. Don't set a background color manually |
| KPI card | Label (12) above value (18, weight 500). An optional sparkline in `primary` at 55% opacity |
| Modal / Drawer | `surface` fill so forms stay readable |
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
| Cocoa for the main action and active states | Cocoa as decoration everywhere |
| Flat surfaces with hairline borders | Blur, gradients, or heavy shadows |
| Check contrast in both light and dark | `--omed-text-muted` or fill-only colors for readable text |
| Warm cream and cocoa neutrals | Cool blue-slate or pure grays mixed into the theme |
| Pair danger with a label and icon | Danger shown by color alone |

## 9. Theme modes

The dashboard supports three preferences: **system** (default), **light**, and **dark**.

- System follows `prefers-color-scheme` and updates live when the OS setting changes.
- The theme menu in the header (`ThemeMenu`) switches between light, dark, and system.
- The choice is stored in `localStorage` under `omed.theme`.
- `init-script.ts` runs in the document head and sets `data-theme` on `<html>` before the first paint, so there is no flash of the wrong theme.
- Tokens flip through `:root[data-theme="dark"]`, and Ant Design flips through `getAntdTheme(mode)`.

## 10. Usage

`ThemeProvider` and `ConfigProvider` are already wired in `app/providers.tsx`:

```tsx
// app/providers.tsx
function ThemedApp({ children }: { children: ReactNode }) {
  const { mode } = useTheme();

  return (
    <ConfigProvider theme={getAntdTheme(mode)} componentSize="small">
      <AntdApp>{children}</AntdApp>
    </ConfigProvider>
  );
}
```

In components, read colors from tokens:

```tsx
<span style={{ color: "var(--omed-text-secondary)" }}>Last updated</span>
<span style={{ color: "var(--omed-danger-text)" }}>Unpaid</span>
```
