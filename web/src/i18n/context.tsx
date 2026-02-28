import type { ReactNode } from "react";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { resolveLocale, t, type Locale, type TranslateParams } from "./index";

const LOCALE_STORAGE_KEY = "bp.locale";

interface I18nContextValue {
  locale: Locale;
  setLocale: (next: Locale) => void;
  tr: (key: string, fallback?: string, params?: TranslateParams) => string;
}

const I18nContext = createContext<I18nContextValue | undefined>(undefined);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(() => {
    const raw = typeof window !== "undefined" ? window.localStorage.getItem(LOCALE_STORAGE_KEY) : null;
    if (raw) return resolveLocale(raw);
    const browser = typeof navigator !== "undefined" ? navigator.language : "zh";
    return browser.toLowerCase().startsWith("zh") ? "zh" : "en";
  });

  useEffect(() => {
    if (typeof window === "undefined") return;
    window.localStorage.setItem(LOCALE_STORAGE_KEY, locale);
    document.documentElement.lang = locale;
  }, [locale]);

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
  }, []);

  const tr = useCallback(
    (key: string, fallback?: string, params?: TranslateParams) => t(locale, key, fallback, params),
    [locale]
  );

  const value = useMemo(
    () => ({
      locale,
      setLocale,
      tr,
    }),
    [locale, setLocale, tr]
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) {
    throw new Error("useI18n must be used within I18nProvider");
  }
  return ctx;
}

