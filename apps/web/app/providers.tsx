import { QueryClientProvider } from "@tanstack/react-query";
import { App as AntdApp, ConfigProvider } from "antd";
import enUS from "antd/locale/en_US";
import idID from "antd/locale/id_ID";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import "./i18n";
import { getQueryClient } from "./lib/query-client";
import { getAntdTheme } from "./theme/antd";
import { ThemeProvider, useTheme } from "./theme/theme-context";

function ThemedApp({ children }: { children: ReactNode }) {
  const { mode } = useTheme();
  const { i18n } = useTranslation();

  return (
    <ConfigProvider
      theme={getAntdTheme(mode)}
      locale={i18n.language === "id" ? idID : enUS}
      componentSize="small"
    >
      <AntdApp>{children}</AntdApp>
    </ConfigProvider>
  );
}

export function Providers({ children }: { children: ReactNode }) {
  const queryClient = getQueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ThemedApp>{children}</ThemedApp>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
