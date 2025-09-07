import React, { createContext, useContext, type ReactNode } from "react";
import { useUserSync, type UserInfo } from "../lib/userSync";

interface UserContextType {
  userInfo: UserInfo | null;
  loading: boolean;
  error: string | null;
  syncUser: () => Promise<void>;
  isLoaded: boolean;
  isSignedIn: boolean | undefined;
}

const UserContext = createContext<UserContextType | undefined>(undefined);

export function UserProvider({ children }: { children: ReactNode }) {
  const userSync = useUserSync();

  return (
    <UserContext.Provider value={userSync}>{children}</UserContext.Provider>
  );
}

export function useUserContext() {
  const context = useContext(UserContext);
  if (context === undefined) {
    throw new Error("useUserContext must be used within a UserProvider");
  }
  return context;
}
