"use client";

import { useQuery, useMutation } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { apiFetch } from "@/lib/api/client";
import { formatDate } from "@/lib/utils";
import { toast } from "sonner";
import { Check, Zap, Building2, Sparkles } from "lucide-react";

interface Subscription {
  plan: string;
  subscription_status?: string;
  subscription_current_period_end?: string;
}

const PLANS = [
  {
    id: "free",
    name: "Free",
    price: "€0",
    icon: Sparkles,
    features: ["1 pharmacy", "100 inventory items", "Basic alerts", "Email support"],
    highlight: false,
  },
  {
    id: "pro",
    name: "Pro",
    price: "€29",
    icon: Zap,
    features: ["1 pharmacy", "Unlimited inventory", "SMS alerts", "Daily digest emails", "CSV import", "Priority support"],
    highlight: true,
  },
  {
    id: "chain",
    name: "Chain",
    price: "€99",
    icon: Building2,
    features: ["Up to 10 pharmacies", "All Pro features", "Chain dashboard", "Multi-user management", "Dedicated support"],
    highlight: false,
  },
];

export default function BillingPage() {
  const t = useTranslations("billing");

  const { data: subscription } = useQuery<Subscription>({
    queryKey: ["subscription"],
    queryFn: () => apiFetch("/billing/subscription"),
  });

  const checkoutMutation = useMutation({
    mutationFn: (plan: string) =>
      apiFetch<{ url: string }>("/billing/checkout-session", {
        method: "POST",
        body: JSON.stringify({ plan }),
      }),
    onSuccess: (data) => {
      toast.success(t("upgradeSuccess"));
      window.location.href = data.url;
    },
    onError: () => toast.error("Failed to start checkout"),
  });

  const portalMutation = useMutation({
    mutationFn: () =>
      apiFetch<{ url: string }>("/billing/portal-session", { method: "POST" }),
    onSuccess: (data) => {
      window.location.href = data.url;
    },
    onError: () => toast.error("Failed to open billing portal"),
  });

  return (
    <div className="p-6 lg:p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
      </div>

      {/* Current plan banner */}
      {subscription && (
        <div className="bg-emerald-50 border border-emerald-200 rounded-xl p-5 mb-8 flex items-center justify-between flex-wrap gap-4">
          <div>
            <p className="text-sm text-emerald-700 font-medium">{t("currentPlan")}</p>
            <p className="text-2xl font-bold text-emerald-900 capitalize">{subscription.plan}</p>
            {subscription.subscription_status && (
              <p className="text-xs text-emerald-600 capitalize mt-0.5">
                Status: {subscription.subscription_status}
                {subscription.subscription_current_period_end && (
                  <> · Renews {formatDate(subscription.subscription_current_period_end)}</>
                )}
              </p>
            )}
          </div>
          {subscription.plan !== "free" && (
            <button
              onClick={() => portalMutation.mutate()}
              disabled={portalMutation.isPending}
              className="rounded-lg border border-emerald-600 px-4 py-2 text-sm font-semibold text-emerald-700 hover:bg-emerald-100 transition-colors"
            >
              {t("manage")}
            </button>
          )}
        </div>
      )}

      {/* Plan cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {PLANS.map(({ id, name, price, icon: Icon, features, highlight }) => {
          const isCurrent = subscription?.plan === id;
          return (
            <div
              key={id}
              className={`rounded-xl border bg-white shadow-sm p-6 flex flex-col relative ${
                highlight ? "border-emerald-500 shadow-emerald-100" : "border-slate-200"
              }`}
            >
              {highlight && (
                <div className="absolute -top-3 left-1/2 -translate-x-1/2">
                  <span className="bg-emerald-600 text-white text-xs font-bold px-3 py-1 rounded-full">
                    MOST POPULAR
                  </span>
                </div>
              )}

              <div className={`w-10 h-10 rounded-lg flex items-center justify-center mb-4 ${
                highlight ? "bg-emerald-600 text-white" : "bg-slate-100 text-slate-600"
              }`}>
                <Icon className="w-5 h-5" />
              </div>

              <h3 className="text-lg font-bold text-slate-900 mb-1">{name}</h3>
              <div className="text-3xl font-extrabold text-slate-900 mb-1">
                {price}
                <span className="text-base font-normal text-slate-500">{t("perMonth")}</span>
              </div>

              <ul className="space-y-2.5 my-6 flex-1">
                {features.map((feature) => (
                  <li key={feature} className="flex items-center gap-2 text-sm text-slate-600">
                    <Check className="w-4 h-4 text-emerald-600 flex-shrink-0" />
                    {feature}
                  </li>
                ))}
              </ul>

              {isCurrent ? (
                <div className="text-center py-2.5 text-sm font-semibold text-emerald-700 bg-emerald-50 rounded-lg">
                  ✓ Current Plan
                </div>
              ) : id !== "free" ? (
                <button
                  onClick={() => checkoutMutation.mutate(id)}
                  disabled={checkoutMutation.isPending}
                  className={`rounded-lg px-4 py-2.5 text-sm font-semibold transition-colors ${
                    highlight
                      ? "bg-emerald-600 text-white hover:bg-emerald-700"
                      : "border border-slate-200 text-slate-700 hover:bg-slate-50"
                  } disabled:opacity-50`}
                >
                  {t("upgrade")} to {name}
                </button>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
