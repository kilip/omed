import { ConfigProvider } from "antd";
import enUS from "antd/locale/en_US";
import idID from "antd/locale/id_ID";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { buildTheme, type Mode, palette } from "./theme";

const STORAGE_KEY = "omed:theme";

type ThemeCtx = {
  mode: Mode;
  toggle: () => void;
  setMode: (mode: Mode) => void;
};
const Ctx = createContext<ThemeCtx | null>(null);

function initialMode(): Mode {
  if (typeof window === "undefined") return "light";
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "light" || saved === "dark") return saved;
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setMode] = useState<Mode>(initialMode);
  const { i18n } = useTranslation();

  const activeLang = (i18n.resolvedLanguage ?? i18n.language ?? "en").split(
    "-",
  )[0];
  const antdLocale = activeLang === "id" ? idID : enUS;

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, mode);
    const root = document.documentElement;
    root.dataset.theme = mode;
    root.style.colorScheme = mode;
    document.body.style.backgroundColor = palette[mode].bgLayout;
  }, [mode]);

  useEffect(() => {
    document.documentElement.lang = activeLang;
  }, [activeLang]);

  const toggle = useCallback(
    () => setMode((m) => (m === "light" ? "dark" : "light")),
    [],
  );

  const config = useMemo(() => buildTheme(mode), [mode]);
  const value = useMemo(() => ({ mode, toggle, setMode }), [mode, toggle]);

  return (
    <Ctx.Provider value={value}>
      <ConfigProvider theme={config} locale={antdLocale}>
        {children}
      </ConfigProvider>
    </Ctx.Provider>
  );
}

export function useThemeMode() {
  const ctx = useContext(Ctx);
  if (!ctx) throw new Error("useThemeMode must be used inside <ThemeProvider>");
  return ctx;
}
