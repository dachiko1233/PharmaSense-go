import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import LegalPageShell, {
  LegalList,
  LegalParagraph,
  LegalSection,
} from "@/components/layout/LegalPageShell";

export const metadata: Metadata = {
  title: "Privacy Policy | PharmaSense",
  description:
    "How PharmaSense collects, uses, and protects your data. GDPR compliant.",
  robots: "index, follow",
};

interface ProcessorRow {
  name: string;
  purpose: string;
  location: string;
}

export default async function PrivacyPage() {
  const t = await getTranslations("privacy");

  const dataItems = t.raw("dataCollected.items") as string[];
  const legalItems = t.raw("legalBasis.items") as string[];
  const processorRows = t.raw("processors.rows") as ProcessorRow[];
  const processorHeaders = t.raw("processors.headers") as {
    name: string;
    purpose: string;
    location: string;
  };
  const pharmacyItems = t.raw("pharmacyData.items") as string[];
  const retentionItems = t.raw("retention.items") as string[];
  const securityItems = t.raw("security.items") as string[];
  const rightsItems = t.raw("rights.items") as string[];

  return (
    <LegalPageShell title={t("title")}>
      <LegalParagraph>{t("intro")}</LegalParagraph>

      <LegalSection title={t("controller.title")}>
        <LegalParagraph>{t("controller.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("dataCollected.title")}>
        <LegalParagraph>{t("dataCollected.intro")}</LegalParagraph>
        <LegalList items={dataItems} />
      </LegalSection>

      <LegalSection title={t("legalBasis.title")}>
        <LegalParagraph>{t("legalBasis.intro")}</LegalParagraph>
        <LegalList items={legalItems} />
      </LegalSection>

      <LegalSection title={t("processors.title")}>
        <LegalParagraph>{t("processors.intro")}</LegalParagraph>
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50 text-left text-slate-700">
                <th className="px-3 py-2 font-medium">{processorHeaders.name}</th>
                <th className="px-3 py-2 font-medium">{processorHeaders.purpose}</th>
                <th className="px-3 py-2 font-medium">{processorHeaders.location}</th>
              </tr>
            </thead>
            <tbody>
              {processorRows.map((row, i) => (
                <tr key={i} className="border-b border-slate-200 text-slate-600">
                  <td className="px-3 py-2 font-medium text-slate-800">{row.name}</td>
                  <td className="px-3 py-2">{row.purpose}</td>
                  <td className="px-3 py-2">{row.location}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </LegalSection>

      <LegalSection title={t("pharmacyData.title")}>
        <LegalList items={pharmacyItems} />
      </LegalSection>

      <LegalSection title={t("retention.title")}>
        <LegalList items={retentionItems} />
      </LegalSection>

      <LegalSection title={t("security.title")}>
        <LegalList items={securityItems} />
      </LegalSection>

      <LegalSection title={t("rights.title")}>
        <LegalParagraph>{t("rights.intro")}</LegalParagraph>
        <LegalList items={rightsItems} />
        <LegalParagraph>{t("rights.exercise")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("cookies.title")}>
        <LegalParagraph>{t("cookies.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("transfers.title")}>
        <LegalParagraph>{t("transfers.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("changes.title")}>
        <LegalParagraph>{t("changes.body")}</LegalParagraph>
      </LegalSection>

      <LegalSection title={t("contact.title")}>
        <LegalParagraph>{t("contact.body")}</LegalParagraph>
      </LegalSection>
    </LegalPageShell>
  );
}
