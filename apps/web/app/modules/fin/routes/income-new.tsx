import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { CashflowForm } from "../components";
import { accountsQueryOptions, currenciesQueryOptions } from "../queries";
import type { Route } from "./+types/income-new";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(currenciesQueryOptions(wid)),
  ]);
  return { wid };
}

export default function IncomeNewRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  return <CashflowForm wid={wid} flow="INCOME" />;
}
