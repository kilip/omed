import { useOutletContext } from "react-router";
import type { AppContext } from "~/context";

export default function Status() {
  const { token, user } = useOutletContext<AppContext>();
  return (
    <div>
      {user.name} = {token}
    </div>
  );
}
