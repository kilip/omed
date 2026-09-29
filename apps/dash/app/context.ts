import { createContext } from "react-router";
import type { User } from "./shared/auth";

export interface AppContext {
  token: string;
  user: User;
  loading: boolean;
}

export const appContext = createContext<AppContext>();
