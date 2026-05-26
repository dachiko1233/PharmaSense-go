"use client";

import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import * as Dialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { useTranslations } from "next-intl";
import { formatCurrency } from "@/lib/utils";

interface ReturnSupplierModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (quantity: number, notes?: string) => void;
  productName: string;
  currentQuantity: number;
  purchasePrice: number;
  locale: string;
  isLoading?: boolean;
}

export function ReturnSupplierModal({
  open,
  onClose,
  onConfirm,
  productName,
  currentQuantity,
  purchasePrice,
  locale,
  isLoading,
}: ReturnSupplierModalProps) {
  const t = useTranslations("alerts.returnModal");

  const schema = useMemo(
    () =>
      z.object({
        quantity: z.coerce
          .number({ invalid_type_error: "Enter a number" })
          .int()
          .min(1, "Minimum 1 unit")
          .max(currentQuantity, `Maximum ${currentQuantity} units`),
        notes: z.string().optional(),
      }),
    [currentQuantity]
  );

  type FormData = z.infer<typeof schema>;

  const {
    register,
    handleSubmit,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { quantity: currentQuantity, notes: "" },
  });

  useEffect(() => {
    if (open) {
      reset({ quantity: currentQuantity, notes: "" });
    }
  }, [open, currentQuantity, reset]);

  const quantityValue = watch("quantity");
  const refundEstimate = (Number(quantityValue) || 0) * purchasePrice;

  const onSubmit = (data: FormData) => {
    onConfirm(data.quantity, data.notes || undefined);
  };

  return (
    <Dialog.Root open={open} onOpenChange={(o) => !o && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/40 z-40" />
        <Dialog.Content
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-xl bg-white shadow-xl p-6 focus:outline-none"
          aria-describedby={undefined}
        >
          <div className="flex items-center justify-between mb-5">
            <Dialog.Title className="text-lg font-semibold text-slate-900">
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

          {/* Product info */}
          <div className="mb-4 rounded-lg bg-slate-50 px-4 py-3 text-sm">
            <div className="font-medium text-slate-900">{productName}</div>
            <div className="text-slate-500 mt-0.5">{currentQuantity} units in stock</div>
          </div>

          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            {/* Quantity input */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                {t("quantityLabel")}
              </label>
              <input
                type="number"
                min={1}
                max={currentQuantity}
                {...register("quantity", { valueAsNumber: true })}
                className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-emerald-500 focus:outline-none focus:ring-1 focus:ring-emerald-500"
              />
              {errors.quantity && (
                <p className="mt-1 text-xs text-red-600">{errors.quantity.message}</p>
              )}
              <p className="mt-1 text-xs text-slate-500">{t("quantityHelp")}</p>
            </div>

            {/* Estimated refund */}
            {refundEstimate > 0 && (
              <div className="rounded-lg bg-emerald-50 border border-emerald-200 px-4 py-2.5 text-sm flex items-center justify-between">
                <span className="text-emerald-700">Estimated refund</span>
                <span className="text-emerald-800 font-bold">
                  {formatCurrency(refundEstimate, "EUR", locale)}
                </span>
              </div>
            )}

            {/* Notes */}
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                {t("notesLabel")}
              </label>
              <textarea
                {...register("notes")}
                rows={3}
                className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-emerald-500 focus:outline-none focus:ring-1 focus:ring-emerald-500 resize-none"
              />
            </div>

            {/* Actions */}
            <div className="flex gap-3 pt-1">
              <button
                type="button"
                onClick={onClose}
                className="flex-1 rounded-lg border border-slate-200 px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 transition-colors"
              >
                {t("cancel")}
              </button>
              <button
                type="submit"
                disabled={isLoading}
                className="flex-1 rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50 transition-colors"
              >
                {isLoading ? "…" : t("confirm")}
              </button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
