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
import { ErrorBannerProvider } from "./contexts/ErrorBannerContext";
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
      <ErrorBannerProvider>
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
      </ErrorBannerProvider>
    </ClerkProvider>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let message = "Oops!";
  let details = "An unexpected error occurred.";
  let stack: string | undefined;

  if (isRouteErrorResponse(error)) {
    if (error.status === 404) {
      return (
        <main className="pt-16 p-4 container mx-auto max-w-2xl text-center">
          <h1 className="text-6xl font-bold text-gray-800 mb-4">404</h1>
          <h2 className="text-2xl font-semibold text-gray-700 mb-6">
            Page Not Found
          </h2>
          <p className="text-gray-600 mb-8">
            We've tried to express how lost you are in every language we know,
            but somehow none of them are sufficient...
          </p>
          <div className="bg-gray-50 rounded-lg p-6 mb-8 text-left space-y-2">
            <p className="font-mono text-sm">
              🇬🇧 English: "We can't find that page"
            </p>
            <p className="font-mono text-sm" dir="rtl">
              🇮🇱 עברית: "אנחנו לא יכולים למצוא את הדף הזה"
            </p>
            <p className="font-mono text-sm">
              🇪🇸 Español: "No podemos encontrar esa página"
            </p>
            <p className="font-mono text-sm">
              🇫🇷 Français: "Nous ne trouvons pas cette page"
            </p>
            <p className="font-mono text-sm">
              🇩🇪 Deutsch: "Wir können diese Seite nicht finden"
            </p>
            <p className="font-mono text-sm">
              🇯🇵 日本語: "そのページが見つかりません"
            </p>
            <p className="font-mono text-sm" dir="rtl">
              🇸🇦 العربية: "لا يمكننا العثور على هذه الصفحة"
            </p>
            <p className="font-mono text-sm">
              🇷🇺 Русский: "Мы не можем найти эту страницу"
            </p>
          </div>
          <a
            href="/"
            className="inline-block bg-blue-600 hover:bg-blue-700 text-white font-semibold px-6 py-3 rounded-lg transition-colors"
          >
            Go Home
          </a>
        </main>
      );
    } else if (error.status >= 500) {
      return (
        <main className="pt-16 p-4 container mx-auto max-w-2xl text-center">
          <h1 className="text-6xl font-bold text-red-600 mb-4">500</h1>
          <h2 className="text-2xl font-semibold text-gray-700 mb-6">
            Server Error
          </h2>
          <div className="bg-red-50 border border-red-200 rounded-lg p-6 mb-8">
            <p className="text-gray-700 mb-4">
              Something went wrong on our end. The developer has been notified.
            </p>
            <p className="text-gray-600">
              Please try reloading the page, or come back in a few moments.
            </p>
          </div>
          <button
            onClick={() => window.location.reload()}
            className="inline-block bg-blue-600 hover:bg-blue-700 text-white font-semibold px-6 py-3 rounded-lg transition-colors mr-4"
          >
            Reload Page
          </button>
          <a
            href="/"
            className="inline-block bg-gray-600 hover:bg-gray-700 text-white font-semibold px-6 py-3 rounded-lg transition-colors"
          >
            Go Home
          </a>
        </main>
      );
    }
    
    message = `Error ${error.status}`;
    details = error.statusText || details;
  } else if (import.meta.env.DEV && error && error instanceof Error) {
    details = error.message;
    stack = error.stack;
  }

  return (
    <main className="pt-16 p-4 container mx-auto">
      <h1>{message}</h1>
      <p>{details}</p>
      {stack && (
        <pre className="w-full p-4 overflow-x-auto">
          <code>{stack}</code>
        </pre>
      )}
    </main>
  );
}
