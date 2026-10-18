# Omed brand guidelines

Style: **light, flat, compact, modern** (Odoo 20 inspired). Component base: Ant Design. Primary color: **plum**, with a **teal** secondary.

Omed borrows Odoo's layout patterns and color character: a plum primary, a teal accent, pale cool-gray surfaces, soft borders, rounded app tiles, and a control panel above every list. It does not use Odoo's logo, name, or icons.

Related files:

- `apps/web/app/theme/tokens.css`: CSS variables, surface utilities, and the launcher tile classes
- `apps/web/app/theme/antd.ts`: `getAntdTheme(mode)` for `ConfigProvider`
- `apps/web/app/theme/theme-context.tsx`: `ThemeProvider` and `useTheme` (system, light, dark)
- `apps/web/app/theme/init-script.ts`: inline head script that prevents a theme flash on load
- `apps/web/app/components/`: layout building blocks (see section 6)

## 1. Principles

1. **Light and airy.** Neutrals are pale and slightly cool, never pure white on pure white and never warm cream.
2. **Flat surfaces.** Hairline borders separate surfaces. No blur, no decorative shadows. The dotted launcher backdrop is the only decoration.
3. **Compact but breathable.** Small controls, 4/8/12/16 spacing, a clear hierarchy. Dense never means cramped.
4. **One accent per screen.** Plum is only for the main action and active states. Teal marks information and the second app color.
5. **Legible first, pretty second.** Small text must pass AA contrast (4.5:1).

## 2. Color

### Brand and action

| Token | Light | Dark | Used for |
| --- | --- | --- | --- |
| `--omed-brand` | `#714B67` | `#C79BBB` | Logo ring and identity accents |
| `--omed-brand-accent` | `#017E84` | `#4FB8BE` | Logo arc only. Not an action or status color |
| `--omed-primary` | `#714B67` | `#C79BBB` | Primary buttons, links, focus, active menu item |
| `--omed-primary-hover` | `#5F3E56` | `#D6B0CB` | Hover |
| `--omed-primary-active` | `#4D3246` | `#B889AB` | Pressed |
| `--omed-on-primary` | `#FFFFFF` | `#1B1620` | Text on primary buttons and the active statusbar step |
| `--omed-tint` | `rgba(113,75,103,.10)` | `rgba(199,155,187,.16)` | Selected item background, soft badges, search facets |
| `--omed-tint-text` | `#5A3A51` | `#E8D0E1` | Text on tint |

> Brand and primary are the same color. `#714B67` has 7.2:1 contrast against white, so it passes AA for small text and no separate action color is needed.

### Cool neutrals

| Token | Light | Dark | Used for |
| --- | --- | --- | --- |
| `--omed-bg` | `#F9FAFB` | `#17191D` | Page background |
| `--omed-surface` | `#FFFFFF` | `#1F2227` | Navbar, sidebar, cards, chatter, modals |
| `--omed-surface-muted` | `#F3F4F6` | `#2A2D34` | Light cards, hover background, notes |
| `--omed-text` | `#1F2430` | `#E8E9EC` | Body text |
| `--omed-text-secondary` | `#5F636F` | `#A0A4AE` | Supporting text |
| `--omed-text-muted` | `#8A8D97` | `#7B7F8A` | Placeholders and metadata only |
| `--omed-border` | `#E1E3E8` | `#363A42` | Default hairline |
| `--omed-border-soft` | `#EDEEF1` | `#2B2E35` | Table rows, quiet dividers |

> `--omed-text-muted` is about 3.2:1 on the page background. Never use it for text the user must read.

### Semantic

| Role | Fill (light) | Text (light) | Fill and text (dark) |
| --- | --- | --- | --- |
| Success | `#2E9E6B` | `#1F7A4F` | `#6CC596` |
| Warning | `#E3A008` | `#8A5A00` | `#E8B454` |
| Danger | `#C8423A` | `#B3342D` | `#EF7B72` |
| Info | `#017E84` | `#017E84` | `#5CC1C6` |

Each role has a fill token (`--omed-success`) and a text token (`--omed-success-text`). Use the fill for dots, icons, and borders, and the text token for text.

- In light mode the success fill is 3.4:1 and the warning fill is 2.3:1 on white. Both are below AA, so they are never used as text.
- Danger is red and primary is plum, which are close in dark mode. Never rely on color alone for a destructive action: pair it with a label ("Delete member") and an icon.
- Info is teal, so an info message never reads like an action.

### App tile colors

The launcher cycles three fills by module order: `--omed-app-1` plum, `--omed-app-2` teal, `--omed-app-3` coral (`#714B67`, `#017E84`, `#D9604F` in light mode). Icons on tiles are always `--omed-on-app` (white). Tiles are the one place coral appears; do not use it elsewhere.

### Logo and app icons

The mark is a ring with a quarter arc. Assets live in `apps/web/public`:

| Asset | Colors | Used for |
| --- | --- | --- |
| `logo-icon.svg`, `logo-icon-512.png` | Ring `#714B67`, arc `#017E84`, transparent | Light backgrounds |
| `favicon.svg` | Same as above, switches to the dark pair with `prefers-color-scheme` | Browser tab |
| `favicon-16x16.png`, `favicon-32x32.png`, `favicon.ico` | Same as `logo-icon.svg` | Fallback favicons |
| `icon-rounded.svg`, `icon-192.png`, `icon-512.png` | Ring `#C79BBB`, arc `#4FB8BE` on `#17191D` | App icons with rounded corners |
| `icon-maskable-512.png`, `apple-touch-icon.png` | Same as `icon-rounded.svg`, full bleed | Maskable and iOS icons |

When the colors change, edit the three SVGs and regenerate the PNGs and `favicon.ico` from them, so the files never drift apart. The in-app `Logo` component reads `--omed-brand` and `--omed-brand-accent`, so it follows the active theme.

## 3. Surfaces

| Class | Fill | Used for |
| --- | --- | --- |
| `.surface` | `--omed-surface` | Navbar, sidebar, dense panels (tables and forms), chatter, login and invite cards |
| `.surface-muted` | `--omed-surface-muted` | Light cards |
| `.home-backdrop` | `--omed-bg` with a dot grid | Launcher only |

`.surface` and `.surface-muted` have a 1px `--omed-border` hairline and no shadow or blur. The page background is the flat `--omed-bg`.

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
- Radius: small 4 (tags, checkboxes), default 6 (buttons, inputs, search bar), large 8 (cards, modals, drawers), extra large 16 (app tile icons only).
- Navbar height 46px. Sidebar width 200px, 52px collapsed. Menu item height 30px. Chatter width about 340px.
- Table cells: 4px vertical padding, 10px horizontal.

## 6. Layout

The shell follows Odoo's structure. Each part is a component in `apps/web/app/components/` or a route in `apps/web/app/routes/`.

| Part | Component | Behavior |
| --- | --- | --- |
| Navbar | `NavBar` | White surface with a hairline. Left: app switcher button, logo, and the current app name in primary. Right: workspace select, language, theme, user menu |
| App switcher | `routes/launcher.tsx` | Centered grid of app tiles on `.home-backdrop`. A tile is a 64px rounded icon (`.app-tile__icon`) with the label and description below. Modules are discovered from `module.ts`, so a new module gets a tile automatically |
| Sidebar | `ModuleLayout` | The module name and icon on top, then the menu from `module.ts`. Collapses to icons with the button at the bottom. The choice is stored in `localStorage` under `omed.sider`. Starts collapsed below the `md` breakpoint |
| Control panel | `ControlPanel` (through `Page`) | One row. Left: breadcrumb and action buttons. Middle: the search bar. Right: the record counter and the view switcher |
| Search bar | `SearchBar` | One box holding facet pills, the text input, and an optional filter dropdown. Facets use `tint` and can be removed one by one |
| View switcher | `ViewSwitcher` | Icon-only segmented control for `list`, `kanban`, and `form`. Hidden when a screen has fewer than two views |
| Form view | `FormView` | Action buttons on the left and the `StatusBar` on the right, above a white sheet. The `Chatter` sits beside the sheet and drops below it on narrow screens |
| Statusbar | `StatusBar` | Pills in one bordered row. The current stage uses `primary` fill and `on-primary` text |
| Chatter | `Chatter` | Composer with "Send message" and "Log note", a filter (all, conversations, changes), and the message history. Tracked changes use secondary text, notes use `surface-muted` |

Usage notes:

- Every list screen uses `Page`. Pass `search`, `views`, and `count` when the screen supports them. Pass `trail` to add breadcrumbs, for example when a record is open.
- Pages that open a record from a list render `FormView` in place of the list and add the list as a `trail` crumb, like Odoo's list to form navigation.
- `Chatter` is presentational. The caller owns the messages and the `onPost` handler. Until the API stores messages, keep them in local state and show the provisional notice.

## 7. Component guidance

| Component | Rule |
| --- | --- |
| Button | One `type="primary"` per screen, plus the one that moves a record forward in a form (for example "Post"). The rest are default or text. No shadow. `size="small"` |
| Menu / sider | Active item: `tint` background + primary text, no active bar. Hover: `surface-muted` |
| Table | Transparent header, `surface-muted` on hover, `tint` on selected rows, `border-soft` borders, no border on the last row. Rows that open a record are clickable with a pointer cursor |
| Card | Use `.surface` or `.surface-muted`. Don't set a background color manually |
| Kanban card | A `Panel strong` with the identity on top, tags in the middle, and metadata and actions at the bottom. Grid of at least 240px columns |
| KPI card | Label (12) above value (18, weight 500). An optional sparkline in `primary` at 55% opacity |
| Modal / Drawer | `surface` fill so forms stay readable |
| Tag / badge | `tint` + `tint-text`, not solid colors. Neutral states use `border-soft` fill + secondary text |
| Form | Labels on top, `size="small"`, errors use danger |
| Empty state | Short invitation + one action button, friendly tone |

## 8. Voice and copy

- Clear and direct, like a helpful teammate. No jargon.
- Buttons: verb first, 1–3 words ("Add member", "Save changes").
- Errors: what happened, then what to do. No exclamation marks.

## 9. Do and don't

| Do | Don't |
| --- | --- |
| Use tokens (`var(--omed-*)` or the antd theme) | Hardcode hex values in components |
| Plum for the main action and active states | Plum as decoration everywhere |
| Flat surfaces with hairline borders | Blur, gradients, or heavy shadows (the launcher dot grid is the only exception) |
| Check contrast in both light and dark | `--omed-text-muted` or fill-only colors for readable text |
| Pale cool neutrals with a plum and teal accent | Warm cream or cocoa tones mixed into the theme |
| Coral only on launcher tiles | Coral for actions or status |
| Pair danger with a label and icon | Danger shown by color alone |
| Reuse `Page`, `FormView`, and `Chatter` | Rebuild breadcrumbs, statusbars, or message lists per screen |

## 10. Theme modes

The dashboard supports three preferences: **system** (default), **light**, and **dark**.

- System follows `prefers-color-scheme` and updates live when the OS setting changes.
- The theme menu in the navbar (`ThemeMenu`) switches between light, dark, and system.
- The choice is stored in `localStorage` under `omed.theme`.
- `init-script.ts` runs in the document head and sets `data-theme` on `<html>` before the first paint, so there is no flash of the wrong theme.
- Tokens flip through `:root[data-theme="dark"]`, and Ant Design flips through `getAntdTheme(mode)`.

## 11. Usage

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

A screen with a control panel, a list, and a form view:

```tsx
<Page
  title={t("fin.menu.journal")}
  count={entries.length}
  search={{ value: query, onChange: setQuery }}
  views={{ value: view, onChange: setView, views: ["list", "kanban"] }}
  actions={<Button type="primary">{t("fin.journal.new")}</Button>}
>
  <FormView
    title={entry.number}
    buttons={<Button type="primary">{t("fin.journal.post")}</Button>}
    statusbar={<StatusBar current={entry.status} steps={steps} />}
    chatter={<Chatter messages={messages} onPost={post} />}
  >
    {/* sheet content */}
  </FormView>
</Page>
```
