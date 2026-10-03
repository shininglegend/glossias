import React from "react";
import Button from "~/components/ui/Button";
import { Card } from "~/components/ui/Card";
import { useAuthenticatedFetch } from "../../lib/authFetch";

interface PaymentsStatus {
  paused: boolean;
  reason?: string;
  detail?: string;
  since?: string;
  paywall_enabled: boolean;
}

const REASON_LABELS: Record<string, string> = {
  grant_failed: "A verified payment could not be recorded",
  webhook_rejected: "Stripe webhooks keep failing signature verification",
  gateway_down: "Stripe's API is unreachable or erroring",
  not_configured: "Stripe is not configured but the paywall is on",
  manual: "Paused by an operator",
};

/**
 * Super-admin view of the payments fail-open state. While payments are
 * paused, checkout is refused and the paywall is lifted. The resume button
 * clears the pause in the running server (and the persisted record) without
 * a restart; if the cause is still there, the next failure pauses again.
 */
export default function PaymentsStatusCard() {
  const authenticatedFetch = useAuthenticatedFetch();
  const [status, setStatus] = React.useState<PaymentsStatus | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [resuming, setResuming] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [success, setSuccess] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    try {
      const response = await authenticatedFetch("/api/admin/system/payments");
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      setStatus((await response.json()) as PaymentsStatus);
      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to load payments status",
      );
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  const handleResume = async () => {
    if (
      !confirm(
        "Resume payments? Checkout reopens and the paywall is enforced again. Only do this once the cause has been fixed.",
      )
    ) {
      return;
    }
    setResuming(true);
    setError(null);
    setSuccess(null);
    try {
      const response = await authenticatedFetch(
        "/api/admin/system/payments/resume",
        { method: "POST" },
      );
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      const body = (await response.json()) as {
        resumed: boolean;
        status: PaymentsStatus;
      };
      setStatus(body.status);
      setSuccess(
        body.resumed
          ? "Payments resumed. Checkout is open and the paywall is enforced."
          : "Payments were not paused.",
      );
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to resume payments",
      );
    } finally {
      setResuming(false);
    }
  };

  const paused = status?.paused === true;
  const paywallState = !status
    ? ""
    : !status.paywall_enabled
      ? "off (PAYWALL_ENABLED is not set)"
      : paused
        ? "lifted while payments are paused"
        : "enforced";

  return (
    <Card className="p-6">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h2 className="text-xl font-semibold mb-2">Payments</h2>
          {loading ? (
            <p className="text-slate-600 text-sm">Loading...</p>
          ) : status ? (
            <div className="text-sm text-slate-600 space-y-1">
              <p>
                <span className="font-medium text-slate-700">Payments:</span>{" "}
                {paused ? (
                  <span className="text-red-700 font-medium">paused</span>
                ) : (
                  <span className="text-green-700 font-medium">active</span>
                )}
              </p>
              <p>
                <span className="font-medium text-slate-700">Paywall:</span>{" "}
                {paywallState}
              </p>
              {paused ? (
                <>
                  <p>
                    <span className="font-medium text-slate-700">Reason:</span>{" "}
                    {(status.reason && REASON_LABELS[status.reason]) ||
                      status.reason}
                  </p>
                  {status.since ? (
                    <p>
                      <span className="font-medium text-slate-700">Since:</span>{" "}
                      {new Date(status.since).toLocaleString()}
                    </p>
                  ) : null}
                  {status.detail ? (
                    <p className="font-mono text-xs break-words text-slate-500">
                      {status.detail}
                    </p>
                  ) : null}
                </>
              ) : null}
            </div>
          ) : null}
          {error ? <p className="text-sm text-red-700 mt-2">{error}</p> : null}
          {success ? (
            <p className="text-sm text-green-700 mt-2">{success}</p>
          ) : null}
        </div>
        {paused ? (
          <Button
            onClick={() => void handleResume()}
            disabled={resuming}
            icon={
              <span className="material-icons text-sm">
                {resuming ? "hourglass_empty" : "play_arrow"}
              </span>
            }
          >
            {resuming ? "Resuming..." : "Resume payments"}
          </Button>
        ) : (
          <Button variant="outline" onClick={() => void load()}>
            Refresh
          </Button>
        )}
      </div>
    </Card>
  );
}
