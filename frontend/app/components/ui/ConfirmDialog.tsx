import React from "react";
import Button from "./Button";

type ConfirmDialogVariant =
  | "delete"
  | "clear"
  | "danger"
  | "warning"
  | "default";

interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title?: string;
  message?: string;
  confirmText?: string;
  cancelText?: string;
  variant?: ConfirmDialogVariant;
  loading?: boolean;
}

const variantConfig = {
  delete: {
    title: "Delete Item",
    message:
      "This will permanently delete this item. This action cannot be undone.",
    confirmText: "Delete",
    buttonVariant: "danger" as const,
  },
  clear: {
    title: "Clear Data",
    message:
      "This will permanently clear all data. This action cannot be undone.",
    confirmText: "Clear All",
    buttonVariant: "danger" as const,
  },
  danger: {
    title: "Confirm Action",
    message: "This action cannot be undone. Are you sure you want to continue?",
    confirmText: "Continue",
    buttonVariant: "danger" as const,
  },
  warning: {
    title: "Confirm Action",
    message: "Are you sure you want to proceed?",
    confirmText: "Proceed",
    buttonVariant: "default" as const,
  },
  default: {
    title: "Confirm",
    message: "Are you sure?",
    confirmText: "Confirm",
    buttonVariant: "default" as const,
  },
};

export default function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText,
  cancelText = "Cancel",
  variant = "default",
  loading = false,
}: ConfirmDialogProps) {
  const config = variantConfig[variant];
  const finalTitle = title ?? config.title;
  const finalMessage = message ?? config.message;
  const finalConfirmText = confirmText ?? config.confirmText;
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 shadow-lg">
        <h2 className="text-lg font-semibold mb-2">{finalTitle}</h2>
        <p className="text-gray-600 mb-6">{finalMessage}</p>
        <div className="flex gap-3 justify-end">
          <Button variant="outline" onClick={onClose} disabled={loading}>
            {cancelText}
          </Button>
          <Button
            variant={config.buttonVariant}
            onClick={onConfirm}
            disabled={loading}
          >
            {loading ? "Processing..." : finalConfirmText}
          </Button>
        </div>
      </div>
    </div>
  );
}
