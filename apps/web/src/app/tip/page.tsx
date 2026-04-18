import { useTranslations } from "next-intl";
import { TopNav } from "@/components/TopNav";

const OPEN_COLLECTIVE_URL =
  process.env.NEXT_PUBLIC_OPEN_COLLECTIVE_URL ?? "https://opencollective.com/";

export default function TipPage() {
  const t = useTranslations("tip");
  return (
    <>
      <TopNav />
      <main>
        <h1>{t("heading")}</h1>
        <p>{t("intro")}</p>
        <ul>
          <li>
            <strong>{t("familyAid")}</strong> — routed monthly to verified
            partner bond and mutual-aid funds (Envision Freedom Fund, Freedom
            for Immigrants National Bond Fund, RAICES, local rapid-response
            networks). Receipts published quarterly.
          </li>
          <li>
            <strong>{t("operations")}</strong> — pays for Hetzner, Nominatim,
            domain, and the moderator stipend.
          </li>
        </ul>
        <p>
          <a href={OPEN_COLLECTIVE_URL} rel="noopener noreferrer" target="_blank">
            {t("donateButton")}
          </a>
        </p>
        <p style={{ color: "var(--muted)" }}>
          We deliberately do not accept in-app donations on iOS / Android.
          Linking to Open Collective keeps the ledger transparent and avoids
          App Store payment middlemen.
        </p>
      </main>
    </>
  );
}
