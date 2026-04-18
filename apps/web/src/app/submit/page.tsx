import { useTranslations } from "next-intl";
import { TopNav } from "@/components/TopNav";

export default function SubmitPage() {
  const t = useTranslations();
  return (
    <>
      <TopNav />
      <main>
        <h1>{t("nav.submit")}</h1>
        <div className="notice warn">{t("submitNotice")}</div>
        {/*
          Form lands in week 5 alongside the moderation service.
          Required fields:
            - geolocation (auto-snapped to 200m grid by the API)
            - observed_at (must be within last 8 hours)
            - category
            - source URL OR partner-hotline token
            - notes (optional; runs through PII scrubber server-side)
        */}
        <p style={{ color: "var(--muted)" }}>
          The submission form is not yet implemented. See <code>docs/safety-policy.md</code> for the contract it must satisfy.
        </p>
      </main>
    </>
  );
}
