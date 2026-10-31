import { useSuspenseQuery } from "@tanstack/react-query";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { CashflowDetail } from "../components/cashflow-detail";
import { incomeDetailQueryOptions } from "../queries";
import type { Route } from "./+types/income-detail";

export async function clientLoader({ params }: Route.ClientLoaderArgs) {
  const wid = await requireWorkspaceId();
  const id = Number(params.id);
  const queryClient = getQueryClient();
  await queryClient.ensureQueryData(incomeDetailQueryOptions(wid, id));
  return { wid, id };
}

export default function IncomeDetailRoute({
  loaderData,
}: Route.ComponentProps) {
  const { wid, id } = loaderData;
  const { data: item } = useSuspenseQuery(incomeDetailQueryOptions(wid, id));

  return <CashflowDetail wid={wid} flow="INCOME" item={item} />;
}
