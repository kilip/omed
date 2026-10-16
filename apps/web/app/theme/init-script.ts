import { THEME_STORAGE_KEY } from "./theme-context";

/**
 * Inline script for the document head. It sets `data-theme` before the first
 * paint so a dark-mode user does not see a light flash while the app loads.
 * The logic mirrors `ThemeProvider`: a stored light/dark choice wins,
 * otherwise the system preference applies.
 */
export const themeInitScript = `(function(){try{var s=localStorage.getItem(${JSON.stringify(THEME_STORAGE_KEY)});var d=s==="dark"||((s!=="light")&&window.matchMedia("(prefers-color-scheme: dark)").matches);document.documentElement.dataset.theme=d?"dark":"light"}catch(e){}})()`;
