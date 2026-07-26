"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { cn } from "@/lib/utils";
import { PLAN_ORDER, PLAN_BUTTON } from "@/lib/plans";
import PricingCard from "@/components/billing/PricingCard";

export default function Pricing() {
  const t = useTranslations("billing");
  const tLanding = useTranslations("landing.pricing");
  const { locale } = useParams();
  const lc = (locale as string) ?? "en";

  return (
    <div className="mx-auto max-w-6xl px-6 py-20">
      <div className="text-center">
        <h2 className="text-3xl font-bold text-slate-900">{tLanding("title")}</h2>
        <p className="mt-3 text-slate-600">{tLanding("subtitle")}</p>
      </div>

      <div className="mt-12 grid grid-cols-1 gap-6 md:grid-cols-3">
        {PLAN_ORDER.map((planKey) => {
          const features = t.raw(`plans.${planKey}.features`) as string[];
          const buttonColor = PLAN_BUTTON[planKey];

          const cta = (
            <Link
              href={`/${lc}/signup`}
              className={cn(
                "w-full rounded-lg px-4 py-2.5 text-sm font-semibold text-center transition-colors",
                buttonColor
                  ? cn(buttonColor, "text-white")
                  : "border border-slate-300 text-slate-700 hover:bg-slate-50"
              )}
            >
              {tLanding("cta")}
            </Link>
          );

          return (
            <PricingCard
              key={planKey}
              planKey={planKey}
              name={t(`plans.${planKey}.name`)}
              price={t(`plans.${planKey}.price`)}
              features={features}
              perMonthLabel={t("perMonth")}
              cta={cta}
            />
          );
        })}
      </div>
    </div>
  );
}
