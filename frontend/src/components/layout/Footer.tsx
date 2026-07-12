"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";

export default function Footer() {
  const t = useTranslations("footer");
  const { locale } = useParams();
  const lc = (locale as string) ?? "en";

  return (
    <footer className="border-t border-slate-200 bg-white py-6">
      <div className="mx-auto flex max-w-7xl flex-col items-center justify-between gap-4 px-6 md:flex-row">
        <div className="text-sm text-slate-500">{t("rights")}</div>
        <div className="flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-sm text-slate-600">
          <Link href={`/${lc}/privacy`} className="hover:text-emerald-700">
            {t("privacy")}
          </Link>
          <Link href={`/${lc}/terms`} className="hover:text-emerald-700">
            {t("terms")}
          </Link>
          <Link href={`/${lc}/cookies`} className="hover:text-emerald-700">
            {t("cookies")}
          </Link>
          <a
            href="mailto:pharmasense4@gmail.com"
            className="hover:text-emerald-700"
          >
            {t("contact")}
          </a>
        </div>
      </div>
    </footer>
  );
}
