import React from "react";
import { cn } from "~/lib/cn";

type Variant =
  | "primary"
  | "secondary"
  | "ghost"
  | "danger"
  | "warning"
  | "outline";

type Size = "sm" | "md" | "lg";

export type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant;
  size?: Size;
  icon?: React.ReactNode;
};

export default function Button({
  className,
  variant = "primary",
  size = "md",
  icon,
  children,
  ...props
}: ButtonProps) {
  const base =
    "inline-flex items-center justify-center rounded-md font-medium transition-colors focus:outline-none disabled:opacity-50 disabled:pointer-events-none";

  const sizes: Record<Size, string> = {
    sm: "h-8 px-3 text-xs 2",
    md: "h-9 px-3 text-sm gap-2",
    lg: "h-10 px-4 text-base gap-2",
  };

  const variants: Record<Variant, string> = {
    primary: "bg-blue-600 text-white hover:bg-blue-700 shadow-sm",
    secondary:
      "bg-slate-100 text-slate-800 hover:bg-slate-200 border border-slate-200",
    ghost: "bg-transparent hover:bg-slate-100 text-slate-700",
    danger: "bg-rose-600 text-white hover:bg-rose-700",
    warning:
      "bg-amber-50 text-amber-800 hover:bg-amber-100 ring-1 ring-amber-200",
    outline:
      "bg-white text-slate-700 border border-slate-300 hover:bg-slate-50",
  };

  return (
    <button
      className={cn(base, sizes[size], variants[variant], className)}
      {...props}
    >
      {icon}
      {children}
    </button>
  );
}
