import React from "react";
import Button from "./Button";
import Modal from "./Modal";

type ConfirmDialogVariant =
  "delete" | "clear" | "danger" | "warning" | "default";

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
    buttonVariant: "primary" as const,
  },
  default: {
    title: "Confirm",
    message: "Are you sure?",
    confirmText: "Confirm",
    buttonVariant: "primary" as const,
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
  const confirmRef = React.useRef<HTMLButtonElement>(null);

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title ?? config.title}
      description={message ?? config.message}
      closeDisabled={loading}
      initialFocusRef={confirmRef}
    >
      <div className="mt-6 flex gap-3 justify-end">
        <Button variant="outline" onClick={onClose} disabled={loading}>
          {cancelText}
        </Button>
        <Button
          ref={confirmRef}
          variant={config.buttonVariant}
          onClick={onConfirm}
          disabled={loading}
        >
          {loading ? "Processing..." : (confirmText ?? config.confirmText)}
        </Button>
      </div>
    </Modal>
  );
}
