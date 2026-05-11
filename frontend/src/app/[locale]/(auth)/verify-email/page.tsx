"use client";

import { useEffect, useState } from "react";
import { useSearchParams, useParams } from "next/navigation";
import Link from "next/link";
import { apiFetch } from "@/lib/api/client";

export default function VerifyEmailPage() {
  const { locale } = useParams();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");

  useEffect(() => {
    if (!token) {
      setStatus("error");
      return;
    }
    apiFetch("/auth/verify-email", {
      method: "POST",
      body: JSON.stringify({ token }),
    })
      .then(() => setStatus("success"))
      .catch(() => setStatus("error"));
  }, [token]);

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 to-slate-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="bg-white rounded-2xl shadow-xl p-8 text-center">
          {status === "loading" && (
            <>
              <div className="animate-spin w-12 h-12 border-4 border-emerald-600 border-t-transparent rounded-full mx-auto mb-4" />
              <p className="text-slate-600">Verifying your email...</p>
            </>
          )}
          {status === "success" && (
            <>
              <div className="w-14 h-14 bg-emerald-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h2 className="text-xl font-bold text-slate-900 mb-2">Email Verified!</h2>
              <p className="text-slate-500 mb-6">Your email has been successfully verified. All features are now unlocked.</p>
              <Link
                href={`/${locale}/dashboard`}
                className="inline-flex items-center justify-center rounded-lg bg-emerald-600 px-6 py-2.5 text-sm font-semibold text-white hover:bg-emerald-700"
              >
                Go to Dashboard
              </Link>
            </>
          )}
          {status === "error" && (
            <>
              <div className="w-14 h-14 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </div>
              <h2 className="text-xl font-bold text-slate-900 mb-2">Verification Failed</h2>
              <p className="text-slate-500 mb-6">Invalid or expired verification token.</p>
              <Link href={`/${locale}/login`} className="text-emerald-600 hover:text-emerald-700 font-medium">
                Back to Login
              </Link>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
