"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { useParams } from "next/navigation";
import { apiFetch } from "@/lib/api/client";
import { formatCurrency, formatDate, riskBadgeClass, cn } from "@/lib/utils";
import { toast } from "sonner";
import { CheckCircle, ArrowRightLeft, RotateCcw, XCircle, Trash2 } from "lucide-react";
import { ReturnSupplierModal } from "./ReturnSupplierModal";

type Tab = "" | "EXPIRED" | "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

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

interface ReturnActionResponse {
  id: string;
  message: string;
  refund_value: number;
  quantity: number;
}

export default function AlertsPage() {
  const t = useTranslations("alerts");
  const tModal = useTranslations("alerts.returnModal");
  const tRisk = useTranslations("risk");
  const queryClient = useQueryClient();
  const { locale } = useParams();
  const fmtCurrency = (amount: number) => formatCurrency(amount, "EUR", locale as string);

  const [tab, setTab] = useState<Tab>("EXPIRED");
  const [returnTarget, setReturnTarget] = useState<AlertRow | null>(null);

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
      queryClient.invalidateQueries({ queryKey: ["inventory"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: () => toast.error("Action failed"),
  });

  const returnMutation = useMutation({
    mutationFn: ({ batchId, quantity, notes }: { batchId: string; quantity: number; notes?: string }) =>
      apiFetch<ReturnActionResponse>(`/alerts/${batchId}/action`, {
        method: "POST",
        body: JSON.stringify({ action_type: "RETURN", quantity, notes }),
      }),
    onSuccess: (data) => {
      const refundStr = fmtCurrency(data.refund_value);
      toast.success(tModal("success", { quantity: data.quantity, amount: refundStr }));
      setReturnTarget(null);
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: () => {
      toast.error("Return failed");
    },
  });

  const handleReturnConfirm = (quantity: number, notes?: string) => {
    if (!returnTarget) return;
    returnMutation.mutate({ batchId: returnTarget.batch_id, quantity, notes });
  };

  const tabs: { label: string; value: Tab }[] = [
    { label: t("expired"), value: "EXPIRED" },
    { label: t("critical"), value: "CRITICAL" },
    { label: t("high"), value: "HIGH" },
    { label: t("medium"), value: "MEDIUM" },
    { label: t("low"), value: "LOW" },
    { label: t("all"), value: "" },
  ];

  return (
    <div className="p-6 lg:p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
        <p className="text-slate-500 text-sm mt-1">{t("subtitle")}</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 bg-slate-100 p-1 rounded-xl w-fit flex-wrap">
        {tabs.map(({ label, value }) => (
          <button
            key={value || "all"}
            onClick={() => setTab(value)}
            className={cn(
              "px-4 py-2 rounded-lg text-sm font-medium transition-colors",
              tab === value
                ? value === "EXPIRED"
                  ? "bg-red-700 text-white shadow-sm"
                  : "bg-white shadow-sm text-slate-900"
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
          {alerts.map((alert) => {
            const isExpired = alert.risk_level === "EXPIRED";
            const daysAgo = isExpired ? Math.abs(alert.days_until_expiry) : null;

            return (
              <div
                key={alert.assessment_id}
                className={cn(
                  "rounded-xl border bg-white shadow-sm p-5 flex flex-col gap-4",
                  isExpired ? "border-red-800 bg-red-50" :
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
                  <div className={cn("rounded-lg p-2.5", isExpired ? "bg-red-100" : "bg-slate-50")}>
                    <div className="text-slate-500 mb-0.5">{isExpired ? "Expired" : "Expires"}</div>
                    <div className="font-semibold text-slate-900">{formatDate(alert.expiry_date)}</div>
                    <div className={cn("font-bold", isExpired ? "text-red-700" : alert.days_until_expiry <= 30 ? "text-red-600" : "text-slate-600")}>
                      {isExpired
                        ? `${daysAgo} day${daysAgo !== 1 ? "s" : ""} ago`
                        : `${alert.days_until_expiry}d left`}
                    </div>
                  </div>
                  <div className="bg-slate-50 rounded-lg p-2.5">
                    <div className="text-slate-500 mb-0.5">Est. Loss</div>
                    <div className="font-bold text-red-600">{fmtCurrency(alert.estimated_loss ?? 0)}</div>
                    <div className="text-slate-400">{alert.current_quantity} units</div>
                  </div>
                </div>

                {isExpired ? (
                  <div className="bg-red-100 border border-red-300 rounded-lg px-3 py-2 text-xs text-red-800 font-medium">
                    ⚠️ {t("removeFromShelfWarning")}
                  </div>
                ) : alert.suggested_discount_percent ? (
                  <div className="bg-emerald-50 border border-emerald-200 rounded-lg px-3 py-2 text-xs text-emerald-800">
                    💡 Suggested: Apply {alert.suggested_discount_percent}% discount
                  </div>
                ) : null}

                {/* Action buttons */}
                {isExpired ? (
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      onClick={() => actionMutation.mutate({ batchId: alert.batch_id, actionType: "dispose" })}
                      className="flex items-center justify-center gap-1.5 rounded-lg bg-red-700 px-3 py-2 text-xs font-semibold text-white hover:bg-red-800 transition-colors"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                      {t("dispose")}
                    </button>
                    <button
                      onClick={() => setReturnTarget(alert)}
                      className="flex items-center justify-center gap-1.5 rounded-lg border border-slate-200 px-3 py-2 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                    >
                      <RotateCcw className="w-3.5 h-3.5" />
                      {t("return")}
                    </button>
                  </div>
                ) : (
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
                      onClick={() => setReturnTarget(alert)}
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
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Return to Supplier Modal */}
      <ReturnSupplierModal
        open={returnTarget !== null}
        onClose={() => setReturnTarget(null)}
        onConfirm={handleReturnConfirm}
        productName={returnTarget?.product_name ?? ""}
        currentQuantity={returnTarget?.current_quantity ?? 1}
        purchasePrice={returnTarget?.purchase_price ?? 0}
        locale={locale as string}
        isLoading={returnMutation.isPending}
      />
    </div>
  );
}
