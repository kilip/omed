import { useOutletContext } from "react-router";
import type { AuthContext } from "~/shared/contexts/auth";

export default function AccountLists() {
  const { token } = useOutletContext<AuthContext>();
  return <div>{token}</div>;
}
