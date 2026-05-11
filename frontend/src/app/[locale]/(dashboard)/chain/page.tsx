"use client";

import { useQuery } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { apiFetch } from "@/lib/api/client";
import { useAuth } from "@/lib/hooks/useAuth";
import { formatCurrency, planBadgeClass, cn } from "@/lib/utils";
import { Building2, AlertTriangle, Package, TrendingDown } from "lucide-react";

interface ChainPharmacy {
  id: string;
  name: string;
  plan: string;
  city?: string;
  total_batches: number;
  critical_count: number;
  high_count: number;
  estimated_loss: number;
  inventory_value: number;
}

interface ChainDashboardData {
  chain_id: string;
  pharmacies: ChainPharmacy[];
  totals: {
    total_batches: number;
    critical_count: number;
    estimated_loss: number;
  };
}

export default function ChainPage() {
  const t = useTranslations("chain");
  const { pharmacy, role } = useAuth();

  // In a real app, we'd store the chain_id from the token or pharmacy
  // For demo, we fetch it from the first pharmacy's chain_id
  const chainId = "demo"; // placeholder

  const { data, isLoading } = useQuery<ChainDashboardData>({
    queryKey: ["chain-dashboard"],
    queryFn: async () => {
      // First get the pharmacy to get chain_id
      const pharmacies = await apiFetch<any[]>("/pharmacies");
      const chainPharm = pharmacies.find((p: any) => p.role === "chain_admin");
      if (!chainPharm) throw new Error("No chain found");

      // Get chain_id from pharmacy - for demo we use a different endpoint
      // In production, chain_id comes from the pharmacy data
      const pharmaDetail = await apiFetch<any>(`/pharmacies/${chainPharm.id}`);
      const cid = pharmaDetail.chain_id;
      if (!cid) throw new Error("Not part of a chain");

      return apiFetch(`/chains/${cid}/dashboard`);
    },
    enabled: role === "chain_admin",
  });

  if (role !== "chain_admin") {
    return (
      <div className="p-6 lg:p-8 flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <Building2 className="w-12 h-12 mx-auto mb-4 text-slate-300" />
          <h2 className="text-lg font-semibold text-slate-700">Chain Admin Access Required</h2>
          <p className="text-slate-500 text-sm mt-1">This page is only accessible to chain administrators.</p>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="p-6 lg:p-8">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="rounded-xl border border-slate-200 bg-white p-6 animate-pulse">
              <div className="h-4 bg-slate-200 rounded w-3/4 mb-4" />
              <div className="h-8 bg-slate-200 rounded w-1/2" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  const pharmacies = data?.pharmacies ?? [];
  const totals = data?.totals;

  return (
    <div className="p-6 lg:p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
        <p className="text-slate-500 text-sm mt-1">{t("subtitle")}</p>
      </div>

      {/* Total KPIs */}
      {totals && (
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
          <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
            <div className="flex items-center gap-3 mb-3">
              <div className="w-9 h-9 bg-blue-100 rounded-lg flex items-center justify-center">
                <Package className="w-5 h-5 text-blue-600" />
              </div>
              <span className="text-sm text-slate-500">{t("totalBatches")}</span>
            </div>
            <div className="text-2xl font-bold text-slate-900">{totals.total_batches}</div>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
            <div className="flex items-center gap-3 mb-3">
              <div className="w-9 h-9 bg-red-100 rounded-lg flex items-center justify-center">
                <AlertTriangle className="w-5 h-5 text-red-600" />
              </div>
              <span className="text-sm text-slate-500">{t("criticalItems")}</span>
            </div>
            <div className="text-2xl font-bold text-red-600">{totals.critical_count}</div>
          </div>
          <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
            <div className="flex items-center gap-3 mb-3">
              <div className="w-9 h-9 bg-orange-100 rounded-lg flex items-center justify-center">
                <TrendingDown className="w-5 h-5 text-orange-600" />
              </div>
              <span className="text-sm text-slate-500">{t("estimatedLoss")}</span>
            </div>
            <div className="text-2xl font-bold text-orange-600">{formatCurrency(totals.estimated_loss)}</div>
          </div>
        </div>
      )}

      {/* Pharmacy table */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-200">
          <h2 className="font-semibold text-slate-900">Pharmacies in Chain</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase">{t("pharmacy")}</th>
                <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase">{t("city")}</th>
                <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase">{t("plan")}</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase">Batches</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase">Critical</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase">Est. Loss</th>
                <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase">Inventory Value</th>
              </tr>
            </thead>
            <tbody>
              {pharmacies.length === 0 ? (
                <tr>
                  <td colSpan={7} className="py-12 text-center text-slate-500">No pharmacies in chain</td>
                </tr>
              ) : (
                pharmacies.map((p) => (
                  <tr key={p.id} className="border-b border-slate-100 hover:bg-slate-50">
                    <td className="py-3 px-4 font-medium text-slate-900">{p.name}</td>
                    <td className="py-3 px-4 text-slate-600">{p.city ?? "—"}</td>
                    <td className="py-3 px-4">
                      <span className={cn("text-xs px-2.5 py-1 rounded-full font-semibold", planBadgeClass(p.plan))}>
                        {p.plan}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-right text-slate-600">{p.total_batches}</td>
                    <td className="py-3 px-4 text-right">
                      <span className={p.critical_count > 0 ? "text-red-600 font-bold" : "text-slate-400"}>
                        {p.critical_count}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-red-600">
                      {formatCurrency(p.estimated_loss)}
                    </td>
                    <td className="py-3 px-4 text-right text-slate-600">
                      {formatCurrency(p.inventory_value)}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
