import { useAuth } from "@clerk/react-router";
import { useCallback } from "react";
import { useNavigate } from "react-router";

export function useAuthenticatedFetch() {
  const { getToken, isSignedIn } = useAuth();
  const navigate = useNavigate();

  const authenticatedFetch = useCallback(
    async (input: RequestInfo, init: RequestInit = {}) => {
      // First attempt
      let token = await getToken();
      const headers = new Headers(init.headers || {});
      if (token) headers.set("Authorization", `Bearer ${token}`);
      
      let response = await fetch(input, { ...init, headers });
      
      // If we get a 401, try to refresh the token and retry once
      if (response.status === 401 && isSignedIn) {
        console.log("Received 401, attempting to refresh token and retry...");
        
        // Force token refresh by skipping cache
        token = await getToken({ skipCache: true });
        
        if (token) {
          // Retry the request with the new token
          const retryHeaders = new Headers(init.headers || {});
          retryHeaders.set("Authorization", `Bearer ${token}`);
          response = await fetch(input, { ...init, headers: retryHeaders });
          
          // If still 401 after retry, redirect to landing page
          if (response.status === 401) {
            console.log("Still 401 after retry, redirecting to landing page...");
            navigate("/");
          }
        } else {
          // No token available, redirect to landing page
          console.log("No token available after refresh, redirecting to landing page...");
          navigate("/");
        }
      }
      
      return response;
    },
    [getToken, isSignedIn, navigate],
  );

  return authenticatedFetch;
}
