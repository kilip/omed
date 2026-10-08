import dayjs from "dayjs";
import "dayjs/locale/id";
import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import resourcesToBackend from "i18next-resources-to-backend";
import { initReactI18next } from "react-i18next";
import {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  SUPPORTED_LOCALES,
  toLocale,
} from "./config";

const isBrowser = typeof window !== "undefined";

function applyDocumentLocale(lng: string) {
  const locale = toLocale(lng);
  dayjs.locale(locale);
  if (isBrowser) document.documentElement.lang = locale;
}

i18n.on("languageChanged", applyDocumentLocale);

/**
 * Resolves once i18next is initialised and the `common` + `nav` namespaces
 * for the detected language are loaded. Other namespaces are lazy-loaded
 * (one chunk per language/namespace) when a component asks for them.
 */
export const i18nReady = i18n
  .use(LanguageDetector)
  .use(
    resourcesToBackend(
      (lng: string, ns: string) => import(`./locales/${lng}/${ns}.json`),
    ),
  )
  .use(initReactI18next)
  .init({
    supportedLngs: SUPPORTED_LOCALES,
    fallbackLng: DEFAULT_LOCALE,
    nonExplicitSupportedLngs: true,
    load: "languageOnly",
    ns: ["common", "nav"],
    defaultNS: "common",
    fallbackNS: false,
    detection: {
      order: ["localStorage", "navigator"],
      lookupLocalStorage: LOCALE_STORAGE_KEY,
      caches: ["localStorage"],
    },
    interpolation: { escapeValue: false },
    react: { useSuspense: true },
    returnNull: false,
  });

export { i18n };
export default i18n;
