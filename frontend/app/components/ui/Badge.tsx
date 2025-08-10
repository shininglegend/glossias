import React from "react";
import { cn } from "~/lib/cn";

export type BadgeProps = React.HTMLAttributes<HTMLSpanElement> & {
  variant?: "default" | "muted" | "success" | "warning" | "danger";
};

export default function Badge({
  className,
  variant = "default",
  ...props
}: BadgeProps) {
  const variants = {
    default: "bg-slate-100 text-slate-700",
    muted: "bg-slate-50 text-slate-500",
    success: "bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200",
    warning: "bg-amber-50 text-amber-700 ring-1 ring-amber-200",
    danger: "bg-rose-50 text-rose-700 ring-1 ring-rose-200",
  } as const;
  return (
    <span
      className={cn(
        "inline-flex items-center rounded px-2 py-0.5 text-xs",
        variants[variant],
        className
      )}
      {...props}
    />
  );
}
