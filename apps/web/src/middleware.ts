import createMiddleware from "next-intl/middleware";
import { SUPPORTED_LOCALES, DEFAULT_LOCALE } from "./i18n/request";

export default createMiddleware({
  locales: [...SUPPORTED_LOCALES],
  defaultLocale: DEFAULT_LOCALE,
  localePrefix: "as-needed",
});

export const config = {
  matcher: ["/((?!api|_next|sw.js|.*\\..*).*)"],
};
