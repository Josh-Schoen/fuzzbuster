import { useTranslations } from "next-intl";
import { TopNav } from "@/components/TopNav";

export default function KYRPage() {
  const t = useTranslations("kyr");
  return (
    <>
      <TopNav />
      <main>
        <h1>{t("heading")}</h1>
        <p>{t("intro")}</p>
        <ul>
          <li>
            <a href="https://www.aclu.org/know-your-rights/immigrants-rights" rel="noopener noreferrer" target="_blank">
              ACLU Know Your Rights for Immigrants
            </a>
          </li>
          <li>
            <a href="https://www.ilrc.org/red-cards" rel="noopener noreferrer" target="_blank">
              ILRC Red Cards (printable, 19 languages)
            </a>
          </li>
          <li>
            <a href="https://www.ilcm.org/" rel="noopener noreferrer" target="_blank">
              Immigrant Law Center of Minnesota (ILCM)
            </a>
          </li>
          <li>
            <a href="https://www.theadvocatesforhumanrights.org/" rel="noopener noreferrer" target="_blank">
              The Advocates for Human Rights
            </a>
          </li>
        </ul>
      </main>
    </>
  );
}
