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
