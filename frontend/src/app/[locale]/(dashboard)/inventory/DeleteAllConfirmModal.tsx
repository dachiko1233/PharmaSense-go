"use client";

import { useEffect, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { X, AlertTriangle } from "lucide-react";
import { useTranslations } from "next-intl";

const CONFIRM_WORD = "DELETE";

interface DeleteAllConfirmModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  count: number;
  isLoading?: boolean;
}

export function DeleteAllConfirmModal({ open, onClose, onConfirm, count, isLoading }: DeleteAllConfirmModalProps) {
  const t = useTranslations("inventory.deleteAllModal");
  const [typedText, setTypedText] = useState("");

  useEffect(() => {
    if (open) {
      setTypedText("");
    }
  }, [open]);

  const isMatch = typedText === CONFIRM_WORD;

  return (
    <Dialog.Root open={open} onOpenChange={(o) => !o && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/40 z-40" />
        <Dialog.Content
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl bg-white shadow-xl p-6 focus:outline-none"
          aria-describedby={undefined}
        >
          <div className="flex items-center justify-between mb-4">
            <Dialog.Title className="text-lg font-semibold text-slate-900 flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-red-600" />
              {t("title")}
            </Dialog.Title>
            <Dialog.Close asChild>
              <button
                className="rounded-md p-1 text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
                onClick={onClose}
              >
                <X className="w-4 h-4" />
              </button>
            </Dialog.Close>
          </div>

          <p className="text-sm text-slate-700 mb-2">{t("description", { count })}</p>
          <p className="text-sm font-semibold text-red-600 mb-4">{t("irreversible")}</p>

          <label className="block text-sm font-medium text-slate-700 mb-1">
            {t("typeToConfirm", { word: CONFIRM_WORD })}
          </label>
          <input
            autoFocus
            value={typedText}
            onChange={(e) => setTypedText(e.target.value)}
            placeholder={CONFIRM_WORD}
            className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono focus:border-red-500 focus:outline-none focus:ring-1 focus:ring-red-500"
            autoComplete="off"
            spellCheck={false}
          />

          <div className="flex gap-3 pt-5">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 rounded-lg border border-slate-200 px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-colors"
            >
              {t("cancel")}
            </button>
            <button
              type="button"
              onClick={onConfirm}
              disabled={!isMatch || isLoading}
              className="flex-1 rounded-lg bg-red-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isLoading ? "…" : t("confirm")}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
