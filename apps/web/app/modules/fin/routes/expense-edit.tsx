import { useSuspenseQuery } from "@tanstack/react-query";
import { redirect } from "react-router";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { CashflowForm } from "../components";
import {
  accountsQueryOptions,
  currenciesQueryOptions,
  expenseDetailQueryOptions,
} from "../queries";
import type { Route } from "./+types/expense-edit";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  const [item] = await Promise.all([
    queryClient.ensureQueryData(expenseDetailQueryOptions(wid, id)),
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(currenciesQueryOptions(wid)),
  ]);
  if (item.entryStatus !== "DRAFT") {
    throw redirect(`/fin/expense/${id}`);
  }
  return { wid, id };
}

export default function ExpenseEditRoute({ loaderData }: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { data: item } = useSuspenseQuery(expenseDetailQueryOptions(wid, id));

  return (
    <CashflowForm
      wid={wid}
      flow="EXPENSE"
      transactionId={item.id}
      initialValues={item}
    />
  );
}
