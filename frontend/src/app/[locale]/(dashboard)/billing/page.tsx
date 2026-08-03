"use client";

import { useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { useSearchParams } from "next/navigation";
import { apiFetch } from "@/lib/api/client";
import { cn, formatDate, planBadgeClass } from "@/lib/utils";
import { PLAN_ORDER, PLAN_BUTTON, type PlanKey } from "@/lib/plans";
import PricingCard from "@/components/billing/PricingCard";
import { Subscription } from "@/types";
import { toast } from "sonner";
import { CreditCard, CheckCircle2 } from "lucide-react";

export default function BillingPage() {
  const t = useTranslations("billing");
  const tPlan = useTranslations("plan");
  const queryClient = useQueryClient();
  const searchParams = useSearchParams();
  const handledRedirect = useRef(false);

  const justPaid = !!searchParams.get("success");

  const { data, isLoading, isError, refetch } = useQuery<Subscription>({
    queryKey: ["billing-subscription"],
    queryFn: () => apiFetch("/billing/subscription"),
    // After a successful checkout the plan flips via the Dodo webhook, which
    // may land a beat after the customer is redirected back. Poll briefly so
    // the confirmed plan appears without a manual refresh; stop once it's no
    // longer "free" (or react-query gives up on error).
    refetchInterval: (query) =>
      justPaid && (query.state.data?.plan ?? "free") === "free" ? 2000 : false,
  });

  useEffect(() => {
    if (handledRedirect.current) return;
    if (searchParams.get("success")) {
      handledRedirect.current = true;
      toast.success(t("upgradeSuccess"));
      queryClient.invalidateQueries({ queryKey: ["billing-subscription"] });
    } else if (searchParams.get("canceled")) {
      handledRedirect.current = true;
      toast(t("upgradeCanceled"));
    }
  }, [searchParams]);

  const portalMutation = useMutation({
    mutationFn: () =>
      apiFetch<{ url: string }>("/billing/portal-session", { method: "POST" }),
    onSuccess: (res) => {
      window.location.href = res.url;
    },
    onError: (e: Error) => toast.error(e.message || t("portalFailed")),
  });

  const checkoutMutation = useMutation({
    mutationFn: (plan: PlanKey) =>
      apiFetch<{ url: string }>("/billing/checkout-session", {
        method: "POST",
        body: JSON.stringify({ plan }),
      }),
    onSuccess: (res) => {
      window.location.href = res.url;
    },
    onError: (e: Error) => toast.error(e.message || t("upgradeFailed")),
  });

  // Payment confirmation is driven purely by the return-URL param, so the user
  // always sees a clear success state after paying — even if the subscription
  // fetch is still loading, errors out, or the webhook hasn't flipped the plan
  // yet. It never depends on a network call resolving.
  const successBanner = justPaid ? (
    <div className="mb-6 rounded-xl border border-emerald-200 bg-emerald-50 p-4 flex items-start gap-3">
      <CheckCircle2 className="w-5 h-5 text-emerald-600 flex-shrink-0 mt-0.5" />
      <div>
        <p className="font-semibold text-emerald-900">{t("paymentConfirmedTitle")}</p>
        <p className="text-sm text-emerald-700 mt-0.5">{t("paymentConfirmedBody")}</p>
      </div>
    </div>
  ) : null;

  if (isLoading) {
    return (
      <div className="p-6 lg:p-8 max-w-4xl">
        {successBanner}
        <div className="h-8 bg-slate-200 rounded w-48 mb-6 animate-pulse" />
        <div className="rounded-xl border border-slate-200 bg-white p-6 animate-pulse space-y-4">
          <div className="h-4 bg-slate-200 rounded w-3/4" />
          <div className="h-4 bg-slate-200 rounded w-1/2" />
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="p-6 lg:p-8 max-w-4xl">
        {successBanner}
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-6 flex flex-col items-start gap-3">
          <p className="text-sm text-amber-800">{t("loadError")}</p>
          <button
            onClick={() => refetch()}
            className="rounded-lg border border-amber-300 bg-white px-4 py-2 text-sm font-medium text-amber-800 hover:bg-amber-100"
          >
            {t("retry")}
          </button>
        </div>
      </div>
    );
  }

  const currentPlan = (data?.plan ?? "free") as PlanKey;
  const currentIndex = PLAN_ORDER.indexOf(currentPlan);

  return (
    <div className="p-6 lg:p-8 max-w-4xl">
      {successBanner}
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
        <p className="text-sm text-slate-500 mt-1">{t("subtitle")}</p>
      </div>

      {/* Current plan */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6 mb-8">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-9 h-9 bg-emerald-100 rounded-lg flex items-center justify-center">
            <CreditCard className="w-5 h-5 text-emerald-600" />
          </div>
          <h2 className="font-semibold text-slate-900">{t("currentPlan")}</h2>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <span className={cn("text-sm px-3 py-1 rounded-full font-semibold", planBadgeClass(currentPlan))}>
              {tPlan(currentPlan)}
            </span>
            {data?.subscription_status && (
              <span className="text-xs px-2.5 py-1 rounded-full bg-slate-100 text-slate-600 font-medium">
                {t("status")}: {data.subscription_status}
              </span>
            )}
            {data?.subscription_current_period_end && (
              <span className="text-xs text-slate-400">
                {t("renewsOn", { date: formatDate(data.subscription_current_period_end) })}
              </span>
            )}
          </div>

          {currentPlan !== "free" && (
            <button
              onClick={() => portalMutation.mutate()}
              disabled={portalMutation.isPending}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
            >
              {portalMutation.isPending ? t("upgrade") + "…" : t("manageBilling")}
            </button>
          )}
        </div>
      </div>

      {/* Plan comparison */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {PLAN_ORDER.map((planKey, index) => {
          const isCurrent = planKey === currentPlan;
          const canUpgrade = index > currentIndex;
          const features = t.raw(`plans.${planKey}.features`) as string[];

          const isCheckingOut =
            checkoutMutation.isPending && checkoutMutation.variables === planKey;

          const cta = canUpgrade ? (
            <button
              onClick={() => checkoutMutation.mutate(planKey)}
              disabled={checkoutMutation.isPending}
              className={cn(
                "w-full rounded-lg px-4 py-2.5 text-sm font-semibold text-white text-center disabled:opacity-50",
                PLAN_BUTTON[planKey]
              )}
            >
              {isCheckingOut ? t("upgrade") + "…" : t("upgrade")}
            </button>
          ) : isCurrent ? (
            <div className="w-full rounded-lg px-4 py-2.5 text-sm font-semibold text-center text-slate-400 border border-slate-200">
              {t("currentPlanBadge")}
            </div>
          ) : null;

          return (
            <PricingCard
              key={planKey}
              planKey={planKey}
              name={t(`plans.${planKey}.name`)}
              price={t(`plans.${planKey}.price`)}
              features={features}
              isCurrent={isCurrent}
              currentPlanBadgeLabel={t("currentPlanBadge")}
              perMonthLabel={t("perMonth")}
              cta={cta}
            />
          );
        })}
      </div>
    </div>
  );
}
