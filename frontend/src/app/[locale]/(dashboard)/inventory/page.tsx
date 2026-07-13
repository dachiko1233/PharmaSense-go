"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { differenceInDays, isToday } from "date-fns";
import { apiFetch } from "@/lib/api/client";
import { InventoryBatch } from "@/types";
import { formatCurrency, formatDate, riskBadgeClass, cn } from "@/lib/utils";
import { Search, Filter, Download, Trash2 } from "lucide-react";
import { toast } from "sonner";

const RISK_FILTERS = ["", "EXPIRED", "CRITICAL", "HIGH", "MEDIUM", "LOW"];

function expiryLabel(expiryDateStr: string, daysUntilExpiry?: number): { label: string; className: string } {
  const days = daysUntilExpiry ?? differenceInDays(new Date(expiryDateStr), new Date());

  if (days < 0) {
    return {
      label: `Expired ${Math.abs(days)} days ago`,
      className: "text-xs font-semibold text-red-700",
    };
  }
  if (isToday(new Date(expiryDateStr))) {
    return { label: "Expires TODAY", className: "text-xs font-bold text-red-600" };
  }
  if (days <= 30) {
    return { label: `Expires in ${days} days`, className: "text-xs text-red-500" };
  }
  return { label: `Expires in ${days} days`, className: "text-xs text-slate-400" };
}

export default function InventoryPage() {
  const t = useTranslations("inventory");
  const tRisk = useTranslations("risk");
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [riskFilter, setRiskFilter] = useState("");

  const { data, isLoading } = useQuery<{ data: InventoryBatch[]; total: number }>({
    queryKey: ["inventory", search, riskFilter],
    queryFn: () => {
      const params = new URLSearchParams();
      if (search) params.set("search", search);
      if (riskFilter) params.set("risk_level", riskFilter);
      return apiFetch(`/inventory?${params}`);
    },
  });

  const batches = data?.data ?? [];

  const deleteAllMutation = useMutation({
    mutationFn: () => apiFetch<{ deleted: number; zeroed: number; total: number }>("/inventory", { method: "DELETE" }),
    onSuccess: (res) => {
      toast.success(t("deleteAllSuccess", { deleted: res.deleted, zeroed: res.zeroed }));
      queryClient.invalidateQueries({ queryKey: ["inventory"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      queryClient.invalidateQueries({ queryKey: ["timeline"] });
      queryClient.invalidateQueries({ queryKey: ["alerts-critical"] });
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
    onError: () => toast.error(t("deleteAllFailed")),
  });

  const handleDeleteAll = () => {
    if (window.confirm(t("deleteAllConfirm"))) {
      deleteAllMutation.mutate();
    }
  };

  const exportCSV = () => {
    const rows = [
      ["Product", "Batch", "Expiry", "Quantity", "Purchase Price", "Selling Price", "Risk", "Supplier"].join(","),
      ...batches.map((b) =>
        [
          `"${b.product_name}"`,
          b.batch_number ?? "",
          formatDate(b.expiry_date),
          b.current_quantity,
          b.purchase_price,
          b.selling_price,
          b.risk_level ?? "—",
          b.supplier ?? "",
        ].join(",")
      ),
    ];
    const blob = new Blob([rows.join("\n")], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "inventory.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="p-6 lg:p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
          <p className="text-slate-500 text-sm mt-1">{t("subtitle")}</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={exportCSV}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-colors"
          >
            <Download className="w-4 h-4" />
            {t("exportCSV")}
          </button>
          <button
            onClick={handleDeleteAll}
            disabled={deleteAllMutation.isPending}
            className="inline-flex items-center gap-2 rounded-lg border border-red-200 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 transition-colors disabled:opacity-50"
          >
            <Trash2 className="w-4 h-4" />
            {t("deleteAll")}
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-col sm:flex-row gap-3 mb-6">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("searchPlaceholder")}
            className="w-full pl-10 pr-4 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
          />
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <Filter className="w-4 h-4 text-slate-400 flex-shrink-0" />
          {RISK_FILTERS.map((level) => (
            <button
              key={level || "all"}
              onClick={() => setRiskFilter(level)}
              className={cn(
                "px-3 py-1.5 rounded-full text-xs font-medium border transition-colors",
                riskFilter === level
                  ? level
                    ? riskBadgeClass(level) + " border-transparent"
                    : "bg-slate-900 text-white border-transparent"
                  : "bg-white text-slate-600 border-slate-200 hover:border-slate-300"
              )}
            >
              {level ? tRisk(level as any) : "All"}
            </button>
          ))}
        </div>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase tracking-wide">{t("product")}</th>
                <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase tracking-wide">{t("batch")}</th>
                <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase tracking-wide">{t("expiry")}</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase tracking-wide">{t("quantity")}</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase tracking-wide">{t("purchasePrice")}</th>
                <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase tracking-wide">{t("risk")}</th>
                <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase tracking-wide">{t("supplier")}</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                [...Array(8)].map((_, i) => (
                  <tr key={i} className="border-b border-slate-100">
                    {[...Array(7)].map((_, j) => (
                      <td key={j} className="py-3 px-4">
                        <div className="h-4 bg-slate-200 rounded animate-pulse" />
                      </td>
                    ))}
                  </tr>
                ))
              ) : batches.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-12 text-center text-slate-500">{t("noInventory")}</td>
                </tr>
              ) : (
                batches.map((batch) => {
                  const isExpired = batch.risk_level === "EXPIRED" || (batch.days_until_expiry != null && batch.days_until_expiry < 0);
                  const expiry = expiryLabel(batch.expiry_date, batch.days_until_expiry ?? undefined);

                  return (
                    <tr
                      key={batch.id}
                      className={cn(
                        "border-b border-slate-100",
                        isExpired
                          ? "bg-red-50 hover:bg-red-100"
                          : "hover:bg-slate-50"
                      )}
                    >
                      <td className="py-3 px-4">
                        <div className="font-medium text-slate-900">{batch.product_name}</div>
                        <div className="text-xs text-slate-400">{batch.category ?? "—"}</div>
                      </td>
                      <td className="py-3 px-4 text-slate-600 font-mono text-xs">{batch.batch_number ?? "—"}</td>
                      <td className="py-3 px-4">
                        <div className="text-slate-900">{formatDate(batch.expiry_date)}</div>
                        <div className={expiry.className}>{expiry.label}</div>
                      </td>
                      <td className="py-3 px-4 text-right font-medium text-slate-900">{batch.current_quantity}</td>
                      <td className="py-3 px-4 text-right text-slate-600">{formatCurrency(batch.purchase_price)}</td>
                      <td className="py-3 px-4">
                        {batch.risk_level ? (
                          <span className={`rounded-full px-2.5 py-1 text-xs font-semibold ${riskBadgeClass(batch.risk_level)}`}>
                            {tRisk(batch.risk_level as any)}
                          </span>
                        ) : (
                          <span className="text-slate-400 text-xs">—</span>
                        )}
                      </td>
                      <td className="py-3 px-4 text-slate-500 text-xs">{batch.supplier ?? "—"}</td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
        {batches.length > 0 && (
          <div className="px-4 py-3 border-t border-slate-100 text-xs text-slate-500">
            Showing {batches.length} batches
          </div>
        )}
      </div>
    </div>
  );
}
