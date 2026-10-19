import { requireWorkspaceId } from "~/lib/workspace";
import type { Route } from "./+types/transaction";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  return { wid, id };
}

export default function TransactionRoute({ loaderData }: Route.ComponentProps) {
  return (
    <div>
      Transaction {loaderData.id} placeholder for wid: {loaderData.wid}
    </div>
  );
}
