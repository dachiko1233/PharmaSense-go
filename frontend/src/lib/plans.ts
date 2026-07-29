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
  pro: "https://test.checkout.dodopayments.com/buy/pdt_0NkFSz0tF8O0pdbqjmrQu?quantity=1",
  chain:
    "https://test.checkout.dodopayments.com/buy/pdt_0NkFU3fXCPK4jli7YKqTW?quantity=1&redirect_url=https://www.pharmasense.co",
};
