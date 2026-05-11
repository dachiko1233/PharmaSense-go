"use client";

import { useState, useRef } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { apiUrl } from "@/lib/api/client";
import { Upload, FileText, Download, AlertCircle, CheckCircle } from "lucide-react";

interface ImportResult {
  imported: number;
  total: number;
  errors: string[];
}

export default function ImportPage() {
  const t = useTranslations("import");
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [dragging, setDragging] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleFile = (f: File) => {
    if (!f.name.endsWith(".csv")) {
      toast.error("Please select a CSV file");
      return;
    }
    setFile(f);
    setResult(null);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const f = e.dataTransfer.files[0];
    if (f) handleFile(f);
  };

  const handleImport = async () => {
    if (!file) return;
    setLoading(true);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const token = localStorage.getItem("token");
      const res = await fetch(apiUrl("/inventory/import"), {
        method: "POST",
        headers: { Authorization: `Bearer ${token ?? ""}` },
        body: formData,
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: "Import failed" }));
        throw new Error(err.error);
      }
      const data: ImportResult = await res.json();
      setResult(data);
      if (data.imported > 0) {
        toast.success(`${t("success")}: ${data.imported} batches imported`);
      }
    } catch (err: any) {
      toast.error(err.message ?? "Import failed");
    } finally {
      setLoading(false);
    }
  };

  const downloadTemplate = () => {
    const csv = [
      "product_name,batch_number,expiry_date,quantity,purchase_price,selling_price,received_date",
      "Paracetamol 500mg,BN-2024-001,2026-06-30,100,0.50,1.20,2024-01-15",
      "Ibuprofen 400mg,BN-2024-002,2026-03-31,50,0.80,1.80,2024-01-15",
    ].join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "pharmasense_template.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="p-6 lg:p-8 max-w-3xl">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">{t("title")}</h1>
        <p className="text-slate-500 text-sm mt-1">{t("subtitle")}</p>
      </div>

      {/* Format info */}
      <div className="bg-blue-50 border border-blue-200 rounded-xl p-4 mb-6 text-sm text-blue-800">
        <div className="flex gap-2">
          <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" />
          <div>{t("csvFormat")}</div>
        </div>
      </div>

      {/* Drop zone */}
      <div
        onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        onClick={() => inputRef.current?.click()}
        className={`border-2 border-dashed rounded-xl p-12 text-center cursor-pointer transition-colors mb-6 ${
          dragging ? "border-emerald-400 bg-emerald-50" : "border-slate-300 hover:border-slate-400 bg-white"
        }`}
      >
        <input
          ref={inputRef}
          type="file"
          accept=".csv"
          className="hidden"
          onChange={(e) => e.target.files?.[0] && handleFile(e.target.files[0])}
        />
        <Upload className="w-10 h-10 mx-auto mb-3 text-slate-400" />
        <p className="text-slate-600 font-medium">{t("dropzone")}</p>
        {file && (
          <div className="mt-3 flex items-center justify-center gap-2 text-sm text-emerald-700">
            <FileText className="w-4 h-4" />
            {file.name} ({(file.size / 1024).toFixed(1)} KB)
          </div>
        )}
      </div>

      <div className="flex gap-3">
        <button
          onClick={downloadTemplate}
          className="flex items-center gap-2 rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
        >
          <Download className="w-4 h-4" />
          {t("template")}
        </button>

        <button
          onClick={handleImport}
          disabled={!file || loading}
          className="flex items-center gap-2 rounded-lg bg-emerald-600 px-6 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? (
            <svg className="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
          ) : <Upload className="w-4 h-4" />}
          {loading ? t("importing") : t("import")}
        </button>
      </div>

      {/* Result */}
      {result && (
        <div className="mt-6 rounded-xl border border-slate-200 bg-white shadow-sm p-6">
          <div className="flex items-center gap-2 mb-4">
            <CheckCircle className="w-5 h-5 text-emerald-600" />
            <h3 className="font-semibold text-slate-900">{t("success")}</h3>
          </div>
          <div className="grid grid-cols-3 gap-4 mb-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-emerald-600">{result.imported}</div>
              <div className="text-xs text-slate-500">Imported</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-slate-900">{result.total}</div>
              <div className="text-xs text-slate-500">Total rows</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-red-500">{result.errors.length}</div>
              <div className="text-xs text-slate-500">{t("errors")}</div>
            </div>
          </div>
          {result.errors.length > 0 && (
            <div className="bg-red-50 rounded-lg p-3">
              <p className="text-xs font-medium text-red-700 mb-2">Errors:</p>
              <ul className="text-xs text-red-600 space-y-1 list-disc list-inside">
                {result.errors.map((e, i) => <li key={i}>{e}</li>)}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
