import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { PhaseReadiness } from "../types/admin";
import {
  getCachedStoryReadiness,
  setCachedStoryPhase,
  type StoryReadinessMap,
  type StoryReadinessPhase,
} from "../lib/storyReadiness";

interface StoryReadinessContextValue {
  readiness: StoryReadinessMap;
  setPhase: (phase: StoryReadinessPhase, value: PhaseReadiness) => void;
}

const StoryReadinessContext = createContext<StoryReadinessContextValue | null>(
  null,
);

export function StoryReadinessProvider({
  storyId,
  children,
}: {
  storyId: number;
  children: ReactNode;
}) {
  const [readiness, setReadiness] = useState<StoryReadinessMap>(() =>
    getCachedStoryReadiness(storyId),
  );

  useEffect(() => {
    setReadiness(getCachedStoryReadiness(storyId));
  }, [storyId]);

  const setPhase = useCallback(
    (phase: StoryReadinessPhase, value: PhaseReadiness) => {
      setReadiness((current) => {
        const next = setCachedStoryPhase(storyId, phase, value);
        return next === current ? current : next;
      });
    },
    [storyId],
  );

  const value = useMemo(() => ({ readiness, setPhase }), [readiness, setPhase]);
  return (
    <StoryReadinessContext.Provider value={value}>
      {children}
    </StoryReadinessContext.Provider>
  );
}

export function useStoryReadiness(): StoryReadinessContextValue {
  const ctx = useContext(StoryReadinessContext);
  if (!ctx) {
    throw new Error(
      "useStoryReadiness must be used within StoryReadinessProvider",
    );
  }
  return ctx;
}

export function useReportPhase(
  phase: StoryReadinessPhase,
  readiness: PhaseReadiness | null | undefined,
): void {
  const { setPhase } = useStoryReadiness();
  useEffect(() => {
    if (readiness) setPhase(phase, readiness);
  }, [phase, readiness, setPhase]);
}
