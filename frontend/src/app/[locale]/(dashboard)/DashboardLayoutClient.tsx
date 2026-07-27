"use client";

import { useEffect, useState } from "react";
import { useRouter, useParams, usePathname } from "next/navigation";
import Link from "next/link";
import { useTranslations } from "next-intl";
import { useAuth } from "@/lib/hooks/useAuth";
import { apiFetch } from "@/lib/api/client";
import { cn, planBadgeClass } from "@/lib/utils";
import {
  LayoutDashboard, Package, Bell, FileText, Upload,
  Network, Settings, CreditCard, LogOut, ChevronDown,
  Menu, X, Globe
} from "lucide-react";
import { toast } from "sonner";
import Footer from "@/components/layout/Footer";

interface PharmacyOption {
  id: string;
  name: string;
  plan: string;
  role: string;
}

export default function DashboardLayoutClient({ children }: { children: React.ReactNode }) {
  const t = useTranslations("nav");
  const tAuth = useTranslations("auth");
  const router = useRouter();
  const { locale } = useParams();
  const pathname = usePathname();
  const { token, user, pharmacy, role, isAuthenticated, setAuth, setPharmacy, logout, hydrate } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [pharmacies, setPharmacies] = useState<PharmacyOption[]>([]);
  const [switcherOpen, setSwitcherOpen] = useState(false);
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    hydrate();
    setHydrated(true);
  }, []);

  useEffect(() => {
    if (hydrated && !token && !isAuthenticated) {
      router.push(`/${locale}/login`);
    }
  }, [hydrated, token, isAuthenticated]);

  useEffect(() => {
    if (token) {
      apiFetch<PharmacyOption[]>("/pharmacies").then(setPharmacies).catch(() => {});
    }
  }, [token]);

  const handleSwitch = async (pharmacyId: string) => {
    try {
      const res = await apiFetch<{ token: string; pharmacy: any }>("/pharmacies/switch", {
        method: "POST",
        body: JSON.stringify({ pharmacy_id: pharmacyId }),
      });
      setPharmacy(res.pharmacy, res.token);
      setSwitcherOpen(false);
      toast.success(`Switched to ${res.pharmacy.name}`);
    } catch {
      toast.error("Failed to switch pharmacy");
    }
  };

  const handleLogout = async () => {
    try {
      await apiFetch("/auth/logout", { method: "POST" });
    } catch {
      // JWT is stateless; proceed regardless
    }
    logout();
    toast.success("Logged out successfully");
    router.push(`/${locale}/login`);
  };

  const toggleLocale = () => {
    const newLocale = locale === "en" ? "el" : "en";
    const newPath = pathname.replace(`/${locale}/`, `/${newLocale}/`);
    router.push(newPath);
  };

  const navItems = [
    { href: `/${locale}/dashboard`, label: t("dashboard"), icon: LayoutDashboard },
    { href: `/${locale}/inventory`, label: t("inventory"), icon: Package },
    { href: `/${locale}/alerts`, label: t("alerts"), icon: Bell },
    { href: `/${locale}/reports`, label: t("reports"), icon: FileText },
    { href: `/${locale}/import`, label: t("import"), icon: Upload },
    ...(role === "chain_admin" ? [{ href: `/${locale}/chain`, label: t("chain"), icon: Network }] : []),
    { href: `/${locale}/billing`, label: t("billing"), icon: CreditCard },
    { href: `/${locale}/settings`, label: t("settings"), icon: Settings },
  ];

  if (!hydrated || (!isAuthenticated && !token)) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin w-8 h-8 border-4 border-emerald-600 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="h-screen bg-slate-50 flex overflow-hidden">
      {/* Sidebar overlay on mobile */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-20 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside className={cn(
        "fixed inset-y-0 left-0 z-30 w-64 bg-white border-r border-slate-200 flex flex-col transition-transform duration-300",
        sidebarOpen ? "translate-x-0" : "-translate-x-full",
        "lg:relative lg:translate-x-0 lg:flex-shrink-0"
      )}>
        {/* Logo */}
        <div className="px-6 py-5 border-b border-slate-200 flex items-center justify-between flex-shrink-0">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-emerald-600 rounded-lg flex items-center justify-center">
              <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <span className="font-bold text-slate-900 text-lg">PharmaSense</span>
          </div>
          <button onClick={() => setSidebarOpen(false)} className="lg:hidden p-1 rounded text-slate-400 hover:text-slate-600">
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Navigation — scrollable if many items */}
        <nav className="flex-1 px-3 py-4 space-y-0.5 overflow-y-auto">
          {navItems.map(({ href, label, icon: Icon }) => {
            const isActive = pathname === href || pathname.startsWith(href + "/");
            return (
              <Link
                key={href}
                href={href}
                onClick={() => setSidebarOpen(false)}
                className={cn(
                  "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors",
                  isActive
                    ? "bg-emerald-50 text-emerald-700"
                    : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                )}
              >
                <Icon className="w-4 h-4 flex-shrink-0" />
                {label}
              </Link>
            );
          })}
        </nav>

        {/* Bottom: user info + logout — always visible, never scrolls away */}
        <div className="border-t border-slate-200 px-3 py-3 space-y-1 flex-shrink-0">
          {user && (
            <div className="px-3 py-2">
              <div className="text-sm font-medium text-slate-900 truncate">{user.full_name}</div>
              <div className="text-xs text-slate-500 truncate">{user.email}</div>
            </div>
          )}
          <button
            onClick={handleLogout}
            className="flex items-center gap-2 w-full rounded-md px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-50 transition-colors"
          >
            <LogOut className="w-4 h-4 flex-shrink-0" />
            {t("logout")}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Top header */}
        <header className="bg-white border-b border-slate-200 px-4 lg:px-6 py-3 flex items-center gap-4 flex-shrink-0">
          <button
            onClick={() => setSidebarOpen(true)}
            className="lg:hidden p-1.5 rounded text-slate-500 hover:text-slate-700 hover:bg-slate-100"
          >
            <Menu className="w-5 h-5" />
          </button>

          <div className="flex-1" />

          {/* Language switcher */}
          <button
            onClick={toggleLocale}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm text-slate-600 hover:bg-slate-100 border border-slate-200"
          >
            <Globe className="w-4 h-4" />
            <span className="font-medium">{locale === "en" ? "EL" : "EN"}</span>
          </button>

          {/* Pharmacy switcher */}
          {pharmacies.length > 1 && (
            <div className="relative">
              <button
                onClick={() => setSwitcherOpen(!switcherOpen)}
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg border border-slate-200 text-sm text-slate-700 hover:bg-slate-50 max-w-[200px]"
              >
                <span className="truncate">{pharmacy?.name ?? "Select pharmacy"}</span>
                <ChevronDown className="w-4 h-4 flex-shrink-0" />
              </button>
              {switcherOpen && (
                <div className="absolute right-0 top-full mt-1 w-64 bg-white border border-slate-200 rounded-xl shadow-lg z-50 overflow-hidden">
                  {pharmacies.map((p) => (
                    <button
                      key={p.id}
                      onClick={() => handleSwitch(p.id)}
                      className={cn(
                        "w-full px-4 py-3 text-left text-sm flex items-center justify-between hover:bg-slate-50",
                        pharmacy?.id === p.id ? "bg-emerald-50 text-emerald-700" : "text-slate-700"
                      )}
                    >
                      <span>{p.name}</span>
                      <span className={cn("text-xs px-2 py-0.5 rounded-full font-medium", planBadgeClass(p.plan))}>
                        {p.plan}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* User avatar */}
          <div className="w-8 h-8 bg-emerald-600 rounded-full flex items-center justify-center text-white text-sm font-semibold flex-shrink-0">
            {user?.full_name?.[0]?.toUpperCase() ?? "U"}
          </div>
        </header>

        {/* Email verification banner */}
        {user && !user.email_verified && (
          <div className="bg-amber-50 border-b border-amber-200 px-4 py-2 text-sm text-amber-800 flex items-center justify-center gap-2 flex-shrink-0">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.068 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            {tAuth("verifyBanner")}
          </div>
        )}

        {/* Page content */}
        <main className="flex-1 overflow-y-auto">
          {children}
          <Footer />
        </main>
      </div>
    </div>
  );
}
