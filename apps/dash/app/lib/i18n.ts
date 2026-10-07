import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";

import enAuth from "~/locales/en/auth.json";
import enBlog from "~/locales/en/blog.json";
import enCommon from "~/locales/en/common.json";
import enFinance from "~/locales/en/finance.json";
import enHome from "~/locales/en/home.json";
import enNav from "~/locales/en/nav.json";
import enSettings from "~/locales/en/settings.json";
import enUnderConstruction from "~/locales/en/underConstruction.json";

import idAuth from "~/locales/id/auth.json";
import idBlog from "~/locales/id/blog.json";
import idCommon from "~/locales/id/common.json";
import idFinance from "~/locales/id/finance.json";
import idHome from "~/locales/id/home.json";
import idNav from "~/locales/id/nav.json";
import idSettings from "~/locales/id/settings.json";
import idUnderConstruction from "~/locales/id/underConstruction.json";

export const defaultNS = "common";

export const resources = {
  en: {
    common: enCommon,
    auth: enAuth,
    blog: enBlog,
    finance: enFinance,
    home: enHome,
    nav: enNav,
    settings: enSettings,
    underConstruction: enUnderConstruction,
  },
  id: {
    common: idCommon,
    auth: idAuth,
    blog: idBlog,
    finance: idFinance,
    home: idHome,
    nav: idNav,
    settings: idSettings,
    underConstruction: idUnderConstruction,
  },
} as const;

export const SUPPORTED_LANGUAGES = [
  { code: "en", label: "English" },
  { code: "id", label: "Bahasa Indonesia" },
] as const;

export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number]["code"];

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    supportedLngs: ["en", "id"],
    defaultNS,
    ns: [
      "common",
      "auth",
      "blog",
      "finance",
      "home",
      "nav",
      "settings",
      "underConstruction",
    ],
    detection: {
      order: ["localStorage"],
      lookupLocalStorage: "omed:lang",
      caches: ["localStorage"],
    },
    interpolation: {
      escapeValue: false,
    },
  });

export default i18n;

declare module "i18next" {
  interface CustomTypeOptions {
    defaultNS: typeof defaultNS;
    resources: (typeof resources)["en"];
  }
}
