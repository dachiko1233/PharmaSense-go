"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { apiFetch } from "@/lib/api/client";
import { RiskAssessment } from "@/types";
import { formatCurrency, formatDate, riskBadgeClass, cn } from "@/lib/utils";
import { toast } from "sonner";
import { CheckCircle, ArrowRightLeft, RotateCcw, XCircle } from "lucide-react";

type Tab = "" | "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

interface AlertRow {
  assessment_id: string;
  batch_id: string;
  risk_level: string;
  days_until_expiry: number;
  estimated_loss?: number;
  suggested_discount_percent?: number;
  product_name: string;
  category?: string;
  batch_number?: string;
  expiry_date: string;
  current_quantity: number;
  purchase_price: number;
}

export default function AlertsPage() {
  const t = useTranslations("alerts");
  const tRisk = useTranslations("risk");
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>("CRITICAL");

  const { data: alerts = [], isLoading } = useQuery<AlertRow[]>({
    queryKey: ["alerts", tab],
    queryFn: () => apiFetch(`/alerts${tab ? `?risk_level=${tab}` : ""}`),
  });

  const actionMutation = useMutation({
    mutationFn: ({ batchId, actionType, discountPct }: { batchId: string; actionType: string; discountPct?: number }) =>
      apiFetch(`/alerts/${batchId}/action`, {
        method: "POST",
        body: JSON.stringify({ action_type: actionType, discount_percent: discountPct }),
      }),
    onSuccess: () => {
      toast.success(t("actionRecorded"));
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
    onError: () => toast.error("Action failed"),
  });

  const tabs: { label: string; value: Tab }[] = [
    { label: t("all"), value: "" },
    { label: t("critical"), value: "CRITICAL" },
    { label: t("high"), value: "HIGH" },
    { label: t("medium"), value: "MEDIUM" },
    { label: t("low"), value: "LOW" },
  ];

  return (
    <div className="p-6 lg:p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
        <p className="text-slate-500 text-sm mt-1">{t("subtitle")}</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 bg-slate-100 p-1 rounded-xl w-fit">
        {tabs.map(({ label, value }) => (
          <button
            key={value}
            onClick={() => setTab(value)}
            className={cn(
              "px-4 py-2 rounded-lg text-sm font-medium transition-colors",
              tab === value
                ? "bg-white shadow-sm text-slate-900"
                : "text-slate-600 hover:text-slate-900"
            )}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Alert cards */}
      {isLoading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="rounded-xl border border-slate-200 bg-white p-5 animate-pulse">
              <div className="h-4 bg-slate-200 rounded w-3/4 mb-3" />
              <div className="h-3 bg-slate-200 rounded w-1/2 mb-4" />
              <div className="h-8 bg-slate-200 rounded" />
            </div>
          ))}
        </div>
      ) : alerts.length === 0 ? (
        <div className="text-center py-16 text-slate-500">
          <CheckCircle className="w-12 h-12 mx-auto mb-4 text-emerald-300" />
          <p className="text-lg font-medium">{t("noAlerts")}</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-4">
          {alerts.map((alert) => (
            <div
              key={alert.assessment_id}
              className={cn(
                "rounded-xl border bg-white shadow-sm p-5 flex flex-col gap-4",
                alert.risk_level === "CRITICAL" ? "border-red-200" :
                alert.risk_level === "HIGH" ? "border-orange-200" :
                alert.risk_level === "MEDIUM" ? "border-yellow-200" : "border-green-200"
              )}
            >
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h3 className="font-semibold text-slate-900 text-sm leading-tight">{alert.product_name}</h3>
                  <p className="text-xs text-slate-400 mt-0.5">{alert.category ?? ""} {alert.batch_number ? `· ${alert.batch_number}` : ""}</p>
                </div>
                <span className={`rounded-full px-2.5 py-1 text-xs font-semibold flex-shrink-0 ${riskBadgeClass(alert.risk_level)}`}>
                  {tRisk(alert.risk_level as any)}
                </span>
              </div>

              <div className="grid grid-cols-2 gap-3 text-xs">
                <div className="bg-slate-50 rounded-lg p-2.5">
                  <div className="text-slate-500 mb-0.5">Expires</div>
                  <div className="font-semibold text-slate-900">{formatDate(alert.expiry_date)}</div>
                  <div className={cn("font-bold", alert.days_until_expiry <= 30 ? "text-red-600" : "text-slate-600")}>
                    {alert.days_until_expiry}d left
                  </div>
                </div>
                <div className="bg-slate-50 rounded-lg p-2.5">
                  <div className="text-slate-500 mb-0.5">Est. Loss</div>
                  <div className="font-bold text-red-600">{formatCurrency(alert.estimated_loss ?? 0)}</div>
                  <div className="text-slate-400">{alert.current_quantity} units</div>
                </div>
              </div>

              {alert.suggested_discount_percent && (
                <div className="bg-emerald-50 border border-emerald-200 rounded-lg px-3 py-2 text-xs text-emerald-800">
                  💡 Suggested: Apply {alert.suggested_discount_percent}% discount
                </div>
              )}

              {/* Action buttons */}
              <div className="grid grid-cols-2 gap-2">
                <button
                  onClick={() => actionMutation.mutate({
                    batchId: alert.batch_id,
                    actionType: "discount",
                    discountPct: alert.suggested_discount_percent,
                  })}
                  className="flex items-center justify-center gap-1.5 rounded-lg bg-emerald-600 px-3 py-2 text-xs font-semibold text-white hover:bg-emerald-700 transition-colors"
                >
                  <CheckCircle className="w-3.5 h-3.5" />
                  {t("applyDiscount")}
                </button>
                <button
                  onClick={() => actionMutation.mutate({ batchId: alert.batch_id, actionType: "transfer" })}
                  className="flex items-center justify-center gap-1.5 rounded-lg bg-blue-600 px-3 py-2 text-xs font-semibold text-white hover:bg-blue-700 transition-colors"
                >
                  <ArrowRightLeft className="w-3.5 h-3.5" />
                  {t("transfer")}
                </button>
                <button
                  onClick={() => actionMutation.mutate({ batchId: alert.batch_id, actionType: "return" })}
                  className="flex items-center justify-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                >
                  <RotateCcw className="w-3.5 h-3.5" />
                  {t("return")}
                </button>
                <button
                  onClick={() => actionMutation.mutate({ batchId: alert.batch_id, actionType: "dismiss" })}
                  className="flex items-center justify-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-500 hover:bg-slate-50 transition-colors"
                >
                  <XCircle className="w-3.5 h-3.5" />
                  {t("dismiss")}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
