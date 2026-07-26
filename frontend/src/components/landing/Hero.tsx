"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Play } from "lucide-react";
import { apiFetch } from "@/lib/api/client";
import { useAuth } from "@/lib/hooks/useAuth";
import LocaleSwitcher from "@/components/layout/LocaleSwitcher";
import LoginForm from "@/components/auth/LoginForm";

const DEMO_EMAIL = "admin@pharmasense.cy";
const DEMO_PASSWORD = "Demo1234!";

export default function Hero() {
  const t = useTranslations("landing.hero");
  const { locale } = useParams();
  const lc = (locale as string) ?? "en";
  const router = useRouter();
  const { setAuth } = useAuth();
  const [loading, setLoading] = useState(false);

  const handleTryDemo = async () => {
    setLoading(true);
    try {
      const res = await apiFetch<{ token: string; user: any; pharmacy: any }>("/auth/login", {
        method: "POST",
        body: JSON.stringify({ email: DEMO_EMAIL, password: DEMO_PASSWORD }),
      });
      setAuth(res.token, res.user, res.pharmacy);
      router.push(`/${lc}/dashboard`);
    } catch {
      toast.error("Demo login failed. Please try again.");
      setLoading(false);
    }
  };

  return (
    <div className="relative bg-gradient-to-br from-emerald-50 to-slate-100">
      <div className="absolute right-6 top-5 z-10">
        <LocaleSwitcher />
      </div>

      <div className="mx-auto flex max-w-6xl flex-col items-center gap-12 px-6 py-16 md:flex-row md:items-center md:gap-10 md:py-24">
        {/* Left: brand message */}
        <div className="w-full md:flex-[1.4]">
          <div className="mb-6 flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-600">
              <svg className="h-7 w-7 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <span className="text-3xl font-extrabold tracking-tight text-emerald-700 sm:text-4xl">
              PharmaSense
            </span>
          </div>

          <h1 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-4xl md:text-5xl">
            {t("headline")}
          </h1>
          <p className="mx-auto mt-5 max-w-xl text-lg text-slate-600 md:mx-0">
            {t("subtitle")}
          </p>
        </div>

        {/* Right: login card */}
        <div className="w-full max-w-sm md:flex-1">
          <div className="rounded-2xl bg-white p-8 shadow-[0_20px_50px_-12px_rgba(16,185,129,0.18)] ring-1 ring-slate-900/5">
            <h2 className="mb-6 text-xl font-bold text-slate-900">{t("loginHeading")}</h2>

            <LoginForm showForgotPassword showSignupLink={false} />

            <div className="my-6 flex items-center gap-3">
              <div className="h-px flex-1 bg-slate-200" />
              <span className="text-xs font-medium text-slate-400">or</span>
              <div className="h-px flex-1 bg-slate-200" />
            </div>

            <button
              type="button"
              onClick={handleTryDemo}
              disabled={loading}
              className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-emerald-600 bg-white px-4 py-2.5 text-sm font-semibold text-emerald-700 transition-colors hover:bg-emerald-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? (
                <svg className="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
              ) : (
                <Play className="h-4 w-4 fill-current" />
              )}
              {t("tryDemo")}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
