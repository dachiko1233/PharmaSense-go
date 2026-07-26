"use client";

import Image from "next/image";
import { useTranslations } from "next-intl";

export default function DashboardPreview() {
  const t = useTranslations("landing.screenshot");

  return (
    <div className="mx-auto max-w-6xl px-6 py-20">
      <h2 className="text-center text-3xl font-bold text-slate-900">
        {t("title")}
      </h2>
      <div className="mt-10 overflow-hidden rounded-xl border border-slate-200 shadow-lg">
        <Image
          src="/dash.png"
          alt={t("alt")}
          width={1853}
          height={958}
          className="h-auto w-full"
        />
      </div>
    </div>
  );
}
