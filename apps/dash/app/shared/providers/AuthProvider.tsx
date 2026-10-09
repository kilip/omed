import type { Session, User } from "@omed/better-auth";
import {
  createContext,
  type PropsWithChildren,
  useContext,
  useEffect,
  useState,
} from "react";
import type { AuthContext } from "~/contexts/auth";
import i18n, { type SupportedLanguage } from "~/lib/i18n";
import { updateCachedAuthUser } from "~/middleware/auth";

const isLocale = (v: unknown): v is SupportedLanguage =>
  v === "en" || v === "id";

export interface AuthContextValue extends AuthContext {
  updateUser: (patch: Partial<User>) => void;
}

const AuthCtx = createContext<AuthContextValue | null>(null);

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
  const [currentUser, setCurrentUser] = useState<User>(user);

  useEffect(() => {
    setCurrentUser(user);
    if (isLocale(user?.locale) && user.locale !== i18n.resolvedLanguage) {
      void i18n.changeLanguage(user.locale);
    }
  }, [user]);

  const updateUser = (patch: Partial<User>) => {
    setCurrentUser((prev) => ({ ...prev, ...patch }));
    updateCachedAuthUser(patch);
    if (isLocale(patch.locale) && patch.locale !== i18n.resolvedLanguage) {
      void i18n.changeLanguage(patch.locale);
    }
  };

  return (
    <AuthCtx.Provider
      value={{ user: currentUser, session, token, loading, updateUser }}
    >
      {children}
    </AuthCtx.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthCtx);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}

/** Like `useAuth`, but returns `null` outside `AuthProvider` (e.g. /login). */
export function useOptionalAuth() {
  return useContext(AuthCtx);
}
