import { type ThemeConfig, theme } from "antd";

export type ThemeMode = "light" | "dark";

// Hex values live here (and in tokens.css) only. Components use tokens.
const palette = {
  light: {
    primary: "#0B63CE",
    primaryHover: "#0A56B3",
    primaryActive: "#084A98",
    text: "#14233A",
    success: "#4F7F52",
    warning: "#B7791F",
    danger: "#B83A4B",
    info: "#0F766E",
    tint: "rgba(11,99,206,0.10)",
    tintText: "#0A428F",
    border: "rgba(20,35,58,0.12)",
    borderSoft: "rgba(20,35,58,0.07)",
    container: "rgba(255,255,255,0.6)",
    elevated: "#FFFFFF",
  },
  dark: {
    primary: "#6AA8F5",
    primaryHover: "#8BBCF8",
    primaryActive: "#4F93E6",
    text: "#E4EDF9",
    success: "#7FB382",
    warning: "#E0B04F",
    danger: "#E0707F",
    info: "#4FD1C5",
    tint: "rgba(106,168,245,0.16)",
    tintText: "#BBD8FB",
    border: "rgba(228,237,249,0.14)",
    borderSoft: "rgba(228,237,249,0.08)",
    container: "rgba(255,255,255,0.06)",
    elevated: "#18202E",
  },
} as const;

export function getAntdTheme(mode: ThemeMode): ThemeConfig {
  const p = palette[mode];

  return {
    algorithm: [
      theme.compactAlgorithm,
      mode === "dark" ? theme.darkAlgorithm : theme.defaultAlgorithm,
    ],
    token: {
      colorPrimary: p.primary,
      colorInfo: p.info,
      colorSuccess: p.success,
      colorWarning: p.warning,
      colorError: p.danger,
      colorTextBase: p.text,
      colorBgContainer: p.container,
      colorBgElevated: p.elevated,
      colorBorder: p.border,
      colorBorderSecondary: p.borderSoft,
      fontFamily:
        'Inter, ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif',
      fontSize: 14,
      borderRadius: 10,
      borderRadiusSM: 6,
      borderRadiusLG: 14,
      fontWeightStrong: 500,
    },
    components: {
      Button: {
        primaryShadow: "none",
        defaultShadow: "none",
        dangerShadow: "none",
        fontWeight: 500,
      },
      Layout: {
        bodyBg: "transparent",
        headerBg: "transparent",
        siderBg: "transparent",
        headerHeight: 44,
        headerPadding: "0 12px",
      },
      Menu: {
        itemBg: "transparent",
        itemSelectedBg: p.tint,
        itemSelectedColor: p.primary,
        activeBarBorderWidth: 0,
        itemHeight: 28,
      },
      Table: {
        headerBg: "transparent",
        headerSplitColor: "transparent",
        rowHoverBg: p.tint,
        borderColor: p.borderSoft,
        cellPaddingBlockSM: 5,
        cellPaddingInlineSM: 12,
        cellFontSizeSM: 13,
      },
      Card: {
        bodyPadding: 12,
        bodyPaddingSM: 12,
        headerPadding: 12,
      },
      Tag: {
        defaultBg: p.tint,
        defaultColor: p.tintText,
      },
      Modal: { contentBg: p.elevated, headerBg: p.elevated },
      Drawer: { colorBgElevated: p.elevated },
    },
  };
}
