import en from "./en";
import zh from "./zh";

export type Locale = "en" | "zh";
type LocaleDict = Record<string, string>;

const dictMap: Record<Locale, LocaleDict> = {
  en,
  zh,
};

export function t(locale: Locale, key: string, fallback?: string): string {
  const dict = dictMap[locale] ?? dictMap.en;
  return dict[key] ?? fallback ?? key;
}

export function resolveLocale(raw?: string | null): Locale {
  if (raw === "zh" || raw === "en") {
    return raw;
  }
  return "zh";
}

export function getLocaleDictionaries(): Record<Locale, LocaleDict> {
  return dictMap;
}

