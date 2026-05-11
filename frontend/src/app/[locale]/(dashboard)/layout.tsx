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
  CreditCard, Network, Settings, LogOut, ChevronDown,
  Menu, X, Globe
} from "lucide-react";
import { toast } from "sonner";

interface PharmacyOption {
  id: string;
  name: string;
  plan: string;
  role: string;
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const t = useTranslations("nav");
  const tAuth = useTranslations("auth");
  const router = useRouter();
  const { locale } = useParams();
  const pathname = usePathname();
  const { token, user, pharmacy, role, isAuthenticated, setAuth, setPharmacy, logout, hydrate } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [pharmacies, setPharmacies] = useState<PharmacyOption[]>([]);
  const [switcherOpen, setSwitcherOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  useEffect(() => {
    hydrate();
  }, []);

  useEffect(() => {
    if (!token && !isAuthenticated) {
      router.push(`/${locale}/login`);
    }
  }, [token, isAuthenticated]);

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
    } catch (err: any) {
      toast.error("Failed to switch pharmacy");
    }
  };

  const handleLogout = () => {
    logout();
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
    { href: `/${locale}/billing`, label: t("billing"), icon: CreditCard },
    ...(role === "chain_admin" ? [{ href: `/${locale}/chain`, label: t("chain"), icon: Network }] : []),
    { href: `/${locale}/settings`, label: t("settings"), icon: Settings },
  ];

  if (!isAuthenticated && !token) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin w-8 h-8 border-4 border-emerald-600 border-t-transparent rounded-full" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 flex">
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
        "lg:static lg:translate-x-0"
      )}>
        {/* Logo */}
        <div className="px-6 py-5 border-b border-slate-200 flex items-center justify-between">
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

        {/* Navigation */}
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

        {/* Logout */}
        <div className="px-3 py-4 border-t border-slate-200">
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-slate-600 hover:bg-red-50 hover:text-red-600 w-full transition-colors"
          >
            <LogOut className="w-4 h-4" />
            {t("logout")}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Top header */}
        <header className="bg-white border-b border-slate-200 px-4 lg:px-6 py-3 flex items-center gap-4">
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

          {/* Plan badge */}
          {pharmacy?.plan && (
            <span className={cn("text-xs px-2.5 py-1 rounded-full font-semibold hidden sm:inline-flex", planBadgeClass(pharmacy.plan))}>
              {pharmacy.plan.toUpperCase()}
            </span>
          )}

          {/* User menu */}
          <div className="relative">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex items-center gap-2 hover:bg-slate-100 rounded-lg px-2 py-1.5"
            >
              <div className="w-8 h-8 bg-emerald-600 rounded-full flex items-center justify-center text-white text-sm font-semibold">
                {user?.full_name?.[0]?.toUpperCase() ?? "U"}
              </div>
              <div className="hidden sm:block text-left">
                <div className="text-sm font-medium text-slate-900 leading-tight">{user?.full_name ?? "User"}</div>
                <div className="text-xs text-slate-500 capitalize">{role}</div>
              </div>
            </button>
          </div>
        </header>

        {/* Email verification banner */}
        {user && !user.email_verified && (
          <div className="bg-amber-50 border-b border-amber-200 px-4 py-2 text-sm text-amber-800 flex items-center justify-center gap-2">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.068 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            {tAuth("verifyBanner")}
          </div>
        )}

        {/* Page content */}
        <main className="flex-1 overflow-auto">
          {children}
        </main>
      </div>
    </div>
  );
}
