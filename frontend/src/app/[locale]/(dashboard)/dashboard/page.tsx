"use client";

import { useQuery } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { apiFetch } from "@/lib/api/client";
import { formatCurrency, riskBadgeClass } from "@/lib/utils";
import { DashboardStats, TimelinePoint, RiskAssessment } from "@/types";
import {
  AreaChart, Area, BarChart, Bar, XAxis, YAxis,
  CartesianGrid, Tooltip, ResponsiveContainer
} from "recharts";
import { AlertTriangle, TrendingDown, TrendingUp, Package } from "lucide-react";
import { format } from "date-fns";

function KpiCard({ title, value, sub, icon: Icon, iconClass }: {
  title: string;
  value: string;
  sub?: string;
  icon: any;
  iconClass: string;
}) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
      <div className="flex items-center justify-between mb-4">
        <span className="text-sm font-medium text-slate-500">{title}</span>
        <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${iconClass}`}>
          <Icon className="w-5 h-5" />
        </div>
      </div>
      <div className="text-2xl font-bold text-slate-900">{value}</div>
      {sub && <div className="text-sm text-slate-500 mt-1">{sub}</div>}
    </div>
  );
}

export default function DashboardPage() {
  const t = useTranslations("dashboard");
  const tRisk = useTranslations("risk");

  const { data: stats, isLoading: statsLoading } = useQuery<DashboardStats>({
    queryKey: ["dashboard"],
    queryFn: () => apiFetch("/risk/dashboard"),
  });

  const { data: timeline } = useQuery<TimelinePoint[]>({
    queryKey: ["timeline"],
    queryFn: () => apiFetch("/risk/timeline"),
  });

  const { data: alerts } = useQuery<RiskAssessment[]>({
    queryKey: ["alerts-critical"],
    queryFn: () => apiFetch("/risk/assessments?risk_level=CRITICAL"),
  });

  const timelineData = timeline?.map((p) => ({
    month: format(new Date(p.month), "MMM"),
    batches: p.batch_count,
    value: Math.round(p.value),
  })) ?? [];

  const topAtRisk = alerts?.slice(0, 10).map((a) => ({
    name: a.product_name.length > 20 ? a.product_name.slice(0, 20) + "…" : a.product_name,
    loss: Math.round(a.estimated_loss ?? 0),
  })) ?? [];

  if (statsLoading) {
    return (
      <div className="p-6 lg:p-8">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="rounded-xl border border-slate-200 bg-white p-6 animate-pulse">
              <div className="h-4 bg-slate-200 rounded w-3/4 mb-4" />
              <div className="h-8 bg-slate-200 rounded w-1/2" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 lg:p-8">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
        <p className="text-slate-500 text-sm mt-1">{t("subtitle")}</p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <KpiCard
          title={t("criticalItems")}
          value={String(stats?.critical_count ?? 0)}
          sub={`${stats?.high_count ?? 0} high risk`}
          icon={AlertTriangle}
          iconClass="bg-red-100 text-red-600"
        />
        <KpiCard
          title={t("estimatedLoss")}
          value={formatCurrency(stats?.estimated_loss ?? 0)}
          sub="if no action taken"
          icon={TrendingDown}
          iconClass="bg-orange-100 text-orange-600"
        />
        <KpiCard
          title={t("potentialSavings")}
          value={formatCurrency(stats?.potential_savings ?? 0)}
          sub="with recommended actions"
          icon={TrendingUp}
          iconClass="bg-emerald-100 text-emerald-600"
        />
        <KpiCard
          title={t("inventoryValue")}
          value={formatCurrency(stats?.total_inventory_value ?? 0)}
          sub={`${stats?.total_batches ?? 0} active batches`}
          icon={Package}
          iconClass="bg-blue-100 text-blue-600"
        />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        {/* Expiry Timeline */}
        <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
          <h2 className="text-base font-semibold text-slate-900 mb-4">{t("expiryTimeline")}</h2>
          <ResponsiveContainer width="100%" height={240}>
            <AreaChart data={timelineData}>
              <defs>
                <linearGradient id="colorValue" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#059669" stopOpacity={0.1} />
                  <stop offset="95%" stopColor="#059669" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
              <XAxis dataKey="month" tick={{ fontSize: 12, fill: "#64748b" }} />
              <YAxis tick={{ fontSize: 12, fill: "#64748b" }} />
              <Tooltip
                contentStyle={{ borderRadius: "8px", border: "1px solid #e2e8f0", fontSize: "12px" }}
                formatter={(v) => [formatCurrency(v as number), "Value at risk"]}
              />
              <Area
                type="monotone"
                dataKey="value"
                stroke="#059669"
                strokeWidth={2}
                fill="url(#colorValue)"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Top At-Risk Products */}
        <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
          <h2 className="text-base font-semibold text-slate-900 mb-4">{t("topAtRisk")}</h2>
          <ResponsiveContainer width="100%" height={240}>
            <BarChart data={topAtRisk} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" horizontal={false} />
              <XAxis type="number" tick={{ fontSize: 11, fill: "#64748b" }} tickFormatter={(v) => `€${v}`} />
              <YAxis dataKey="name" type="category" width={120} tick={{ fontSize: 11, fill: "#64748b" }} />
              <Tooltip
                contentStyle={{ borderRadius: "8px", border: "1px solid #e2e8f0", fontSize: "12px" }}
                formatter={(v) => [formatCurrency(v as number), "Est. Loss"]}
              />
              <Bar dataKey="loss" fill="#ef4444" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Recent Critical Alerts */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
        <h2 className="text-base font-semibold text-slate-900 mb-4">{t("recentAlerts")}</h2>
        {!alerts?.length ? (
          <p className="text-slate-500 text-sm py-4 text-center">{t("noAlerts")}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200">
                  <th className="text-left py-2 px-3 text-xs font-medium text-slate-500 uppercase tracking-wide">Product</th>
                  <th className="text-left py-2 px-3 text-xs font-medium text-slate-500 uppercase tracking-wide">Risk</th>
                  <th className="text-left py-2 px-3 text-xs font-medium text-slate-500 uppercase tracking-wide">Days Left</th>
                  <th className="text-right py-2 px-3 text-xs font-medium text-slate-500 uppercase tracking-wide">Est. Loss</th>
                </tr>
              </thead>
              <tbody>
                {alerts.slice(0, 8).map((alert) => (
                  <tr key={alert.id} className="border-b border-slate-100 hover:bg-slate-50">
                    <td className="py-3 px-3 font-medium text-slate-900">{alert.product_name}</td>
                    <td className="py-3 px-3">
                      <span className={`rounded-full px-3 py-1 text-xs font-semibold ${riskBadgeClass(alert.risk_level)}`}>
                        {tRisk(alert.risk_level)}
                      </span>
                    </td>
                    <td className="py-3 px-3 text-slate-600">{alert.days_until_expiry}d</td>
                    <td className="py-3 px-3 text-right font-medium text-red-600">
                      {formatCurrency(alert.estimated_loss ?? 0)}
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
