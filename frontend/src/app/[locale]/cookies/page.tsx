import type { Metadata } from "next";
import Link from "next/link";
import { getTranslations } from "next-intl/server";
import LegalPageShell, {
  LegalList,
  LegalParagraph,
  LegalSection,
} from "@/components/layout/LegalPageShell";

export const metadata: Metadata = {
  title: "Cookie Policy | PharmaSense",
  description:
    "PharmaSense uses only essential cookies — no tracking, advertising, or analytics. GDPR/ePrivacy compliant.",
  robots: "index, follow",
};

interface CookieRow {
  name: string;
  purpose: string;
  duration: string;
}

export default async function CookiesPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  const t = await getTranslations("cookies");

  const rows = t.raw("rows") as CookieRow[];
  const headers = t.raw("headers") as {
    name: string;
    purpose: string;
    duration: string;
  };
  const doNotUse = t.raw("doNotUse") as string[];

  return (
    <LegalPageShell title={t("title")}>
      <LegalParagraph>{t("intro")}</LegalParagraph>

      <LegalSection title={t("tableTitle")}>
        <div className="overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-slate-200 bg-slate-50 text-left text-slate-700">
                <th className="px-3 py-2 font-medium">{headers.name}</th>
                <th className="px-3 py-2 font-medium">{headers.purpose}</th>
                <th className="px-3 py-2 font-medium">{headers.duration}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr key={i} className="border-b border-slate-200 text-slate-600">
                  <td className="px-3 py-2 font-mono text-xs text-slate-800">{row.name}</td>
                  <td className="px-3 py-2">{row.purpose}</td>
                  <td className="px-3 py-2">{row.duration}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </LegalSection>

      <LegalSection title={t("doNotUseTitle")}>
        <LegalList items={doNotUse} />
      </LegalSection>

      <LegalParagraph>{t("banner")}</LegalParagraph>

      <LegalParagraph>
        {t("more")}{" "}
        <Link
          href={`/${locale}/privacy`}
          className="font-medium text-emerald-700 underline hover:text-emerald-800"
        >
          {t("privacyLink")}
        </Link>
        .
      </LegalParagraph>
    </LegalPageShell>
  );
}
