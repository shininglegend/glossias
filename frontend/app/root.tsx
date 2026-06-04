import {
  isRouteErrorResponse,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
} from "react-router";

import { ClerkProvider } from "@clerk/react-router";

import type { Route } from "./+types/root";
import stylesheet from "./app.css?url";
import NavBar from "./components/NavBar";
import { UserProvider } from "./contexts/UserContext";

// Clerk
const PUBLISHABLE_KEY = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY;

if (!PUBLISHABLE_KEY) {
  throw new Error("Add your Clerk Publishable Key to the .env file");
}

export const links: Route.LinksFunction = () => [
  { rel: "preconnect", href: "https://fonts.googleapis.com" },
  {
    rel: "preconnect",
    href: "https://fonts.gstatic.com",
    crossOrigin: "anonymous",
  },
  {
    rel: "stylesheet",
    href: "https://fonts.googleapis.com/css2?family=Inter:ital,opsz,wght@0,14..32,100..900;1,14..32,100..900&display=swap",
  },
  {
    rel: "stylesheet",
    href: "https://fonts.googleapis.com/icon?family=Material+Icons",
  },
  { rel: "stylesheet", href: stylesheet },
];

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        <link rel="icon" href="/logo.png" />
        <Meta />
        <Links />
      </head>
      <body className="min-h-screen flex flex-col">
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export function HydrateFallback() {
  return (
    <div className="min-h-screen bg-white flex items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
        <p className="text-gray-600">Loading glossias application...</p>
      </div>
    </div>
  );
}

export default function App() {
  return (
    <ClerkProvider publishableKey={PUBLISHABLE_KEY}>
      <UserProvider>
        <div id="app-shell" className="flex-1 flex flex-col">
          <NavBar />
          <div className="pt-16 p-4 container mx-auto flex-1">
            <main>
              <Outlet />
            </main>
          </div>
        </div>
      </UserProvider>
    </ClerkProvider>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let message = "Something went wrong";
  let details = "An unexpected error has occurred. Our team has been notified.";
  let stack: string | undefined;

  if (isRouteErrorResponse(error)) {
    message = error.status === 404 ? "Page Not Found" : "Application Error";
    details =
      error.status === 404
        ? "The page you are looking for doesn't exist or has been moved."
        : error.statusText || details;
  } else if (error && error instanceof Error) {
    details = error.message;
    stack = error.stack;
  }

  return (
    <main className="min-h-[70vh] flex items-center justify-center p-4">
      <div className="max-w-xl w-full bg-white dark:bg-slate-900 rounded-2xl shadow-xl border border-slate-200 dark:border-slate-800 p-8 text-center transition-all">
        <div className="w-16 h-16 bg-red-50 dark:bg-red-950/30 rounded-full flex items-center justify-center mx-auto mb-6 text-red-500 dark:text-red-400">
          <span className="material-icons text-3xl">error_outline</span>
        </div>
        
        <h1 className="text-3xl font-extrabold text-slate-900 dark:text-white tracking-tight mb-3">
          {message}
        </h1>
        
        <p className="text-slate-600 dark:text-slate-400 mb-8 leading-relaxed">
          {details}
        </p>

        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center mb-6">
          <a
            href="/"
            className="w-full sm:w-auto px-6 h-11 inline-flex items-center justify-center font-medium bg-[#3158CE] hover:bg-[#2746a5] text-white rounded-lg transition-colors shadow-sm"
          >
            <span className="material-icons mr-2 text-lg">home</span>
            Go Back Home
          </a>
          <button
            onClick={() => window.location.reload()}
            className="w-full sm:w-auto px-6 h-11 inline-flex items-center justify-center font-medium border border-slate-300 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 text-slate-700 dark:text-slate-300 rounded-lg transition-colors"
          >
            <span className="material-icons mr-2 text-lg">refresh</span>
            Reload Page
          </button>
        </div>

        {stack && (
          <details className="text-left mt-6 border-t border-slate-100 dark:border-slate-800 pt-6">
            <summary className="text-sm font-semibold text-slate-500 dark:text-slate-400 cursor-pointer select-none hover:text-slate-800 dark:hover:text-slate-200 transition-colors">
              Show technical details
            </summary>
            <div className="mt-4 p-4 bg-slate-50 dark:bg-slate-950 border border-slate-200 dark:border-slate-800 rounded-lg overflow-x-auto text-xs font-mono text-slate-700 dark:text-slate-300 max-h-64">
              <pre className="whitespace-pre-wrap">{stack}</pre>
            </div>
          </details>
        )}
      </div>
    </main>
  );
}
