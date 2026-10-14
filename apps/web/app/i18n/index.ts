import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import type { Lang } from "~/lib/format";
import { en } from "./en";
import { id } from "./id";

const STORAGE_KEY = "omed.lang";

function detectLanguage(): Lang {
  if (typeof window === "undefined") return "en";
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "en" || stored === "id") return stored;
  } catch {
    // Fall through to the browser language.
  }
  return navigator.language.toLowerCase().startsWith("id") ? "id" : "en";
}

i18n.use(initReactI18next).init({
  resources: { en: { translation: en }, id: { translation: id } },
  lng: detectLanguage(),
  fallbackLng: "en",
  interpolation: { escapeValue: false },
});

export function setLanguage(lang: Lang) {
  try {
    localStorage.setItem(STORAGE_KEY, lang);
  } catch {
    // Ignore storage failures; the language still changes for this session.
  }
  void i18n.changeLanguage(lang);
}

export default i18n;
