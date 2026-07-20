"use client";

import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import * as Dialog from "@radix-ui/react-dialog";
import { X, AlertCircle } from "lucide-react";
import { useTranslations } from "next-intl";

// Empty-string inputs coerce to 0 under z.coerce.number(), which would let a
// blank required price/quantity field silently pass a min(0) check. Route
// blank/non-numeric input to undefined first so "required" actually fires.
const emptyToUndefined = (val: unknown) =>
  typeof val === "string" && val.trim() === "" ? undefined : val === undefined ? undefined : Number(val);

const requiredNumber = (requiredMsg: string, min: number, minMsg: string) =>
  z.preprocess(
    emptyToUndefined,
    z.number({ required_error: requiredMsg, invalid_type_error: requiredMsg }).min(min, minMsg)
  );

const requiredInt = (requiredMsg: string, min: number, minMsg: string) =>
  z.preprocess(
    emptyToUndefined,
    z
      .number({ required_error: requiredMsg, invalid_type_error: requiredMsg })
      .int("Quantity must be a whole number")
      .min(min, minMsg)
  );

const schema = z.object({
  name: z.string().trim().min(1, "Product name is required"),
  expiry_date: z.string().min(1, "Expiry date is required"),
  quantity: requiredInt("Quantity is required", 1, "Quantity must be at least 1"),
  purchase_price: requiredNumber("Purchase price is required", 0, "Purchase price must be 0 or more"),
  selling_price: requiredNumber("Selling price is required", 0, "Selling price must be 0 or more"),
  barcode: z.string().trim().optional(),
  batch_number: z.string().trim().optional(),
  category: z.string().trim().optional(),
  manufacturer: z.string().trim().optional(),
  supplier: z.string().trim().optional(),
  supplier_email: z.union([z.literal(""), z.string().trim().email("Enter a valid email")]).optional(),
});

export type AddProductFormData = z.infer<typeof schema>;

export interface AddProductPayload {
  name: string;
  expiry_date: string;
  quantity: number;
  purchase_price: number;
  selling_price: number;
  barcode?: string;
  batch_number?: string;
  category?: string;
  manufacturer?: string;
  supplier?: string;
  supplier_email?: string;
}

interface AddProductModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (payload: AddProductPayload) => void;
  isLoading?: boolean;
  serverError?: string | null;
}

const inputClass =
  "w-full rounded-lg border border-slate-200 px-3 py-2 text-sm focus:border-emerald-500 focus:outline-none focus:ring-1 focus:ring-emerald-500";
const labelClass = "block text-sm font-medium text-slate-700 mb-1";
const errorClass = "mt-1 text-xs text-red-600";

export function AddProductModal({ open, onClose, onSubmit, isLoading, serverError }: AddProductModalProps) {
  const t = useTranslations("inventory.addProduct");

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<AddProductFormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: "",
      expiry_date: "",
      quantity: undefined,
      purchase_price: undefined,
      selling_price: undefined,
      barcode: "",
      batch_number: "",
      category: "",
      manufacturer: "",
      supplier: "",
      supplier_email: "",
    },
  });

  useEffect(() => {
    if (open) {
      reset();
    }
  }, [open, reset]);

  const submit = (data: AddProductFormData) => {
    onSubmit({
      name: data.name,
      expiry_date: data.expiry_date,
      quantity: data.quantity,
      purchase_price: data.purchase_price,
      selling_price: data.selling_price,
      barcode: data.barcode || undefined,
      batch_number: data.batch_number || undefined,
      category: data.category || undefined,
      manufacturer: data.manufacturer || undefined,
      supplier: data.supplier || undefined,
      supplier_email: data.supplier_email || undefined,
    });
  };

  return (
    <Dialog.Root open={open} onOpenChange={(o) => !o && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/40 z-40" />
        <Dialog.Content
          className="fixed left-1/2 top-1/2 z-50 w-full max-w-lg -translate-x-1/2 -translate-y-1/2 rounded-xl bg-white shadow-xl p-6 focus:outline-none max-h-[90vh] overflow-y-auto"
          aria-describedby={undefined}
        >
          <div className="flex items-center justify-between mb-5">
            <Dialog.Title className="text-lg font-semibold text-slate-900">{t("title")}</Dialog.Title>
            <Dialog.Close asChild>
              <button
                className="rounded-md p-1 text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
                onClick={onClose}
              >
                <X className="w-4 h-4" />
              </button>
            </Dialog.Close>
          </div>

          {serverError && (
            <div className="mb-4 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2.5 text-sm text-red-700">
              <AlertCircle className="w-4 h-4 mt-0.5 flex-shrink-0" />
              <span>{serverError}</span>
            </div>
          )}

          <form onSubmit={handleSubmit(submit)} className="space-y-4">
            {/* Required fields */}
            <div>
              <label className={labelClass}>{t("name")} *</label>
              <input {...register("name")} className={inputClass} placeholder={t("namePlaceholder")} />
              {errors.name && <p className={errorClass}>{errors.name.message}</p>}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className={labelClass}>{t("expiryDate")} *</label>
                <input type="date" {...register("expiry_date")} className={inputClass} />
                {errors.expiry_date && <p className={errorClass}>{errors.expiry_date.message}</p>}
              </div>
              <div>
                <label className={labelClass}>{t("quantity")} *</label>
                <input
                  type="number"
                  min={1}
                  step={1}
                  {...register("quantity")}
                  className={inputClass}
                />
                {errors.quantity && <p className={errorClass}>{errors.quantity.message}</p>}
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className={labelClass}>{t("purchasePrice")} *</label>
                <input
                  type="number"
                  min={0}
                  step={0.01}
                  {...register("purchase_price")}
                  className={inputClass}
                />
                {errors.purchase_price && <p className={errorClass}>{errors.purchase_price.message}</p>}
              </div>
              <div>
                <label className={labelClass}>{t("sellingPrice")} *</label>
                <input
                  type="number"
                  min={0}
                  step={0.01}
                  {...register("selling_price")}
                  className={inputClass}
                />
                {errors.selling_price && <p className={errorClass}>{errors.selling_price.message}</p>}
              </div>
            </div>

            {/* Optional fields */}
            <div className="pt-2 border-t border-slate-100">
              <p className="text-xs font-medium text-slate-400 uppercase tracking-wide mb-3 mt-3">
                {t("optionalSection")}
              </p>

              <div className="grid grid-cols-2 gap-4 mb-4">
                <div>
                  <label className={labelClass}>{t("barcode")}</label>
                  <input {...register("barcode")} className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>{t("batchNumber")}</label>
                  <input {...register("batch_number")} className={inputClass} />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4 mb-4">
                <div>
                  <label className={labelClass}>{t("category")}</label>
                  <input {...register("category")} className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>{t("manufacturer")}</label>
                  <input {...register("manufacturer")} className={inputClass} />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className={labelClass}>{t("supplier")}</label>
                  <input {...register("supplier")} className={inputClass} />
                </div>
                <div>
                  <label className={labelClass}>{t("supplierEmail")}</label>
                  <input type="email" {...register("supplier_email")} className={inputClass} />
                  {errors.supplier_email && <p className={errorClass}>{errors.supplier_email.message}</p>}
                </div>
              </div>
            </div>

            <div className="flex gap-3 pt-2">
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
                {isLoading ? "…" : t("submit")}
              </button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
