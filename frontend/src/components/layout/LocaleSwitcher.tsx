"use client";

import { useParams, usePathname, useRouter } from "next/navigation";
import { Globe } from "lucide-react";

export default function LocaleSwitcher() {
  const router = useRouter();
  const pathname = usePathname();
  const { locale } = useParams();
  const current = (locale as string) ?? "en";
  const next = current === "en" ? "el" : "en";

  const handleSwitch = () => {
    const newPath = pathname.replace(`/${current}`, `/${next}`);
    router.push(newPath);
  };

  return (
    <button
      onClick={handleSwitch}
      className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-50"
      aria-label="Switch language"
    >
      <Globe className="h-4 w-4" />
      {next.toUpperCase()}
    </button>
  );
}
