import React, { createContext, useContext, useState } from "react";

interface ErrorBannerContextType {
  showError: boolean;
  setShowError: (show: boolean) => void;
}

const ErrorBannerContext = createContext<ErrorBannerContextType | undefined>(
  undefined,
);

export function ErrorBannerProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [showError, setShowError] = useState(false);

  return (
    <ErrorBannerContext.Provider value={{ showError, setShowError }}>
      {showError && (
        <div className="fixed top-16 left-0 right-0 z-50 bg-red-600 text-white px-4 py-3 shadow-lg">
          <div className="mt-2 max-w-7xl mx-auto flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
            <div className="flex items-center gap-3">
              <span className="text-xl">⚠️</span>
              <div>
                <p className="font-semibold">Server Error</p>
                <p className="text-sm">
                  Something went wrong. The developer has been notified.
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => window.location.reload()}
                className="bg-white text-red-600 px-4 py-2 rounded font-semibold hover:bg-red-50 transition-colors text-sm"
              >
                Reload Page
              </button>
              <button
                onClick={() => setShowError(false)}
                className="text-white hover:text-red-200 px-2 text-xl"
                aria-label="Dismiss"
              >
                ✕
              </button>
            </div>
          </div>
        </div>
      )}
      {children}
    </ErrorBannerContext.Provider>
  );
}

export function useErrorBanner() {
  const context = useContext(ErrorBannerContext);
  if (!context) {
    throw new Error("useErrorBanner must be used within ErrorBannerProvider");
  }
  return context;
}
