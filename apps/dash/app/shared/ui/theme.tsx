import { type ThemeConfig, theme } from "antd";

const font =
  "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif";

const components: ThemeConfig["components"] = {
  Layout: {
    siderBg: "transparent",
    headerBg: "transparent",
    bodyBg: "transparent",
  },
  Menu: {
    itemBg: "transparent",
    subMenuItemBg: "transparent",
    itemBorderRadius: 12,
    itemMarginInline: 8,
    itemHeight: 42,
  },
  Table: { headerBg: "transparent", colorBgContainer: "transparent" },
  Button: { controlHeight: 38, fontWeight: 500 },
};

export const lightTheme: ThemeConfig = {
  algorithm: theme.defaultAlgorithm,
  token: {
    colorPrimary: "#6366f1",
    colorBgContainer: "rgba(255,255,255,0.55)",
    colorBgElevated: "rgba(255,255,255,0.9)",
    colorBorder: "rgba(255,255,255,0.7)",
    colorBorderSecondary: "rgba(255,255,255,0.5)",
    borderRadius: 14,
    fontFamily: font,
  },
  components: {
    ...components,
    Menu: {
      ...components.Menu,
      itemSelectedBg: "rgba(99,102,241,0.15)",
      itemSelectedColor: "#4338ca",
    },
  },
};

export const darkTheme: ThemeConfig = {
  algorithm: theme.darkAlgorithm,
  token: {
    colorPrimary: "#818cf8",
    colorBgContainer: "rgba(20,24,45,0.5)",
    colorBgElevated: "rgba(24,28,52,0.92)",
    colorBorder: "rgba(255,255,255,0.12)",
    colorBorderSecondary: "rgba(255,255,255,0.08)",
    borderRadius: 14,
    fontFamily: font,
  },
  components: {
    ...components,
    Menu: {
      ...components.Menu,
      itemSelectedBg: "rgba(129,140,248,0.22)",
      itemSelectedColor: "#c7d2fe",
    },
  },
};
