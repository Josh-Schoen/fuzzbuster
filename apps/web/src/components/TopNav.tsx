import Link from "next/link";
import { useTranslations } from "next-intl";

export function TopNav() {
  const t = useTranslations("nav");
  return (
    <nav className="top">
      <Link href="/" className="brand">Fuzzbuster</Link>
      <Link href="/">{t("map")}</Link>
      <Link href="/resources">{t("resources")}</Link>
      <Link href="/submit">{t("submit")}</Link>
      <Link href="/kyr">{t("kyr")}</Link>
      <Link href="/tip">{t("tip")}</Link>
    </nav>
  );
}
