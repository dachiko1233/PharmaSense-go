"use client";

import { useState } from "react";
import { useRouter, useParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { toast } from "sonner";
import { apiFetch } from "@/lib/api/client";
import { useAuth } from "@/lib/hooks/useAuth";

const schema = z.object({
  pharmacy_name: z.string().min(2),
  license_number: z.string().min(1),
  city: z.string().min(1),
  full_name: z.string().min(2),
  email: z.string().email(),
  password: z.string().min(8, "Password must be at least 8 characters"),
});

type FormData = z.infer<typeof schema>;

export default function SignupPage() {
  const t = useTranslations("auth");
  const router = useRouter();
  const { locale } = useParams();
  const { setAuth } = useAuth();
  const [loading, setLoading] = useState(false);

  const { register, handleSubmit, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  const onSubmit = async (data: FormData) => {
    setLoading(true);
    try {
      const res = await apiFetch<{ token: string; user: any; pharmacy: any }>("/auth/signup", {
        method: "POST",
        body: JSON.stringify(data),
      });
      setAuth(res.token, res.user, res.pharmacy);
      toast.success(t("signupSuccess"));
      router.push(`/${locale}/dashboard`);
    } catch (err: any) {
      toast.error(err.message ?? "Signup failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 to-slate-100 flex items-center justify-center p-4">
      <div className="w-full max-w-lg">
        <div className="bg-white rounded-2xl shadow-xl p-8">
          <div className="mb-6 text-center">
            <div className="inline-flex items-center justify-center w-14 h-14 bg-emerald-600 rounded-xl mb-4">
              <svg className="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <h1 className="text-2xl font-bold text-slate-900">Create your account</h1>
            <p className="text-slate-500 text-sm mt-1">Start monitoring expiry dates in minutes</p>
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="col-span-2">
                <label className="block text-sm font-medium text-slate-700 mb-1">{t("pharmacyName")}</label>
                <input
                  {...register("pharmacy_name")}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  placeholder="Nicosia Central Pharmacy"
                />
                {errors.pharmacy_name && <p className="text-red-500 text-xs mt-1">{errors.pharmacy_name.message}</p>}
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t("licenseNumber")}</label>
                <input
                  {...register("license_number")}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  placeholder="CY-PH-2024-001"
                />
                {errors.license_number && <p className="text-red-500 text-xs mt-1">{errors.license_number.message}</p>}
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">{t("city")}</label>
                <input
                  {...register("city")}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  placeholder="Nicosia"
                />
                {errors.city && <p className="text-red-500 text-xs mt-1">{errors.city.message}</p>}
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-medium text-slate-700 mb-1">{t("fullName")}</label>
                <input
                  {...register("full_name")}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  placeholder="Nikos Stavrou"
                />
                {errors.full_name && <p className="text-red-500 text-xs mt-1">{errors.full_name.message}</p>}
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-medium text-slate-700 mb-1">{t("email")}</label>
                <input
                  {...register("email")}
                  type="email"
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  placeholder="nikos@pharmacy.cy"
                />
                {errors.email && <p className="text-red-500 text-xs mt-1">{errors.email.message}</p>}
              </div>

              <div className="col-span-2">
                <label className="block text-sm font-medium text-slate-700 mb-1">{t("password")}</label>
                <input
                  {...register("password")}
                  type="password"
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
                {errors.password && <p className="text-red-500 text-xs mt-1">{errors.password.message}</p>}
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-emerald-700 disabled:opacity-50"
            >
              {loading ? (
                <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
              ) : null}
              {t("signup")}
            </button>
          </form>

          <div className="mt-6 text-center text-sm text-slate-500">
            {t("hasAccount")}{" "}
            <Link href={`/${locale}/login`} className="text-emerald-600 font-medium hover:text-emerald-700">
              {t("login")}
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
