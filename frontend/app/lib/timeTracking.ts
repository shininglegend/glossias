import { useCallback, useRef, useEffect } from "react";
import { useAuthenticatedFetch } from "./authFetch";

const DEBUG = process.env.NODE_ENV === "development";
const log = (...args: any[]) => DEBUG && console.log("[TimeTracker]", ...args);

class TimeTracker {
  private startTime: number | null = null;
  private cleanup: (() => void) | null = null;
  private trackingId: string | null = null;

  async startTracking(
    route?: string,
    authenticatedFetch?: (
      input: RequestInfo,
      init?: RequestInit,
    ) => Promise<Response>,
  ) {
    const targetRoute = route || window.location.pathname;
    const extractedStoryId = targetRoute.includes("/stories/")
      ? parseInt(targetRoute.split("/stories/")[1]) || null
      : null;

    log("Starting tracking:", {
      targetRoute,
      extractedStoryId,
      wasAlreadyTracking: this.isTracking(),
    });

    // Get tracking ID from backend
    if (authenticatedFetch) {
      await this.getTrackingId(
        targetRoute,
        extractedStoryId,
        authenticatedFetch,
      );
    }

    this.startTime = Date.now();
    this.setupPageLeaveTracking();
  }

  async endTracking(
    authenticatedFetch: (
      input: RequestInfo,
      init?: RequestInit,
    ) => Promise<Response>,
    useBeacon = false,
  ) {
    log("Attempting to end tracking:", {
      hasStartTime: !!this.startTime,
      hasTrackingId: !!this.trackingId,
      useBeacon,
    });

    if (!this.startTime || !this.trackingId) {
      log("Ending early - no start time or tracking ID");
      return;
    }

    const elapsedMs = Date.now() - this.startTime;
    log("Calculated elapsed time:", { elapsedMs });

    // Only record if elapsed time is meaningful (>1 second)
    if (elapsedMs < 1000) {
      log("Ending early - elapsed time too short");
      this.reset();
      return;
    }

    const payload = {
      elapsed_ms: elapsedMs,
      tracking_id: this.trackingId,
    };

    log("Recording time:", payload);

    if (useBeacon && navigator.sendBeacon) {
      const formData = new FormData();
      formData.append("elapsed_ms", payload.elapsed_ms.toString());
      if (payload.tracking_id) {
        formData.append("tracking_id", payload.tracking_id);
      }
      const beaconSent = navigator.sendBeacon(
        "/api/time-tracking/record",
        formData,
      );
      log("Beacon sent:", beaconSent);
    } else {
      try {
        log("Sending POST request to record time");
        const response = await authenticatedFetch("/api/time-tracking/record", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        log("POST request completed:", response.status, response.statusText);
      } catch (error) {
        console.error("Failed to record time tracking:", error);
      }
    }

    this.reset();
  }

  private reset() {
    log("Resetting tracker");
    this.startTime = null;
    this.trackingId = null;
    this.cleanupPageLeaveTracking();
  }

  private async getTrackingId(
    route: string,
    storyId: number | null,
    authenticatedFetch: (
      input: RequestInfo,
      init?: RequestInit,
    ) => Promise<Response>,
  ) {
    try {
      log("Getting tracking ID");
      const payload: { route: string; story_id?: number } = { route };
      if (storyId) {
        payload.story_id = storyId;
      }

      const response = await authenticatedFetch("/api/time-tracking/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data = await response.json();
      this.trackingId = data.tracking_id;
      log("Got tracking ID:", this.trackingId);
    } catch (error) {
      log("Failed to get tracking ID:", error);
    }
  }

  private setupPageLeaveTracking() {
    const handlePageLeave = () => {
      log("Page leave detected");
      // Use beacon for page leave to ensure delivery
      this.endTracking(() => Promise.resolve(new Response()), true);
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === "hidden") {
        log("Page hidden - tab switch detected");
        // Don't end tracking on tab switch, only record current time
        if (this.startTime && this.trackingId) {
          const elapsedMs = Date.now() - this.startTime;
          if (elapsedMs >= 1000) {
            const payload = {
              elapsed_ms: elapsedMs,
              tracking_id: this.trackingId,
            };
            log("Recording time on tab switch:", payload);
            const formData = new FormData();
            formData.append("elapsed_ms", payload.elapsed_ms.toString());
            formData.append("tracking_id", payload.tracking_id);
            navigator.sendBeacon("/api/time-tracking/record", formData);
          }
          // Reset start time for next session but keep tracking ID
          this.startTime = Date.now();
        }
      }
    };

    window.addEventListener("beforeunload", handlePageLeave);
    window.addEventListener("pagehide", handlePageLeave);
    document.addEventListener("visibilitychange", handleVisibilityChange);

    this.cleanup = () => {
      window.removeEventListener("beforeunload", handlePageLeave);
      window.removeEventListener("pagehide", handlePageLeave);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }

  private cleanupPageLeaveTracking() {
    if (this.cleanup) {
      log("Cleaning up page leave tracking");
      this.cleanup();
      this.cleanup = null;
    }
  }

  isTracking() {
    return this.startTime !== null;
  }
}

const globalTracker = new TimeTracker();

export function useTimeTracking() {
  const authenticatedFetch = useAuthenticatedFetch();
  const hasStartedRef = useRef(false);

  const startTracking = useCallback(
    async (route?: string) => {
      log("useTimeTracking startTracking called:", {
        route,
        alreadyStarted: hasStartedRef.current,
      });
      if (hasStartedRef.current) return;

      hasStartedRef.current = true;
      await globalTracker.startTracking(route, authenticatedFetch);
    },
    [authenticatedFetch],
  );

  const endTracking = useCallback(async () => {
    log("useTimeTracking endTracking called:", {
      wasStarted: hasStartedRef.current,
    });
    if (!hasStartedRef.current) return;

    hasStartedRef.current = false;
    await globalTracker.endTracking(authenticatedFetch);
  }, [authenticatedFetch]);

  useEffect(() => {
    return () => {
      // Only cleanup on component unmount, not on tab switch
      if (hasStartedRef.current && document.visibilityState !== "hidden") {
        log("useTimeTracking cleanup effect triggered");
        hasStartedRef.current = false;
        globalTracker.endTracking(authenticatedFetch);
      }
    };
  }, [authenticatedFetch]);

  return { startTracking, endTracking };
}
