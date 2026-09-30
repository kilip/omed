import { createContext, type PropsWithChildren, useContext } from "react";
import type { Session, User } from "~/shared/auth";
import type { AuthContext } from "~/shared/contexts/auth";

const AuthCtx = createContext<AuthContext | null>(null);

export function AuthProvider({
  user,
  session,
  token = "",
  loading = false,
  children,
}: PropsWithChildren<{
  user: User;
  session: Session;
  token?: string;
  loading?: boolean;
}>) {
  return (
    <AuthCtx.Provider value={{ user, session, token, loading }}>
      {children}
    </AuthCtx.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthCtx);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
