import type { ThemeConfig } from "antd";

// Omed dashboard theme — adjust tokens here, everything else inherits.
export const omedTheme: ThemeConfig = {
  token: {
    colorPrimary: "#2B4C7E", // deep slate blue — feels "financial/enterprise" not generic SaaS purple
    colorInfo: "#2B4C7E",
    colorLink: "#2B4C7E",
    borderRadius: 6,
    fontFamily:
      "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
    fontSize: 14,
  },
  components: {
    Layout: {
      siderBg: "#141B2D",
      headerBg: "#ffffff",
      bodyBg: "#F5F6FA",
      headerHeight: 56,
    },
    Menu: {
      darkItemBg: "#141B2D",
      darkItemSelectedBg: "#2B4C7E",
      darkItemHoverBg: "#1E2740",
      darkSubMenuItemBg: "#0F1524",
      itemBorderRadius: 6,
      itemMarginInline: 8,
    },
  },
};
