import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { useAuth } from "@clerk/react-router";
import Button from "~/components/ui/Button";
import { Card, CardContent } from "~/components/ui/Card";
import { LEGAL_CONTACT_EMAIL } from "~/components/LegalPage";
import { useAuthenticatedFetch } from "~/lib/authFetch";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Get access");
}

type PricingCourse = {
  course_id: number;
  course_number: string;
  name: string;
  story_count: number;
  has_access: boolean;
  expires_at?: string;
};

type PricingResponse = {
  amount_cents: number;
  currency: string;
  name: string;
  course?: PricingCourse | null;
  payable_courses: PricingCourse[];
  payments_enabled?: boolean;
};

function formatMoney(cents: number, currency: string) {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency.toUpperCase(),
  }).format(cents / 100);
}

export default function PricingPage() {
  const [params] = useSearchParams();
  const courseParam = params.get("course");
  const { isSignedIn } = useAuth();
  const fetchAuth = useAuthenticatedFetch();
  const [data, setData] = useState<PricingResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [checkingOut, setCheckingOut] = useState(false);

  useEffect(() => {
    if (!isSignedIn) {
      setLoading(false);
      return;
    }
    const q = courseParam
      ? `?course_id=${encodeURIComponent(courseParam)}`
      : "";
    void (async () => {
      try {
        const res = await fetchAuth(`/api/pricing${q}`);
        if (!res.ok) {
          throw new Error("Could not load pricing");
        }
        setData((await res.json()) as PricingResponse);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Could not load pricing");
      } finally {
        setLoading(false);
      }
    })();
  }, [courseParam, fetchAuth, isSignedIn]);

  const selected = useMemo(() => {
    if (!data) return null;
    if (data.course) return data.course;
    if (data.payable_courses.length === 1) return data.payable_courses[0];
    return null;
  }, [data]);

  const startCheckout = async (courseId: number) => {
    setCheckingOut(true);
    setError(null);
    try {
      const res = await fetchAuth("/api/checkout", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ course_id: courseId }),
      });
      const body = await res.json();
      if (body.already_active) {
        window.location.href = "/";
        return;
      }
      if (!res.ok || !body.url) {
        throw new Error("Could not start checkout");
      }
      window.location.href = body.url;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not start checkout");
      setCheckingOut(false);
    }
  };

  return (
    <div className="w-full max-w-lg mx-auto px-4 py-10">
      <h1 className="text-3xl font-bold tracking-tight text-slate-900 mb-6">
        Get access
      </h1>
      {!isSignedIn ? (
        <p className="text-slate-600">Sign in to continue.</p>
      ) : loading ? (
        <p className="text-slate-600">Loading...</p>
      ) : (
        <Card>
          <CardContent className="p-6 flex flex-col gap-4">
            {error ? (
              <p className="text-rose-700" role="alert">
                {error}
              </p>
            ) : null}
            {data && data.amount_cents > 0 ? (
              <p className="text-2xl font-semibold text-slate-900">
                {formatMoney(data.amount_cents, data.currency)}
                <span className="ml-2 text-base font-normal text-slate-500">
                  / year
                </span>
              </p>
            ) : null}
            {selected ? (
              <>
                <p className="text-slate-700">
                  {selected.name} ({selected.course_number}) —{" "}
                  {selected.story_count} stories
                </p>
                {data?.payments_enabled === false ? null : (
                  <Button
                    onClick={() => void startCheckout(selected.course_id)}
                    disabled={checkingOut}
                  >
                    {checkingOut ? "Redirecting..." : "Continue to payment"}
                  </Button>
                )}
              </>
            ) : data && data.payable_courses.length > 1 ? (
              <ul className="flex flex-col gap-3">
                {data.payable_courses.map((c) => (
                  <li key={c.course_id}>
                    {data?.payments_enabled === false ? (
                      <p className="text-slate-700">
                        {c.name} ({c.course_number}) — {c.story_count} stories
                      </p>
                    ) : (
                      <Button
                        variant="outline"
                        onClick={() => void startCheckout(c.course_id)}
                        disabled={checkingOut}
                      >
                        {c.name} ({c.course_number}) — {c.story_count} stories
                      </Button>
                    )}
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-slate-600">
                No payable courses.{" "}
                <Link to="/" className="underline">
                  Home
                </Link>
              </p>
            )}
            {data?.payments_enabled === false ? (
              <p className="text-slate-600">
                Payments are temporarily paused while we fix a problem with our
                payment system. Paid stories stay open in the meantime, so you
                can keep working. Questions? Email{" "}
                <a className="underline" href={`mailto:${LEGAL_CONTACT_EMAIL}`}>
                  {LEGAL_CONTACT_EMAIL}
                </a>
                .
              </p>
            ) : null}
            <p className="text-sm text-slate-500">
              If this presents a hardship, email{" "}
              <a className="underline" href={`mailto:${LEGAL_CONTACT_EMAIL}`}>
                {LEGAL_CONTACT_EMAIL}
              </a>
              .
            </p>
            <p className="text-sm text-slate-500">
              If you were not granted access despite making a payment, please do
              not resubmit payment or you will be charged twice. Reach out to{" "}
              <a className="underline" href={`mailto:${LEGAL_CONTACT_EMAIL}`}>
                {LEGAL_CONTACT_EMAIL}
              </a>{" "}
              instead.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
