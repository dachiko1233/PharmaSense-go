"use client";

import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import Link from "next/link";
import LoginForm from "@/components/auth/LoginForm";

export default function LoginPageClient() {
  const t = useTranslations("auth");
  const tFooter = useTranslations("footer");
  const params = useParams();
  const locale = params.locale as string;

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 to-slate-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="bg-white rounded-2xl shadow-xl p-8">
          <div className="mb-8 text-center">
            <div className="inline-flex items-center justify-center w-14 h-14 bg-emerald-600 rounded-xl mb-4">
              <svg className="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <h1 className="text-2xl font-bold text-slate-900">PharmaSense</h1>
            <p className="text-slate-500 text-sm mt-1">Expiry monitoring for pharmacies</p>
          </div>

          <LoginForm showForgotPassword />

          <p className="mt-4 text-center text-xs text-slate-500">
            {t("loginLegalNote")}{" "}
            <Link href={`/${locale}/privacy`} className="text-emerald-700 underline hover:text-emerald-800">
              {tFooter("privacy")}
            </Link>{" "}
            {t("and")}{" "}
            <Link href={`/${locale}/terms`} className="text-emerald-700 underline hover:text-emerald-800">
              {tFooter("terms")}
            </Link>
            .
          </p>
        </div>
      </div>
    </div>
  );
}
