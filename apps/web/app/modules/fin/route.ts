import {
  index,
  prefix,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

export default prefix("fin", [
  index("modules/fin/routes/dashboard.tsx"),
  route("accounts", "modules/fin/routes/accounts.tsx"),
  route("journal", "modules/fin/routes/journal.tsx"),
  route("ledger", "modules/fin/routes/ledger.tsx"),
  route("trial-balance", "modules/fin/routes/trial-balance.tsx"),
]) satisfies RouteConfigEntry[];
