import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router";
import { LEGAL_CONTACT_EMAIL } from "~/components/LegalPage";
import { useUserContext } from "~/contexts/UserContext";
import { useAuthenticatedFetch } from "~/lib/authFetch";
import { pageMeta } from "~/lib/pageTitle";

export function meta() {
  return pageMeta("Payment complete");
}

export default function CheckoutSuccessPage() {
  const [params] = useSearchParams();
  const sessionId = params.get("session_id");
  const { syncUser } = useUserContext();
  const fetchAuth = useAuthenticatedFetch();
  const [ready, setReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!sessionId) {
      setError("Missing payment session.");
      return;
    }
    let stopped = false;
    let attempts = 0;
    let intervalId = 0;
    const stop = () => {
      stopped = true;
      window.clearInterval(intervalId);
    };
    const fail = (message: string) => {
      setError(message);
      stop();
    };
    const tick = async () => {
      if (stopped) return;
      attempts += 1;
      try {
        const res = await fetchAuth("/api/checkout/confirm", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ session_id: sessionId }),
        });
        const body = (await res.json().catch(() => null)) as {
          ok?: boolean;
          pending?: boolean;
        } | null;
        if (stopped) return;
        if (res.ok && body?.ok) {
          stop();
          await syncUser({ force: true });
          setReady(true);
          return;
        }
        if (res.ok && body?.pending) {
          if (attempts >= 8) {
            fail("Payment is still processing. Refresh this page in a moment.");
          }
          return;
        }
        // 400/403 mean this session can never confirm for this user.
        // Anything else (502 from Stripe, 500 from the grant) may clear
        // on a later tick, and the webhook grants independently.
        if (res.status === 400 || res.status === 403 || attempts >= 8) {
          fail("Could not confirm access.");
        }
      } catch {
        if (!stopped && attempts >= 8) {
          fail("Could not confirm access.");
        }
      }
    };
    void tick();
    intervalId = window.setInterval(() => {
      void tick();
    }, 1500);
    return stop;
  }, [sessionId, fetchAuth, syncUser]);

  return (
    <div className="w-full max-w-lg mx-auto px-4 py-16 text-center">
      <h1 className="text-3xl font-bold tracking-tight text-slate-900 mb-4">
        Payment received
      </h1>
      <p className="text-slate-600 mb-6">
        {error
          ? error
          : ready
            ? "Your access is ready."
            : "Confirming your access..."}
      </p>
      <p className="text-sm text-slate-500 mb-6">
        If you were not granted access despite making a payment, please do not
        resubmit payment or you will be charged twice. Reach out to{" "}
        <a className="underline" href={`mailto:${LEGAL_CONTACT_EMAIL}`}>
          {LEGAL_CONTACT_EMAIL}
        </a>{" "}
        instead.
      </p>
      <Link to="/" className="text-primary-600 underline">
        Go to stories
      </Link>
    </div>
  );
}
