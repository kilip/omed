import { Outlet, redirect } from "react-router";
import { type DecimalStyle, setActiveDecimalStyle } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { finSettingsQueryOptions } from "../queries";

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const settings = await getQueryClient().ensureQueryData(
    finSettingsQueryOptions(wid),
  );

  setActiveDecimalStyle((settings.decimalStyle as DecimalStyle) ?? "comma");

  if (!settings.onboarding) {
    return redirect("/fin/onboarding");
  }

  return { wid, settings };
}

export function FinLayout() {
  return <Outlet />;
}

export default FinLayout;
