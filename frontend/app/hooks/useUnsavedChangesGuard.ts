import React from "react";
import { useBlocker } from "react-router";

const MESSAGE = "You have unsaved changes. Leave without saving?";

/**
 * Prompts the user before leaving the page while `hasUnsavedChanges` is true.
 * Covers both in-app React Router navigation and full page unloads.
 */
export function useUnsavedChangesGuard(hasUnsavedChanges: boolean) {
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      hasUnsavedChanges && currentLocation.pathname !== nextLocation.pathname,
  );

  React.useEffect(() => {
    if (blocker.state !== "blocked") return;
    if (window.confirm(MESSAGE)) {
      blocker.proceed();
    } else {
      blocker.reset();
    }
  }, [blocker]);

  React.useEffect(() => {
    if (!hasUnsavedChanges) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [hasUnsavedChanges]);
}
