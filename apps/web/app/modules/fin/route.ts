import {
  index,
  prefix,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

export default prefix("fin", [
  index("modules/fin/routes/dashboard.tsx"),
  route("accounts", "modules/fin/routes/accounts.tsx"),
  route("transactions", "modules/fin/routes/transactions.tsx"),
  route("transactions/new", "modules/fin/routes/transaction-new.tsx"),
  route("transactions/:id", "modules/fin/routes/transaction.tsx"),
  route("transactions/:id/edit", "modules/fin/routes/transaction-edit.tsx"),
  route("journal", "modules/fin/routes/journal.tsx"),
  route("journal/:id", "modules/fin/routes/entry.tsx"),
  route("ledger", "modules/fin/routes/ledger.tsx"),
  route("trial-balance", "modules/fin/routes/trial-balance.tsx"),
  route("setup", "modules/fin/routes/setup.tsx"),
]) satisfies RouteConfigEntry[];
