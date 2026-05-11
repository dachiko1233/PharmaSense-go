"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { apiFetch } from "@/lib/api/client";
import { NotificationSettings } from "@/types";
import { useForm } from "react-hook-form";
import { useEffect } from "react";
import { toast } from "sonner";
import { Bell, MessageSquare } from "lucide-react";

export default function SettingsPage() {
  const t = useTranslations("settings");
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery<NotificationSettings>({
    queryKey: ["notification-settings"],
    queryFn: () => apiFetch("/settings/notifications"),
  });

  const { register, handleSubmit, setValue, watch } = useForm({
    defaultValues: {
      sms_enabled: false,
      phone_number: "",
    },
  });

  const smsEnabled = watch("sms_enabled");

  useEffect(() => {
    if (data) {
      setValue("sms_enabled", data.sms_enabled);
      setValue("phone_number", data.phone_number ?? "");
    }
  }, [data]);

  const saveMutation = useMutation({
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

  const onSubmit = (formData: any) => {
    saveMutation.mutate({
      sms_enabled: formData.sms_enabled,
      phone_number: formData.phone_number || undefined,
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

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Email notifications */}
        <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-9 h-9 bg-blue-100 rounded-lg flex items-center justify-center">
              <Bell className="w-5 h-5 text-blue-600" />
            </div>
            <h2 className="font-semibold text-slate-900">{t("emailNotifications")}</h2>
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
              Daily digest at 8 AM (Pro plan)
            </div>
          </div>

          {data && (
            <div className="mt-4 text-xs text-slate-500">
              Email: <span className="font-medium text-slate-700">{data.email}</span>
              {!data.email_verified && (
                <span className="ml-2 bg-amber-100 text-amber-700 px-2 py-0.5 rounded-full">unverified</span>
              )}
            </div>
          )}
        </div>

        {/* SMS notifications */}
        <div className="rounded-xl border border-slate-200 bg-white shadow-sm p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-9 h-9 bg-emerald-100 rounded-lg flex items-center justify-center">
              <MessageSquare className="w-5 h-5 text-emerald-600" />
            </div>
            <h2 className="font-semibold text-slate-900">{t("smsNotifications")}</h2>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-slate-700">{t("smsEnabled")}</div>
                <div className="text-xs text-slate-400 mt-0.5">Get SMS alerts for CRITICAL expiry items (Pro plan)</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  {...register("sms_enabled")}
                  type="checkbox"
                  className="sr-only peer"
                />
                <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-emerald-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-emerald-600"></div>
              </label>
            </div>

            {smsEnabled && (
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t("phoneNumber")}</label>
                <input
                  {...register("phone_number")}
                  type="tel"
                  placeholder={t("phonePlaceholder")}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
                <p className="text-xs text-slate-400 mt-1">Use E.164 format: +357 99 000000</p>
              </div>
            )}
          </div>
        </div>

        <button
          type="submit"
          disabled={saveMutation.isPending}
          className="inline-flex items-center gap-2 rounded-lg bg-emerald-600 px-6 py-2.5 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
        >
          {saveMutation.isPending ? (
            <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          ) : null}
          {t("save")}
        </button>
      </form>
    </div>
  );
}
