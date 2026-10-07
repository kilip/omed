import { type ThemeConfig, theme } from "antd";

export type Mode = "light" | "dark";

export const palette = {
  light: {
    primary: "#C2410C",
    primaryHover: "#EA580C",
    primaryActive: "#9A3412",
    primaryTint: "#FFF1E8",
    bgLayout: "#FBF8F4",
    bgContainer: "#FFFFFF",
    bgElevated: "#FFFFFF",
    border: "#EAE2D8",
    borderSecondary: "#F1EAE1",
    text: "#2B2420",
    textSecondary: "#6B5F55",
    textTertiary: "#9A8C80",
    tableHeader: "#F6F1EA",
    rowHover: "#FFF8F2",
  },
  dark: {
    primary: "#F07A3C",
    primaryHover: "#F59560",
    primaryActive: "#D9622B",
    primaryTint: "#3A2A20",
    bgLayout: "#1C1815",
    bgContainer: "#26211D",
    bgElevated: "#2F2924",
    border: "#3D352E",
    borderSecondary: "#322B25",
    text: "#F3EDE6",
    textSecondary: "#B8AB9F",
    textTertiary: "#8A7D72",
    tableHeader: "#2B2521",
    rowHover: "#2D2621",
  },
} as const;

const fontFamily =
  '"Plus Jakarta Sans", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif';

export function buildTheme(mode: Mode): ThemeConfig {
  const p = palette[mode];
  const isDark = mode === "dark";

  return {
    algorithm: isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
    token: {
      colorPrimary: p.primary,
      colorPrimaryHover: p.primaryHover,
      colorPrimaryActive: p.primaryActive,
      colorPrimaryBg: p.primaryTint,
      colorInfo: "#3B7EA1",
      colorSuccess: "#3F8F5F",
      colorWarning: "#D99A1E",
      colorError: "#C8372D",

      colorBgLayout: p.bgLayout,
      colorBgContainer: p.bgContainer,
      colorBgElevated: p.bgElevated,
      colorBorder: p.border,
      colorBorderSecondary: p.borderSecondary,
      colorText: p.text,
      colorTextSecondary: p.textSecondary,
      colorTextTertiary: p.textTertiary,

      fontFamily,
      fontSize: 14,
      controlHeight: 40,
      borderRadius: 10,
      borderRadiusSM: 8,
      borderRadiusLG: 14,

      boxShadow: "0 8px 24px rgba(43, 36, 32, 0.08)",
      boxShadowSecondary: "0 8px 24px rgba(43, 36, 32, 0.08)",
      motionDurationMid: "0.18s",
    },
    components: {
      Layout: {
        bodyBg: p.bgLayout,
        siderBg: p.bgLayout,
        headerBg: p.bgLayout,
        headerHeight: 64,
        headerPadding: "0 24px",
      },
      Menu: {
        itemBg: "transparent",
        itemHeight: 40,
        itemBorderRadius: 10,
        itemMarginInline: 12,
        itemSelectedBg: p.primaryTint,
        itemSelectedColor: p.primary,
        itemHoverBg: p.rowHover,
        activeBarBorderWidth: 0,
      },
      Button: {
        primaryShadow: "none",
        defaultShadow: "none",
        fontWeight: 500,
      },
      Card: { borderRadiusLG: 14, paddingLG: 24 },
      Table: {
        headerBg: p.tableHeader,
        headerColor: p.textSecondary,
        rowHoverBg: p.rowHover,
        borderColor: p.borderSecondary,
      },
      Tag: { borderRadiusSM: 999 },
      Modal: { borderRadiusLG: 14 },
    },
  };
}
