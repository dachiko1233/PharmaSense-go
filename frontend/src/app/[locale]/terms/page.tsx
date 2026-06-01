import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import LegalPageShell, {
  LegalList,
  LegalParagraph,
  LegalSection,
} from "@/components/layout/LegalPageShell";

export const metadata: Metadata = {
  title: "Terms of Service | PharmaSense",
  description:
    "Terms of Service for PharmaSense — the SaaS expiry-monitoring platform for licensed pharmacies in Cyprus and the EU.",
  robots: "index, follow",
};

export default async function TermsPage() {
  const t = await getTranslations("terms");

  const eligibility = t.raw("eligibility.items") as string[];
  const account = t.raw("account.items") as string[];
  const acceptableUse = t.raw("acceptableUse.items") as string[];
  const billing = t.raw("billing.items") as string[];

  return (
    <LegalPageShell title={t("title")}>
      <LegalParagraph>{t("intro")}</LegalParagraph>

      <LegalSection title={t("acceptance.title")}>
        <LegalParagraph>{t("acceptance.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("service.title")}>
        <LegalParagraph>{t("service.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("eligibility.title")}>
        <LegalList items={eligibility} />
      </LegalSection>

      <LegalSection title={t("account.title")}>
        <LegalList items={account} />
      </LegalSection>

      <LegalSection title={t("acceptableUse.title")}>
        <LegalParagraph>{t("acceptableUse.intro")}</LegalParagraph>
        <LegalList items={acceptableUse} />
      </LegalSection>

      <LegalSection title={t("billing.title")}>
        <LegalList items={billing} />
      </LegalSection>

      <LegalSection title={t("ip.title")}>
        <LegalParagraph>{t("ip.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("dataOwnership.title")}>
        <LegalParagraph>{t("dataOwnership.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("availability.title")}>
        <LegalParagraph>{t("availability.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("liability.title")}>
        <LegalParagraph>{t("liability.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("disclaimer.title")}>
        <LegalParagraph>{t("disclaimer.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("indemnification.title")}>
        <LegalParagraph>{t("indemnification.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("termination.title")}>
        <LegalParagraph>{t("termination.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("modifications.title")}>
        <LegalParagraph>{t("modifications.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("governingLaw.title")}>
        <LegalParagraph>{t("governingLaw.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("disputes.title")}>
        <LegalParagraph>{t("disputes.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("contact.title")}>
        <LegalParagraph>{t("contact.body")}</LegalParagraph>
      </LegalSection>
    </LegalPageShell>
  );
}
