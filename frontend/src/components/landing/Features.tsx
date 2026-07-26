"use client";

import { useTranslations } from "next-intl";
import { Bell, TrendingDown, RotateCcw } from "lucide-react";

const FEATURES = [
  { key: "alerts", icon: Bell },
  { key: "risk", icon: TrendingDown },
  { key: "returns", icon: RotateCcw },
] as const;

export default function Features() {
  const t = useTranslations("landing.features");

  return (
    <div className="bg-slate-50 py-20">
      <div className="mx-auto max-w-6xl px-6">
        <h2 className="text-center text-3xl font-bold text-slate-900">{t("title")}</h2>

        <div className="mt-12 grid grid-cols-1 gap-6 md:grid-cols-3">
          {FEATURES.map(({ key, icon: Icon }) => (
            <div key={key} className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
              <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-lg bg-emerald-100">
                <Icon className="h-6 w-6 text-emerald-600" />
              </div>
              <h3 className="font-semibold text-slate-900">{t(`${key}.title`)}</h3>
              <p className="mt-2 text-sm text-slate-600">{t(`${key}.body`)}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
