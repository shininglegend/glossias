import { useCallback, useRef, useEffect } from "react";
import { useAuthenticatedFetch } from "./authFetch";

class TimeTracker {
  private currentTrackingId: number | null = null;
  private cleanup: (() => void) | null = null;

  async startTracking(
    authenticatedFetch: (
      input: RequestInfo,
      init?: RequestInit,
    ) => Promise<Response>,
    route?: string,
  ) {
    const currentRoute = route || window.location.pathname;
    const storyId = currentRoute.includes("/stories/")
      ? parseInt(currentRoute.split("/stories/")[1]) || null
      : null;

    try {
      const response = await authenticatedFetch("/api/time-tracking/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          route: currentRoute,
          story_id: storyId,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        this.currentTrackingId = data.tracking_id;
        this.setupPageLeaveTracking(authenticatedFetch);
        return data.tracking_id;
      }
    } catch (error) {
      console.error("Failed to start time tracking:", error);
    }
    return null;
  }

  async endTracking(
    authenticatedFetch: (
      input: RequestInfo,
      init?: RequestInit,
    ) => Promise<Response>,
    trackingId?: number,
    useBeacon = false,
  ) {
    const id = trackingId || this.currentTrackingId;
    if (!id) return;

    if (useBeacon && navigator.sendBeacon) {
      const formData = new FormData();
      formData.append("tracking_id", id.toString());
      navigator.sendBeacon("/api/time-tracking/end", formData);
    } else {
      fetch("/api/time-tracking/end", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tracking_id: id }),
      }).catch(() => {}); // Fire and forget
    }

    if (id === this.currentTrackingId) {
      this.currentTrackingId = null;
      this.cleanupPageLeaveTracking();
    }
  }

  private setupPageLeaveTracking(
    authenticatedFetch: (
      input: RequestInfo,
      init?: RequestInit,
    ) => Promise<Response>,
  ) {
    if (!this.currentTrackingId) return;

    const trackingId = this.currentTrackingId;
    const handlePageLeave = () =>
      this.endTracking(authenticatedFetch, trackingId, true);

    window.addEventListener("beforeunload", handlePageLeave);
    window.addEventListener("pagehide", handlePageLeave);
    document.addEventListener("visibilitychange", () => {
      if (document.visibilityState === "hidden") {
        handlePageLeave();
      }
    });

    this.cleanup = () => {
      window.removeEventListener("beforeunload", handlePageLeave);
      window.removeEventListener("pagehide", handlePageLeave);
      document.removeEventListener("visibilitychange", handlePageLeave);
    };
  }

  private cleanupPageLeaveTracking() {
    if (this.cleanup) {
      this.cleanup();
      this.cleanup = null;
    }
  }

  getCurrentTrackingId() {
    return this.currentTrackingId;
  }
}

const globalTracker = new TimeTracker();

export function useTimeTracking() {
  const authenticatedFetch = useAuthenticatedFetch();
  const hasStartedRef = useRef(false);
  const pendingStartRef = useRef<Promise<number | null> | null>(null);

  const startTracking = useCallback(
    async (route?: string) => {
      if (hasStartedRef.current) return globalTracker.getCurrentTrackingId();
      if (pendingStartRef.current) return await pendingStartRef.current;

      hasStartedRef.current = true;
      const startPromise = globalTracker.startTracking(
        authenticatedFetch,
        route,
      );
      pendingStartRef.current = startPromise;

      const result = await startPromise;
      pendingStartRef.current = null;
      return result;
    },
    [authenticatedFetch],
  );

  const endTracking = useCallback(
    async (trackingId?: number) => {
      hasStartedRef.current = false;
      pendingStartRef.current = null;
      return await globalTracker.endTracking(authenticatedFetch, trackingId);
    },
    [authenticatedFetch],
  );

  useEffect(() => {
    return () => {
      hasStartedRef.current = false;
      pendingStartRef.current = null;
      globalTracker.endTracking(authenticatedFetch);
    };
  }, [authenticatedFetch]);

  return { startTracking, endTracking };
}
