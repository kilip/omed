import { type ThemeConfig, theme } from "antd";

const sans =
  "'Instrument Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif";

type Palette = {
  bg: string;
  side: string;
  surface: string;
  border: string;
  borderSoft: string;
  text: string;
  muted: string;
  accent: string;
  accentSoft: string;
  hover: string;
};

const light: Palette = {
  bg: "#FBFAF7",
  side: "#F4F2ED",
  surface: "#FFFFFF",
  border: "#E4E0D6",
  borderSoft: "#EEEBE3",
  text: "#2B2A27",
  muted: "#77736A",
  accent: "#3F5EA8",
  accentSoft: "rgba(63, 94, 168, 0.10)",
  hover: "rgba(43, 42, 39, 0.05)",
};

const dark: Palette = {
  bg: "#1B1A18",
  side: "#161513",
  surface: "#232220",
  border: "#37352F",
  borderSoft: "#2C2A26",
  text: "#ECE9E2",
  muted: "#9A958A",
  accent: "#8FA8E8",
  accentSoft: "rgba(143, 168, 232, 0.14)",
  hover: "rgba(236, 233, 226, 0.06)",
};

function build(p: Palette, algorithm: ThemeConfig["algorithm"]): ThemeConfig {
  return {
    algorithm,
    token: {
      colorPrimary: p.accent,
      colorBgBase: p.bg,
      colorBgLayout: p.bg,
      colorBgContainer: p.surface,
      colorBgElevated: p.surface,
      colorTextBase: p.text,
      colorTextSecondary: p.muted,
      colorTextTertiary: p.muted,
      colorBorder: p.border,
      colorBorderSecondary: p.borderSoft,
      colorSplit: p.borderSoft,
      borderRadius: 8,
      borderRadiusLG: 10,
      fontFamily: sans,
      fontSize: 15,
      lineHeight: 1.6,
      boxShadow: "none",
      boxShadowSecondary: "0 8px 24px rgba(0, 0, 0, 0.08)",
    },
    components: {
      Layout: {
        bodyBg: p.bg,
        headerBg: p.bg,
        siderBg: p.side,
        headerPadding: "0 24px",
        headerHeight: 56,
      },
      Menu: {
        itemBg: "transparent",
        subMenuItemBg: "transparent",
        itemColor: p.muted,
        itemHoverBg: p.hover,
        itemHoverColor: p.text,
        itemSelectedBg: p.hover,
        itemSelectedColor: p.text,
        itemBorderRadius: 6,
        itemMarginInline: 8,
        itemMarginBlock: 2,
        itemHeight: 36,
        groupTitleColor: p.muted,
      },
      Button: {
        controlHeight: 36,
        fontWeight: 500,
        primaryShadow: "none",
        defaultShadow: "none",
      },
      Card: {
        borderRadiusLG: 10,
        headerFontSize: 15,
        colorBorderSecondary: p.borderSoft,
      },
      Table: {
        headerBg: "transparent",
        headerColor: p.muted,
        headerSplitColor: "transparent",
        rowHoverBg: p.hover,
        borderColor: p.borderSoft,
      },
      Input: { controlHeight: 36 },
      Select: { controlHeight: 36 },
      DatePicker: { controlHeight: 36 },
      Modal: { borderRadiusLG: 12 },
    },
  };
}

export const lightTheme = build(light, theme.defaultAlgorithm);
export const darkTheme = build(dark, theme.darkAlgorithm);
