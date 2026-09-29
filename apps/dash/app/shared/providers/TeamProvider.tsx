import { ConfigProvider } from "antd";
import {
  createContext,
  type PropsWithChildren,
  useContext,
  useEffect,
  useState,
} from "react";
import { darkTheme, lightTheme } from "~/shared/ui/theme";

type Mode = "light" | "dark";
const KEY = "omed-theme";
const Ctx = createContext<{ mode: Mode; toggle: () => void } | null>(null);

const initial = (): Mode => {
  if (typeof window === "undefined") return "light";
  const saved = localStorage.getItem(KEY) as Mode | null;
  return (
    saved ??
    (matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light")
  );
};

export function ThemeProvider({ children }: PropsWithChildren) {
  const [mode, setMode] = useState<Mode>(initial);

  useEffect(() => {
    document.documentElement.dataset.theme = mode;
    localStorage.setItem(KEY, mode);
  }, [mode]);

  return (
    <Ctx.Provider
      value={{
        mode,
        toggle: () => setMode((m) => (m === "dark" ? "light" : "dark")),
      }}
    >
      <ConfigProvider theme={mode === "dark" ? darkTheme : lightTheme}>
        {children}
      </ConfigProvider>
    </Ctx.Provider>
  );
}

export function useThemeMode() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useThemeMode must be used inside ThemeProvider");
  return ctx;
}
