import { useSuspenseQuery } from "@tanstack/react-query";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { CashflowDetail } from "../components/cashflow-detail";
import { expenseDetailQueryOptions } from "../queries";
import type { Route } from "./+types/expense-detail";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  await queryClient.ensureQueryData(expenseDetailQueryOptions(wid, id));
  return { wid, id };
}

export default function ExpenseDetailRoute({
  loaderData,
}: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { data: item } = useSuspenseQuery(expenseDetailQueryOptions(wid, id));

  return <CashflowDetail wid={wid} flow="EXPENSE" item={item} />;
}
