import { useAuth } from "@clerk/react-router";
import { useCallback, useRef, useEffect } from "react";

class TimeTracker {
  private currentTrackingId: number | null = null;
  private cleanup: (() => void) | null = null;

  async startTracking(getToken: () => Promise<string | null>, route?: string) {
    const token = await getToken();
    const headers = new Headers();
    if (token) headers.set("Authorization", `Bearer ${token}`);
    headers.set("Content-Type", "application/json");

    const currentRoute = route || window.location.pathname;
    const storyId = currentRoute.includes("/stories/")
      ? parseInt(currentRoute.split("/stories/")[1]) || null
      : null;

    try {
      const response = await fetch("/api/time-tracking/start", {
        method: "POST",
        headers,
        body: JSON.stringify({
          route: currentRoute,
          story_id: storyId,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        this.currentTrackingId = data.tracking_id;
        this.setupPageLeaveTracking(getToken);
        return data.tracking_id;
      }
    } catch (error) {
      console.error("Failed to start time tracking:", error);
    }
    return null;
  }

  async endTracking(
    getToken: () => Promise<string | null>,
    trackingId?: number,
    useBeacon = false,
  ) {
    const id = trackingId || this.currentTrackingId;
    if (!id) return;

    if (useBeacon && navigator.sendBeacon) {
      const token = await getToken();
      const formData = new FormData();
      formData.append("tracking_id", id.toString());
      if (token) formData.append("authorization", `Bearer ${token}`);
      navigator.sendBeacon("/api/time-tracking/end", formData);
    } else {
      const token = await getToken();
      const headers = new Headers();
      if (token) headers.set("Authorization", `Bearer ${token}`);
      headers.set("Content-Type", "application/json");

      fetch("/api/time-tracking/end", {
        method: "POST",
        headers,
        body: JSON.stringify({ tracking_id: id }),
      }).catch(() => {}); // Fire and forget
    }

    if (id === this.currentTrackingId) {
      this.currentTrackingId = null;
      this.cleanupPageLeaveTracking();
    }
  }

  private setupPageLeaveTracking(getToken: () => Promise<string | null>) {
    if (!this.currentTrackingId) return;

    const trackingId = this.currentTrackingId;
    const handlePageLeave = () => this.endTracking(getToken, trackingId, true);

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
  const { getToken } = useAuth();
  const hasStartedRef = useRef(false);
  const pendingStartRef = useRef<Promise<number | null> | null>(null);

  const startTracking = useCallback(
    async (route?: string) => {
      if (hasStartedRef.current) return globalTracker.getCurrentTrackingId();
      if (pendingStartRef.current) return await pendingStartRef.current;

      hasStartedRef.current = true;
      const startPromise = globalTracker.startTracking(getToken, route);
      pendingStartRef.current = startPromise;

      const result = await startPromise;
      pendingStartRef.current = null;
      return result;
    },
    [getToken],
  );

  const endTracking = useCallback(
    async (trackingId?: number) => {
      hasStartedRef.current = false;
      pendingStartRef.current = null;
      return await globalTracker.endTracking(getToken, trackingId);
    },
    [getToken],
  );

  useEffect(() => {
    return () => {
      hasStartedRef.current = false;
      pendingStartRef.current = null;
      globalTracker.endTracking(getToken);
    };
  }, [getToken]);

  return { startTracking, endTracking };
}
