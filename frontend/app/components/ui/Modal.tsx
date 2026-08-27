import React from "react";
import { cn } from "~/lib/cn";

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: React.ReactNode;
  description?: React.ReactNode;
  children?: React.ReactNode;
  className?: string;
  /** Blocks Escape and backdrop-click dismissal (e.g. while a request is in flight). */
  closeDisabled?: boolean;
  /** Close when the backdrop (outside the panel) is clicked. Defaults to true. */
  closeOnBackdropClick?: boolean;
  /** Element to focus when the dialog opens. Defaults to the first focusable child. */
  initialFocusRef?: React.RefObject<HTMLElement | null>;
}

/**
 * Accessible modal built on the native `<dialog>` element.
 *
 * `showModal()` gives us the top layer, an inert background (focus trap),
 * Escape handling (via the `cancel` event) and `::backdrop` for free.
 * On top of that we add `aria-labelledby` / `aria-describedby`, initial
 * focus control, and focus restoration to the previously focused element.
 *
 * Visibility is controlled by `isOpen`; the component renders nothing when
 * closed so callers never need to manage `open` themselves.
 */
export default function Modal({
  isOpen,
  onClose,
  title,
  description,
  children,
  className,
  closeDisabled = false,
  closeOnBackdropClick = true,
  initialFocusRef,
}: ModalProps) {
  const dialogRef = React.useRef<HTMLDialogElement>(null);
  const unmountingRef = React.useRef(false);
  const titleId = React.useId();
  const descriptionId = React.useId();

  React.useLayoutEffect(() => {
    if (!isOpen) return;
    const dialog = dialogRef.current;
    if (!dialog) return;

    const previouslyFocused =
      document.activeElement instanceof HTMLElement
        ? document.activeElement
        : null;

    unmountingRef.current = false;
    if (!dialog.open) dialog.showModal();
    initialFocusRef?.current?.focus();

    return () => {
      unmountingRef.current = true;
      if (dialog.open) dialog.close();
      previouslyFocused?.focus();
    };
  }, [isOpen, initialFocusRef]);

  if (!isOpen) return null;

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      aria-describedby={description !== undefined ? descriptionId : undefined}
      aria-modal="true"
      onCancel={(e) => {
        // Escape: keep React state as the single source of truth.
        e.preventDefault();
        if (!closeDisabled) onClose();
      }}
      onClose={() => {
        // The browser closed the dialog on its own (e.g. Chrome's close-watcher
        // ignoring preventDefault); sync React state unless we caused it.
        if (!unmountingRef.current) onClose();
      }}
      onClick={(e) => {
        // Clicks on ::backdrop are dispatched to the <dialog> element itself;
        // clicks inside the panel target its descendants.
        if (
          closeOnBackdropClick &&
          !closeDisabled &&
          e.target === e.currentTarget
        ) {
          onClose();
        }
      }}
      className={cn(
        "m-auto w-full max-w-[calc(100%-2rem)] rounded-lg border border-slate-200 bg-white p-0 text-slate-900 shadow-xl sm:max-w-md",
        "backdrop:bg-black/50",
        className,
      )}
    >
      <div className="p-6">
        <h2 id={titleId} className="text-lg font-semibold">
          {title}
        </h2>
        {description !== undefined && (
          <p id={descriptionId} className="mt-1 text-sm text-slate-600">
            {description}
          </p>
        )}
        {children}
      </div>
    </dialog>
  );
}
