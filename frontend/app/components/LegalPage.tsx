import type { ReactNode } from "react";
import { Footer } from "./Footer";
import { cn } from "~/lib/cn";

export const LEGAL_EFFECTIVE_DATE = "September 7, 2026";
export const LEGAL_CONTACT_EMAIL = "help@glossias.org";

export function LegalPage({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  return (
    <div className="-my-4 mx-[calc(50%-50vw)] flex-1 flex flex-col">
      <article className="flex-1 w-full max-w-3xl mx-auto px-4 py-12">
        <h1 className="text-3xl md:text-4xl font-bold tracking-tight text-slate-900">
          {title}
        </h1>
        <p className="mt-2 text-sm text-slate-500">
          Effective as of {LEGAL_EFFECTIVE_DATE}
        </p>
        <div
          className={cn(
            "mt-8 text-slate-700 leading-relaxed",
            "[&_h2]:text-xl [&_h2]:font-semibold [&_h2]:text-slate-900 [&_h2]:mt-10 [&_h2]:mb-3 [&_h2]:scroll-mt-24",
            "[&_h3]:text-lg [&_h3]:font-semibold [&_h3]:text-slate-800 [&_h3]:mt-6 [&_h3]:mb-2",
            "[&_p]:mb-4",
            "[&_ul]:mb-4 [&_ul]:list-disc [&_ul]:pl-6 [&_ul]:space-y-1",
            "[&_ol]:mb-4 [&_ol]:list-decimal [&_ol]:pl-6 [&_ol]:space-y-2",
            "[&_li]:leading-relaxed",
            "[&_a]:text-primary-600 [&_a]:underline [&_a]:underline-offset-2 hover:[&_a]:text-primary-700",
            "[&_table]:w-full [&_table]:text-sm [&_th]:text-left [&_th]:font-semibold [&_th]:p-2 [&_th]:border-b [&_th]:border-slate-200 [&_th]:align-top",
            "[&_td]:p-2 [&_td]:align-top [&_td]:border-b [&_td]:border-slate-100",
            "[&_strong]:text-slate-900",
          )}
        >
          {children}
        </div>
      </article>
      <Footer />
    </div>
  );
}
