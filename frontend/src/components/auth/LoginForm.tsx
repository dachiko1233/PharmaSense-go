"use client";

import { useState } from "react";
import { useRouter, useParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useTranslations } from "next-intl";
import Link from "next/link";
import { toast } from "sonner";
import { Mail, Lock, LogIn } from "lucide-react";
import { apiFetch } from "@/lib/api/client";
import { useAuth } from "@/lib/hooks/useAuth";

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});

type FormData = z.infer<typeof schema>;

interface LoginFormProps {
  showForgotPassword?: boolean;
  showSignupLink?: boolean;
}

export default function LoginForm({ showForgotPassword = false, showSignupLink = true }: LoginFormProps) {
  const t = useTranslations("auth");
  const router = useRouter();
  const params = useParams();
  const locale = params.locale as string;
  const { setAuth } = useAuth();
  const [loading, setLoading] = useState(false);

  const { register, handleSubmit, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  const onSubmit = async (data: FormData) => {
    setLoading(true);
    try {
      const res = await apiFetch<{ token: string; user: any; pharmacy: any }>("/auth/login", {
        method: "POST",
        body: JSON.stringify(data),
      });
      setAuth(res.token, res.user, res.pharmacy);
      toast.success(t("loginSuccess"));
      router.push(`/${locale}/dashboard`);
    } catch (err: any) {
      toast.error(err.message ?? "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1">{t("email")}</label>
        <div className="relative">
          <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            {...register("email")}
            type="email"
            autoComplete="email"
            className="w-full rounded-lg border border-slate-300 py-2 pl-10 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            placeholder="you@pharmacy.com"
          />
        </div>
        {errors.email && <p className="text-red-500 text-xs mt-1">{errors.email.message}</p>}
      </div>

      <div>
        <label className="block text-sm font-medium text-slate-700 mb-1">{t("password")}</label>
        <div className="relative">
          <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            {...register("password")}
            type="password"
            autoComplete="current-password"
            className="w-full rounded-lg border border-slate-300 py-2 pl-10 pr-3 text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
          />
        </div>
        {errors.password && <p className="text-red-500 text-xs mt-1">{errors.password.message}</p>}
      </div>

      {showForgotPassword && (
        <div className="text-right">
          <Link href={`/${locale}/forgot-password`} className="text-sm text-emerald-600 hover:text-emerald-700">
            {t("forgotPassword")}
          </Link>
        </div>
      )}

      <button
        type="submit"
        disabled={loading}
        className="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {loading ? (
          <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
        ) : (
          <LogIn className="h-4 w-4" />
        )}
        {t("login")}
      </button>

      {showSignupLink && (
        <div className="text-center text-sm text-slate-500">
          {t("noAccount")}{" "}
          <Link href={`/${locale}/signup`} className="text-emerald-600 font-medium hover:text-emerald-700">
            {t("signup")}
          </Link>
        </div>
      )}
    </form>
  );
}
