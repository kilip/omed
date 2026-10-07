import { createContext } from "react-router";
import type { Session, User } from "~/lib/auth";

export interface AuthContext {
  token: string;
  user: User;
  session: Session;
  loading: boolean;
}

export const authContext = createContext<AuthContext>();
