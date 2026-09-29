import { createContext, type PropsWithChildren, useContext } from "react";
import type { Session, User } from "~/shared/auth";

const AuthCtx = createContext<{ user: User; session: Session } | null>(null);

export function AuthProvider({
  user,
  session,
  children,
}: PropsWithChildren<{ user: User; session: Session }>) {
  return (
    <AuthCtx.Provider value={{ user, session }}>{children}</AuthCtx.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthCtx);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
