export type PlanKey = "free" | "pro" | "chain";

export const PLAN_ORDER: PlanKey[] = ["free", "pro", "chain"];

export const PLAN_RING: Record<PlanKey, string> = {
  free: "ring-slate-200",
  pro: "ring-emerald-500",
  chain: "ring-purple-500",
};

export const PLAN_BUTTON: Record<PlanKey, string> = {
  free: "",
  pro: "bg-emerald-600 hover:bg-emerald-700",
  chain: "bg-purple-600 hover:bg-purple-700",
};

export const PLAN_PAYMENT_LINK: Partial<Record<PlanKey, string>> = {
  pro: "https://buy.stripe.com/test_00w9ASeCbetbfbFdajd7q01",
  chain: "https://buy.stripe.com/test_3cIbJ08dNdp74x15HRd7q02",
};
