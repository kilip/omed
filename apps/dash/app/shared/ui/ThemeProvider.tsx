import { ConfigProvider } from "antd";
import {
	createContext,
	type ReactNode,
	useCallback,
	useContext,
	useEffect,
	useMemo,
	useState,
} from "react";
import { buildTheme, type Mode, palette } from "./theme";

const STORAGE_KEY = "omed:theme";

type ThemeCtx = { mode: Mode; toggle: () => void };
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

	useEffect(() => {
		localStorage.setItem(STORAGE_KEY, mode);
		const root = document.documentElement;
		root.dataset.theme = mode;
		root.style.colorScheme = mode;
		document.body.style.backgroundColor = palette[mode].bgLayout;
	}, [mode]);

	const toggle = useCallback(
		() => setMode((m) => (m === "light" ? "dark" : "light")),
		[],
	);

	const config = useMemo(() => buildTheme(mode), [mode]);
	const value = useMemo(() => ({ mode, toggle }), [mode, toggle]);

	return (
		<Ctx.Provider value={value}>
			<ConfigProvider theme={config}>{children}</ConfigProvider>
		</Ctx.Provider>
	);
}

export function useThemeMode() {
	const ctx = useContext(Ctx);
	if (!ctx) throw new Error("useThemeMode must be used inside <ThemeProvider>");
	return ctx;
}
