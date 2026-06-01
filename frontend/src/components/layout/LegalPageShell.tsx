"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useTranslations } from "next-intl";
import LocaleSwitcher from "./LocaleSwitcher";

export default function LegalPageShell({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  const tLegal = useTranslations("legal");
  const tFooter = useTranslations("footer");
  const { locale } = useParams();
  const lc = (locale as string) ?? "en";

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="border-b border-slate-200 bg-white">
        <nav className="mx-auto flex max-w-4xl items-center justify-between px-6 py-4">
          <Link href={`/${lc}`} className="flex items-center gap-2 font-semibold text-emerald-700">
            <span className="inline-flex h-7 w-7 items-center justify-center rounded-md bg-emerald-600">
              <svg className="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </span>
            PharmaSense
          </Link>
          <div className="flex items-center gap-4 text-sm text-slate-600">
            <Link href={`/${lc}/privacy`} className="hover:text-emerald-700">
              {tFooter("privacy")}
            </Link>
            <Link href={`/${lc}/terms`} className="hover:text-emerald-700">
              {tFooter("terms")}
            </Link>
            <Link href={`/${lc}/cookies`} className="hover:text-emerald-700">
              {tFooter("cookies")}
            </Link>
            <LocaleSwitcher />
          </div>
        </nav>
      </header>

      <main className="mx-auto max-w-4xl px-6 py-12">
        <h1 className="text-4xl font-bold text-emerald-700">{title}</h1>
        <p className="mt-2 text-sm text-slate-500">{tLegal("lastUpdated")}</p>
        <div className="mt-8 space-y-2">{children}</div>
      </main>

      <footer className="mt-12 border-t border-slate-200 bg-white py-6 text-center text-sm text-slate-500">
        <div className="mx-auto max-w-4xl px-6">
          <div>{tFooter("rights")}</div>
          <div className="mt-2 flex flex-wrap items-center justify-center gap-x-4 gap-y-1 text-slate-600">
            <Link href={`/${lc}/privacy`} className="hover:text-emerald-700">
              {tFooter("privacy")}
            </Link>
            <span aria-hidden>·</span>
            <Link href={`/${lc}/terms`} className="hover:text-emerald-700">
              {tFooter("terms")}
            </Link>
            <span aria-hidden>·</span>
            <Link href={`/${lc}/cookies`} className="hover:text-emerald-700">
              {tFooter("cookies")}
            </Link>
            <span aria-hidden>·</span>
            <a href="mailto:support@pharmasense.co" className="hover:text-emerald-700">
              {tFooter("contact")}
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}

export function LegalSection({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="mt-8">
      <h2 className="mb-4 text-2xl font-semibold text-slate-800">{title}</h2>
      <div className="space-y-3">{children}</div>
    </section>
  );
}

export function LegalParagraph({ children }: { children: React.ReactNode }) {
  return <p className="leading-relaxed text-slate-600">{children}</p>;
}

export function LegalList({ items }: { items: string[] }) {
  return (
    <ul className="list-disc space-y-2 pl-6 text-slate-600">
      {items.map((item, i) => (
        <li key={i} className="leading-relaxed">
          {item}
        </li>
      ))}
    </ul>
  );
}
