import {
  index,
  prefix,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

export default prefix("fin", [
  index("modules/fin/routes/launcher.tsx"),
  route("overview", "modules/fin/routes/dashboard.tsx"),
  route("income", "modules/fin/routes/income.tsx"),
  route("income/new", "modules/fin/routes/income-new.tsx"),
  route("expense", "modules/fin/routes/expense.tsx"),
  route("expense/new", "modules/fin/routes/expense-new.tsx"),
  route("accounts", "modules/fin/routes/accounts.tsx"),
  route("periods", "modules/fin/routes/periods.tsx"),
  route("opening-balance", "modules/fin/routes/opening-balance.tsx"),
  route("transactions", "modules/fin/routes/transactions.tsx"),
  route("transactions/new", "modules/fin/routes/transaction-new.tsx"),
  route("transactions/:id", "modules/fin/routes/transaction.tsx"),
  route("transactions/:id/edit", "modules/fin/routes/transaction-edit.tsx"),
  route("journal", "modules/fin/routes/journal.tsx"),
  route("journal/new", "modules/fin/routes/entry-new.tsx"),
  route("journal/:id", "modules/fin/routes/entry.tsx"),
  route("ledger", "modules/fin/routes/ledger.tsx"),
  route("trial-balance", "modules/fin/routes/trial-balance.tsx"),
  route("onboarding", "modules/fin/routes/onboarding.tsx"),
]) satisfies RouteConfigEntry[];
