"use client";

import { useTranslations } from "next-intl";
import { AlertTriangle, ShieldCheck } from "lucide-react";

export default function ProblemSolution() {
  const t = useTranslations("landing.problem");

  return (
    <div className="mx-auto max-w-6xl px-6 py-20">
      <div className="grid grid-cols-1 gap-8 md:grid-cols-2">
        <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
          <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-lg bg-red-100">
            <AlertTriangle className="h-6 w-6 text-red-600" />
          </div>
          <h3 className="text-xl font-semibold text-slate-900">{t("problemTitle")}</h3>
          <p className="mt-3 text-slate-600">{t("problemBody")}</p>
        </div>

        <div className="rounded-xl border border-emerald-200 bg-emerald-50 p-8 shadow-sm">
          <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-lg bg-emerald-600">
            <ShieldCheck className="h-6 w-6 text-white" />
          </div>
          <h3 className="text-xl font-semibold text-slate-900">{t("solutionTitle")}</h3>
          <p className="mt-3 text-slate-600">{t("solutionBody")}</p>
        </div>
      </div>
    </div>
  );
}
