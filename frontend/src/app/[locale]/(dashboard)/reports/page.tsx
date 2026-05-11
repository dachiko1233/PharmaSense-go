"use client";

import { useQuery } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { apiFetch } from "@/lib/api/client";
import { SavingsPoint, WastePoint, CategoryStat } from "@/types";
import { formatCurrency } from "@/lib/utils";
import {
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  LineChart, Line
} from "recharts";
import { format } from "date-fns";

export default function ReportsPage() {
  const t = useTranslations("reports");

  const { data: savings = [] } = useQuery<SavingsPoint[]>({
    queryKey: ["savings"],
    queryFn: () => apiFetch("/reports/savings"),
  });

  const { data: waste = [] } = useQuery<WastePoint[]>({
    queryKey: ["waste"],
    queryFn: () => apiFetch("/reports/waste"),
  });

  const { data: categories = [] } = useQuery<CategoryStat[]>({
    queryKey: ["categories"],
    queryFn: () => apiFetch("/reports/categories"),
  });

  const savingsData = savings.map((s) => ({
    month: format(new Date(s.month), "MMM yy"),
    savings: Math.round(s.savings),
    actions: s.actions_taken,
  }));

  const wasteData = waste.map((w) => ({
    month: format(new Date(w.month), "MMM yy"),
    value: Math.round(w.waste_value),
    batches: w.expired_batches,
  }));

  return (
    <div className="p-6 lg:p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* Savings Chart */}
        <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
          <h2 className="text-base font-semibold text-slate-900 mb-4">{t("savings")}</h2>
          {savingsData.length === 0 ? (
            <p className="text-center text-slate-400 py-12">{t("noData")}</p>
          ) : (
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={savingsData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                <XAxis dataKey="month" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} tickFormatter={(v) => `€${v}`} />
                <Tooltip formatter={(v) => [formatCurrency(v as number), "Saved"]} />
                <Bar dataKey="savings" fill="#059669" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* Waste Trend */}
        <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
          <h2 className="text-base font-semibold text-slate-900 mb-4">{t("waste")}</h2>
          {wasteData.length === 0 ? (
            <p className="text-center text-slate-400 py-12">{t("noData")}</p>
          ) : (
            <ResponsiveContainer width="100%" height={240}>
              <LineChart data={wasteData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                <XAxis dataKey="month" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} tickFormatter={(v) => `€${v}`} />
                <Tooltip formatter={(v) => [formatCurrency(v as number), "Waste Value"]} />
                <Line type="monotone" dataKey="value" stroke="#ef4444" strokeWidth={2} dot={{ r: 4 }} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>
      </div>

      {/* Categories Table */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
        <h2 className="text-base font-semibold text-slate-900 mb-4">{t("categories")}</h2>
        {categories.length === 0 ? (
          <p className="text-center text-slate-400 py-8">{t("noData")}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 border-b border-slate-200">
                <tr>
                  <th className="text-left py-3 px-4 text-xs font-semibold text-slate-500 uppercase">Category</th>
                  <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase">Batches</th>
                  <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase">At Risk</th>
                  <th className="text-right py-3 px-4 text-xs font-semibold text-slate-500 uppercase">Total Loss</th>
                </tr>
              </thead>
              <tbody>
                {categories.map((cat) => (
                  <tr key={cat.category} className="border-b border-slate-100 hover:bg-slate-50">
                    <td className="py-3 px-4 font-medium text-slate-900">{cat.category}</td>
                    <td className="py-3 px-4 text-right text-slate-600">{cat.batch_count}</td>
                    <td className="py-3 px-4 text-right">
                      <span className={cat.at_risk_count > 0 ? "text-red-600 font-semibold" : "text-slate-400"}>
                        {cat.at_risk_count}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-right font-medium text-red-600">
                      {formatCurrency(cat.total_loss)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
