import { Check } from "lucide-react";
import { cn, planBadgeClass } from "@/lib/utils";
import { PLAN_RING, type PlanKey } from "@/lib/plans";

interface PricingCardProps {
  planKey: PlanKey;
  name: string;
  price: string;
  features: string[];
  isCurrent?: boolean;
  currentPlanBadgeLabel?: string;
  perMonthLabel?: string;
  cta?: React.ReactNode;
}

export default function PricingCard({
  planKey,
  name,
  price,
  features,
  isCurrent,
  currentPlanBadgeLabel,
  perMonthLabel,
  cta,
}: PricingCardProps) {
  return (
    <div
      className={cn(
        "rounded-xl border bg-white shadow-sm p-6 flex flex-col ring-2",
        isCurrent ? PLAN_RING[planKey] : "ring-transparent border-slate-200"
      )}
    >
      <div className="flex items-center justify-between mb-2">
        <h3 className="font-semibold text-slate-900">{name}</h3>
        {isCurrent && currentPlanBadgeLabel && (
          <span className={cn("text-xs px-2 py-0.5 rounded-full font-medium", planBadgeClass(planKey))}>
            {currentPlanBadgeLabel}
          </span>
        )}
      </div>

      <div className="mb-4">
        <span className="text-3xl font-bold text-slate-900">{price}</span>
        {planKey !== "free" && perMonthLabel && (
          <span className="text-sm text-slate-400">{perMonthLabel}</span>
        )}
      </div>

      <ul className="space-y-2 mb-6 flex-1">
        {features.map((feature) => (
          <li key={feature} className="flex items-start gap-2 text-sm text-slate-600">
            <Check className="w-4 h-4 text-emerald-600 mt-0.5 flex-shrink-0" />
            {feature}
          </li>
        ))}
      </ul>

      {cta}
    </div>
  );
}
