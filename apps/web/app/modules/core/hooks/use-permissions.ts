import { useQuery } from "@tanstack/react-query";
import { permissionsQueryOptions } from "../queries";

export function usePermissions(wid: number) {
  const { data } = useQuery(permissionsQueryOptions(wid));
  return data ?? { roles: [], permissions: [], owner: false };
}
