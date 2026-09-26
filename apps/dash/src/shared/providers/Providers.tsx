"use client";

import { App as AntApp, ConfigProvider } from "antd";
import { omedTheme } from "@/shared/ui/layouts/theme";
import { AuthProvider } from "./AuthProvider";

// Wrap the app root with this once. Handles antd SSR style injection
// (required on Next 16 App Router) + global theme tokens + AntApp
// context (needed for message/notification/modal static methods).
export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ConfigProvider theme={omedTheme}>
      <AntApp>
        <AuthProvider>{children}</AuthProvider>
      </AntApp>
    </ConfigProvider>
  );
}
