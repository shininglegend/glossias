import { NavLink, useLocation } from "react-router";
import { useMemo } from "react";
import {
  SignedIn,
  SignedOut,
  UserButton,
  SignInButton,
} from "@clerk/react-router";

export default function NavBar() {
  const location = useLocation();
  const isAdmin = useMemo(
    () => location.pathname.startsWith("/admin"),
    [location.pathname],
  );

  return (
    <header className="sticky top-0 z-50 bg-slate-900 text-white border-b border-white/10">
      <div className="max-w-6xl mx-auto px-4 py-3 flex items-center justify-between gap-4">
        <NavLink to="/" className="font-bold tracking-tight">
          Logos Stories
        </NavLink>

        <nav className="inline-flex items-center gap-2" aria-label="Primary">
          <NavItem to="/" end>
            Home
          </NavItem>
          <NavItem to="/admin">Admin</NavItem>
        </nav>
      </div>

      {isAdmin && (
        <div
          role="navigation"
          aria-label="Admin"
          className="bg-gray-900 border-t border-white/10"
        >
          <div className="max-w-6xl mx-auto px-4 py-2 items-center gap-2">
            <NavItem to="/admin" end>
              Dashboard
            </NavItem>
            <NavItem to="/admin/stories/add">Add Story</NavItem>
          </div>
        </div>
      )}
      <SignedOut>
        <SignInButton />
      </SignedOut>
      <SignedIn>
        <UserButton />
      </SignedIn>
    </header>
  );
}

function NavItem({
  to,
  end,
  children,
}: {
  to: string;
  end?: boolean;
  children: React.ReactNode;
}) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        [
          "px-2 py-1 rounded-md text-slate-300 hover:text-white hover:bg-white/10",
          isActive ? "bg-blue-500/20 text-white" : "",
        ]
          .filter(Boolean)
          .join(" ")
      }
    >
      {children}
    </NavLink>
  );
}
