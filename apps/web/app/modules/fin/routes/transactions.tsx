import { requireWorkspaceId } from "~/lib/workspace";
import type { Route } from "./+types/transactions";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export default function TransactionsRoute({
  loaderData,
}: Route.ComponentProps) {
  return <div>Transactions placeholder for wid: {loaderData.wid}</div>;
}
