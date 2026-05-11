"use client";

import { useSearchParams, useRouter, useParams } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";

export default function MockStripePage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const { locale } = useParams();
  const plan = searchParams.get("plan") ?? "pro";
  const [simulating, setSimulating] = useState(false);

  const simulate = () => {
    setSimulating(true);
    setTimeout(() => {
      toast.success(`Simulated upgrade to ${plan} plan (dev mode)`);
      router.push(`/${locale}/billing?success=true`);
    }, 1500);
  };

  return (
    <div className="min-h-screen bg-slate-900 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl shadow-2xl max-w-md w-full p-8">
        <div className="bg-yellow-100 border border-yellow-300 rounded-lg px-4 py-2.5 text-yellow-800 text-sm font-medium text-center mb-6">
          🧪 DEV MODE — Mock Stripe Checkout
        </div>

        <h1 className="text-2xl font-bold text-slate-900 mb-2">Stripe Checkout</h1>
        <p className="text-slate-500 text-sm mb-6">
          This is a simulated Stripe checkout for local development.
          No real payment will be made.
        </p>

        <div className="bg-slate-50 rounded-lg p-4 mb-6 text-sm">
          <div className="flex justify-between mb-2">
            <span className="text-slate-600">Plan</span>
            <span className="font-semibold text-slate-900 capitalize">{plan}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-slate-600">Amount</span>
            <span className="font-semibold text-slate-900">
              {plan === "chain" ? "€99.00/mo" : plan === "pro" ? "€29.00/mo" : "Free"}
            </span>
          </div>
        </div>

        <button
          onClick={simulate}
          disabled={simulating}
          className="w-full rounded-lg bg-emerald-600 px-4 py-3 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
        >
          {simulating ? "Processing..." : "Simulate Successful Payment"}
        </button>

        <button
          onClick={() => router.push(`/${locale}/billing?canceled=true`)}
          className="w-full mt-3 rounded-lg border border-slate-200 px-4 py-2.5 text-sm text-slate-600 hover:bg-slate-50"
        >
          Cancel
        </button>
      </div>
    </div>
  );
}
