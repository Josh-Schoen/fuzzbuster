import { useTranslations } from "next-intl";
import { TopNav } from "@/components/TopNav";
import { SightingsMap } from "@/components/SightingsMap";

export default function HomePage() {
  const t = useTranslations();
  return (
    <>
      <TopNav />
      <main>
        <h1>{t("appName")}</h1>
        <p>{t("tagline")}</p>
        <div className="notice">{t("decayNotice")}</div>
        <SightingsMap />
      </main>
    </>
  );
}
