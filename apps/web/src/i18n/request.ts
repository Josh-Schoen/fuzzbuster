import { getRequestConfig } from "next-intl/server";

// Locales supported at launch. Spanish, Hmong, and Somali are the three
// most-spoken non-English languages relevant to Minnesota immigrant
// communities. Add Karen, Oromo, Amharic, Arabic as partner orgs come on
// board.
export const SUPPORTED_LOCALES = ["en", "es", "hmn", "so"] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "en";

export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale = (SUPPORTED_LOCALES as readonly string[]).includes(requested ?? "")
    ? (requested as Locale)
    : DEFAULT_LOCALE;
  return {
    locale,
    messages: (await import(`../../messages/${locale}.json`)).default,
  };
});
