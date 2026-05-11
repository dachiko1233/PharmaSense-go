"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { toast } from "sonner";
import { apiFetch } from "@/lib/api/client";

const schema = z.object({ email: z.string().email() });
type FormData = z.infer<typeof schema>;

export default function ForgotPasswordPage() {
  const { locale } = useParams();
  const [sent, setSent] = useState(false);
  const { register, handleSubmit, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  const onSubmit = async (data: FormData) => {
    try {
      await apiFetch("/auth/forgot-password", {
        method: "POST",
        body: JSON.stringify(data),
      });
      setSent(true);
    } catch {
      toast.error("Failed to send reset email");
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 to-slate-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="bg-white rounded-2xl shadow-xl p-8">
          <h1 className="text-2xl font-bold text-slate-900 mb-2">Reset Password</h1>
          <p className="text-slate-500 text-sm mb-6">Enter your email and we&apos;ll send you a reset link.</p>

          {sent ? (
            <div className="bg-emerald-50 border border-emerald-200 rounded-lg p-4 text-emerald-800 text-sm">
              If that email exists, a reset link has been sent. Check your inbox.
            </div>
          ) : (
            <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Email</label>
                <input
                  {...register("email")}
                  type="email"
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-emerald-500"
                />
                {errors.email && <p className="text-red-500 text-xs mt-1">{errors.email.message}</p>}
              </div>
              <button
                type="submit"
                className="w-full rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-emerald-700"
              >
                Send Reset Link
              </button>
            </form>
          )}

          <div className="mt-6 text-center">
            <Link href={`/${locale}/login`} className="text-sm text-emerald-600 hover:text-emerald-700">
              ← Back to login
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
