import { NavLink, useLocation } from "react-router";
import { useMemo } from "react";
import {
  SignedIn,
  SignedOut,
  UserButton,
  SignInButton,
} from "@clerk/react-router";
import { useUserContext } from "../contexts/UserContext";

export default function NavBar() {
  const location = useLocation();
  const { userInfo, loading } = useUserContext();
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
            {userInfo?.is_super_admin && (
              <NavItem to="/admin/courses">Courses</NavItem>
            )}
            <NavItem to="/admin/stories/add">Add Story</NavItem>
          </div>
        </div>
      )}

      <div className="flex items-center gap-3">
        <SignedOut>
          <SignInButton />
        </SignedOut>
        <SignedIn>
          {userInfo && !loading && (
            <div className="text-xs text-slate-300">
              {userInfo.is_super_admin ? (
                <span className="bg-red-500/20 text-red-300 px-2 py-1 rounded">
                  Super Admin
                </span>
              ) : (userInfo.course_admin_rights && userInfo.course_admin_rights.length > 0) ? (
                <span className="bg-blue-500/20 text-blue-300 px-2 py-1 rounded">
                  Course Admin
                </span>
              ) : (
                <span className="text-slate-400">User</span>
              )}
            </div>
          )}
          <UserButton />
        </SignedIn>
      </div>
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
