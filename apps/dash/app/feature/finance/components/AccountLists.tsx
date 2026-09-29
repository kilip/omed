import { useOutletContext } from "react-router";
import type { AppContext } from "~/context";

export default function AccountLists() {
  const { token } = useOutletContext<AppContext>();
  return <div>{token}</div>;
}
