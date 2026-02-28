import en from "./en";
import zh from "./zh";

export type Locale = "en" | "zh";
type LocaleDict = Record<string, string>;
export type TranslateParams = Record<string, string | number | boolean | null | undefined>;

const dictMap: Record<Locale, LocaleDict> = {
  en,
  zh,
};

export function t(locale: Locale, key: string, fallback?: string, params?: TranslateParams): string {
  const dict = dictMap[locale] ?? dictMap.en;
  const template = dict[key] ?? fallback ?? key;
  return formatTemplate(template, params);
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

function formatTemplate(template: string, params?: TranslateParams): string {
  if (!params) return template;
  return template.replace(/\{([a-zA-Z0-9_]+)\}/g, (_, rawKey: string) => {
    const value = params[rawKey];
    return value === null || value === undefined ? "" : String(value);
  });
}
