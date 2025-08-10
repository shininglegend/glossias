import React from "react";
import { cn } from "~/lib/cn";

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

export default function Input({ className, ...props }: InputProps) {
  return (
    <input
      className={cn(
        "w-full rounded-md border border-slate-300 bg-white py-2 px-3 text-sm shadow-sm outline-none placeholder:text-slate-400 focus:border-blue-500 focus:ring-2 focus:ring-blue-200",
        className
      )}
      {...props}
    />
  );
}
