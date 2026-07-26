"use client";

import { useTranslations } from "next-intl";
import { CalendarCheck, Mail } from "lucide-react";

export default function CtaSection() {
  const t = useTranslations("landing.cta");

  return (
    <div className="bg-emerald-600 py-20">
      <div className="mx-auto max-w-3xl px-6 text-center">
        <h2 className="text-3xl font-bold text-white">{t("title")}</h2>
        <p className="mt-3 text-emerald-50">{t("subtitle")}</p>

        <div className="mt-8 flex flex-col items-center gap-4">
          <a
            href="mailto:pharmasense4@gmail.com?subject=Book%20a%20demo"
            className="inline-flex items-center gap-2 rounded-lg bg-white px-6 py-3 text-sm font-semibold text-emerald-700 shadow-sm transition-colors hover:bg-emerald-50"
          >
            <CalendarCheck className="h-4 w-4" />
            {t("bookDemo")}
          </a>

          <div className="flex items-center gap-2 text-sm text-emerald-50">
            <Mail className="h-4 w-4" />
            <span>{t("contact")}</span>
            <a
              href="mailto:pharmasense4@gmail.com"
              className="font-medium text-white underline hover:text-emerald-100"
            >
              pharmasense4@gmail.com
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
