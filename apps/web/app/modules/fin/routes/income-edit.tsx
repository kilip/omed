import { useSuspenseQuery } from "@tanstack/react-query";
import { redirect } from "react-router";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { CashflowForm } from "../components";
import {
  accountsQueryOptions,
  currenciesQueryOptions,
  incomeDetailQueryOptions,
} from "../queries";
import type { Route } from "./+types/income-edit";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  const [item] = await Promise.all([
    queryClient.ensureQueryData(incomeDetailQueryOptions(wid, id)),
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(currenciesQueryOptions(wid)),
  ]);
  if (item.entryStatus !== "DRAFT") {
    throw redirect(`/fin/income/${id}`);
  }
  return { wid, id };
}

export default function IncomeEditRoute({ loaderData }: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { data: item } = useSuspenseQuery(incomeDetailQueryOptions(wid, id));

  return (
    <CashflowForm
      wid={wid}
      flow="INCOME"
      transactionId={item.id}
      initialValues={item}
    />
  );
}
