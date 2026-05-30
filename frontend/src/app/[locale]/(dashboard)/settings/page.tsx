"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { apiFetch } from "@/lib/api/client";
import { NotificationSettings } from "@/types";
import { useForm } from "react-hook-form";
import { useEffect } from "react";
import { toast } from "sonner";
import { Bell, Mail, MessageSquare } from "lucide-react";

function formatDigestTime(digestTime: string | undefined): string {
  if (!digestTime) return "8:00 AM";
  const hour = parseInt(digestTime.split(":")[0], 10);
  const period = hour >= 12 ? "PM" : "AM";
  const display = hour % 12 || 12;
  return `${display}:00 ${period}`;
}

const DIGEST_HOUR_OPTIONS = [
  { value: "06:00", label: "6:00 AM" },
  { value: "07:00", label: "7:00 AM" },
  { value: "08:00", label: "8:00 AM (recommended)" },
  { value: "09:00", label: "9:00 AM" },
  { value: "12:00", label: "12:00 PM (noon)" },
  { value: "15:00", label: "3:00 PM" },
  { value: "18:00", label: "6:00 PM" },
  { value: "20:00", label: "8:00 PM" },
];

export default function SettingsPage() {
  const t = useTranslations("settings");
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery<NotificationSettings>({
    queryKey: ["notification-settings"],
    queryFn: () => apiFetch("/settings/notifications"),
  });

  // ── SMS form ──────────────────────────────────────────────────────
  const smsForm = useForm({
    defaultValues: { sms_enabled: false, phone_number: "" },
  });
  const smsEnabled = smsForm.watch("sms_enabled");

  useEffect(() => {
    if (data) {
      smsForm.setValue("sms_enabled", data.sms_enabled);
      smsForm.setValue("phone_number", data.phone_number ?? "");
    }
  }, [data]);

  const smsMutation = useMutation({
    mutationFn: (payload: { sms_enabled: boolean; phone_number?: string }) =>
      apiFetch("/settings/notifications", {
        method: "PATCH",
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      toast.success(t("saved"));
      queryClient.invalidateQueries({ queryKey: ["notification-settings"] });
    },
    onError: () => toast.error("Failed to save settings"),
  });

  const onSMSSubmit = (formData: any) => {
    smsMutation.mutate({
      sms_enabled: formData.sms_enabled,
      phone_number: formData.phone_number || undefined,
    });
  };

  // ── Email Preferences form ────────────────────────────────────────
  const digestForm = useForm({
    defaultValues: { daily_digest_enabled: true, digest_time: "08:00" },
  });

  useEffect(() => {
    if (data) {
      digestForm.setValue("daily_digest_enabled", data.daily_digest_enabled);
      digestForm.setValue("digest_time", (data.digest_time ?? "08:00").slice(0, 5));
    }
  }, [data]);

  const digestMutation = useMutation({
    mutationFn: (payload: {
      daily_digest_enabled: boolean;
      digest_time: string;
    }) =>
      apiFetch("/settings/notifications", {
        method: "PATCH",
        body: JSON.stringify(payload),
      }),
    onSuccess: () => {
      toast.success(t("emailPreferences.saved"));
      queryClient.invalidateQueries({ queryKey: ["notification-settings"] });
    },
    onError: () => toast.error("Failed to save preferences"),
  });

  const onDigestSubmit = (formData: any) => {
    digestMutation.mutate({
      daily_digest_enabled: formData.daily_digest_enabled,
      digest_time: formData.digest_time,
    });
  };

  if (isLoading) {
    return (
      <div className="p-6 lg:p-8">
        <div className="h-8 bg-slate-200 rounded w-48 mb-6 animate-pulse" />
        <div className="rounded-xl border border-slate-200 bg-white p-6 animate-pulse space-y-4">
          <div className="h-4 bg-slate-200 rounded w-3/4" />
          <div className="h-4 bg-slate-200 rounded w-1/2" />
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 lg:p-8 max-w-2xl">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
      </div>

      <div className="space-y-6">
        {/* ── Email Preferences ──────────────────────────────────── */}
        <form
          onSubmit={digestForm.handleSubmit(onDigestSubmit)}
          className="rounded-xl border border-slate-200 bg-white shadow-sm p-6"
        >
          <div className="flex items-center gap-3 mb-5">
            <div className="w-9 h-9 bg-emerald-100 rounded-lg flex items-center justify-center">
              <Mail className="w-5 h-5 text-emerald-600" />
            </div>
            <h2 className="font-semibold text-slate-900">
              {t("emailPreferences.title")}
            </h2>
          </div>

          <div className="space-y-4">
            {/* Daily digest toggle */}
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="text-sm font-medium text-slate-700">
                  {t("emailPreferences.dailyDigest")}
                </div>
                <div className="text-xs text-slate-400 mt-0.5">
                  {t("emailPreferences.dailyDigestHelp")}
                </div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer shrink-0">
                <input
                  {...digestForm.register("daily_digest_enabled")}
                  type="checkbox"
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-emerald-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-600" />
              </label>
            </div>

            {/* Delivery time */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                {t("emailPreferences.digestTime")}
              </label>
              <select
                {...digestForm.register("digest_time")}
                className="px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 bg-white"
              >
                {DIGEST_HOUR_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
              <p className="text-xs text-slate-400 mt-1">
                {t("emailPreferences.digestTimeHelp")}
              </p>
            </div>
          </div>

          <div className="mt-5">
            <button
              type="submit"
              disabled={digestMutation.isPending}
              className="inline-flex items-center gap-2 rounded-lg bg-emerald-600 px-5 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
            >
              {digestMutation.isPending && (
                <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
              )}
              {t("emailPreferences.save")}
            </button>
          </div>
        </form>

        {/* ── Email Notifications (info only) ───────────────────── */}
        <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-9 h-9 bg-blue-100 rounded-lg flex items-center justify-center">
              <Bell className="w-5 h-5 text-blue-600" />
            </div>
            <h2 className="font-semibold text-slate-900">
              {t("emailNotifications")}
            </h2>
          </div>

          <div className="space-y-3 text-sm text-slate-600">
            <div className="flex items-center gap-2">
              <div className="w-1.5 h-1.5 bg-emerald-500 rounded-full" />
              Welcome email on signup
            </div>
            <div className="flex items-center gap-2">
              <div className="w-1.5 h-1.5 bg-emerald-500 rounded-full" />
              Password reset emails
            </div>
            <div className="flex items-center gap-2">
              <div className="w-1.5 h-1.5 bg-emerald-500 rounded-full" />
              {t("emailPreferences.digestAt", { time: formatDigestTime(data?.digest_time) })}
            </div>
          </div>

          {data && (
            <div className="mt-4 text-xs text-slate-500">
              Email:{" "}
              <span className="font-medium text-slate-700">{data.email}</span>
              {!data.email_verified && (
                <span className="ml-2 bg-amber-100 text-amber-700 px-2 py-0.5 rounded-full">
                  unverified
                </span>
              )}
            </div>
          )}
        </div>

        {/* ── SMS Notifications ─────────────────────────────────── */}
        <form
          onSubmit={smsForm.handleSubmit(onSMSSubmit)}
          className="rounded-xl border border-slate-200 bg-white shadow-sm p-6"
        >
          <div className="flex items-center gap-3 mb-4">
            <div className="w-9 h-9 bg-emerald-100 rounded-lg flex items-center justify-center">
              <MessageSquare className="w-5 h-5 text-emerald-600" />
            </div>
            <h2 className="font-semibold text-slate-900">
              {t("smsNotifications")}
            </h2>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-slate-700">
                  {t("smsEnabled")}
                </div>
                <div className="text-xs text-slate-400 mt-0.5">
                  Get SMS alerts for CRITICAL expiry items (Pro plan)
                </div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  {...smsForm.register("sms_enabled")}
                  type="checkbox"
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-emerald-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-600" />
              </label>
            </div>

            {smsEnabled && (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  {t("phoneNumber")}
                </label>
                <input
                  {...smsForm.register("phone_number")}
                  type="tel"
                  placeholder={t("phonePlaceholder")}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
                <p className="text-xs text-slate-400 mt-1">
                  Use E.164 format: +357 99 000000
                </p>
              </div>
            )}
          </div>

          <div className="mt-5">
            <button
              type="submit"
              disabled={smsMutation.isPending}
              className="inline-flex items-center gap-2 rounded-lg bg-emerald-600 px-5 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
            >
              {smsMutation.isPending && (
                <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
              )}
              {t("save")}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
