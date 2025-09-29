import { useCallback, useRef } from "react";
import { useApiService } from "../services/api";
import type { NavigationGuidanceResponse } from "../types/api";

interface CachedGuidance {
  data: NavigationGuidanceResponse;
  timestamp: number;
}

const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes

export function useNavigationGuidance() {
  const api = useApiService();
  const cache = useRef<Map<string, CachedGuidance>>(new Map());

  const getNavigationGuidance = useCallback(
    async (storyId: string, currentPage: string): Promise<NavigationGuidanceResponse | null> => {
      const cacheKey = `${storyId}-${currentPage}`;
      const now = Date.now();

      // Check cache first
      const cached = cache.current.get(cacheKey);
      if (cached && now - cached.timestamp < CACHE_DURATION) {
        return cached.data;
      }

      try {
        const response = await api.getNavigationGuidance(storyId, currentPage);
        if (response.success && response.data) {
          // Cache the result
          cache.current.set(cacheKey, {
            data: response.data,
            timestamp: now
          });
          return response.data;
        }
        return null;
      } catch (error) {
        console.error("Failed to get navigation guidance:", error);
        return null;
      }
    },
    [api]
  );

  const clearCache = useCallback(() => {
    cache.current.clear();
  }, []);

  return {
    getNavigationGuidance,
    clearCache
  };
}
