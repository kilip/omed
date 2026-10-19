/**
 * PROVISIONAL DATA. Dashboard and trial balance run on this in-memory sample.
 * Journal entries, transactions, and ledger are wired to the real API.
 * When the backend ships dashboard and trial balance endpoints, replace these
 * with API queries. Amounts are whole rupiah.
 */

export type AccountType =
  | "ASSET"
  | "LIABILITY"
  | "EQUITY"
  | "REVENUE"
  | "EXPENSE";

export type NormalBalance = "DEBIT" | "CREDIT";

export type MockAccount = {
  code: string;
  name: string;
  type: AccountType;
  normalBalance: NormalBalance;
  /** Balance on 1 Oct 2026, on the account's normal side. */
  opening: number;
};

export type JournalLine = {
  account: string;
  label: string;
  debit: number;
  credit: number;
};

export type JournalEntry = {
  id: number;
  number: string;
  date: string;
  reference: string;
  status: "DRAFT" | "POSTED";
  lines: JournalLine[];
};

export const PERIOD_START = "2026-10-01";
export const PERIOD_END = "2026-10-14";

export const mockAccounts: MockAccount[] = [
  {
    code: "1110",
    name: "Cash on hand",
    type: "ASSET",
    normalBalance: "DEBIT",
    opening: 12_500_000,
  },
  {
    code: "1120",
    name: "Bank BCA",
    type: "ASSET",
    normalBalance: "DEBIT",
    opening: 150_000_000,
  },
  {
    code: "1130",
    name: "Accounts receivable",
    type: "ASSET",
    normalBalance: "DEBIT",
    opening: 42_750_000,
  },
  {
    code: "1210",
    name: "Equipment",
    type: "ASSET",
    normalBalance: "DEBIT",
    opening: 60_000_000,
  },
  {
    code: "2110",
    name: "Accounts payable",
    type: "LIABILITY",
    normalBalance: "CREDIT",
    opening: 31_200_000,
  },
  {
    code: "2120",
    name: "Tax payable",
    type: "LIABILITY",
    normalBalance: "CREDIT",
    opening: 5_600_000,
  },
  {
    code: "3100",
    name: "Owner's capital",
    type: "EQUITY",
    normalBalance: "CREDIT",
    opening: 150_000_000,
  },
  {
    code: "3200",
    name: "Retained earnings",
    type: "EQUITY",
    normalBalance: "CREDIT",
    opening: 35_200_000,
  },
  {
    code: "4100",
    name: "Sales revenue",
    type: "REVENUE",
    normalBalance: "CREDIT",
    opening: 105_250_000,
  },
  {
    code: "4200",
    name: "Service revenue",
    type: "REVENUE",
    normalBalance: "CREDIT",
    opening: 0,
  },
  {
    code: "5100",
    name: "Cost of goods sold",
    type: "EXPENSE",
    normalBalance: "DEBIT",
    opening: 62_000_000,
  },
  {
    code: "5210",
    name: "Salaries",
    type: "EXPENSE",
    normalBalance: "DEBIT",
    opening: 0,
  },
  {
    code: "5220",
    name: "Rent",
    type: "EXPENSE",
    normalBalance: "DEBIT",
    opening: 0,
  },
  {
    code: "5230",
    name: "Utilities",
    type: "EXPENSE",
    normalBalance: "DEBIT",
    opening: 0,
  },
];

export const mockEntries: JournalEntry[] = [
  {
    id: 43,
    number: "JE/2026/0043",
    date: "2026-10-14",
    reference: "Payroll October",
    status: "DRAFT",
    lines: [
      {
        account: "5210",
        label: "Salaries October",
        debit: 28_000_000,
        credit: 0,
      },
      {
        account: "1120",
        label: "Net pay from bank",
        debit: 0,
        credit: 27_000_000,
      },
      {
        account: "2120",
        label: "Income tax withheld",
        debit: 0,
        credit: 1_000_000,
      },
    ],
  },
  {
    id: 42,
    number: "JE/2026/0042",
    date: "2026-10-13",
    reference: "Electricity and internet",
    status: "POSTED",
    lines: [
      {
        account: "5230",
        label: "Utilities October",
        debit: 3_350_000,
        credit: 0,
      },
      {
        account: "1110",
        label: "Paid from petty cash",
        debit: 0,
        credit: 3_350_000,
      },
    ],
  },
  {
    id: 41,
    number: "JE/2026/0041",
    date: "2026-10-12",
    reference: "INV-0107 Maju Jaya",
    status: "POSTED",
    lines: [
      {
        account: "1120",
        label: "Payment received",
        debit: 42_000_000,
        credit: 0,
      },
      {
        account: "4200",
        label: "Service revenue",
        debit: 0,
        credit: 42_000_000,
      },
    ],
  },
  {
    id: 40,
    number: "JE/2026/0040",
    date: "2026-10-09",
    reference: "PO-220 supplier payment",
    status: "POSTED",
    lines: [
      {
        account: "2110",
        label: "Settle supplier invoice",
        debit: 18_600_000,
        credit: 0,
      },
      { account: "1120", label: "Bank transfer", debit: 0, credit: 18_600_000 },
    ],
  },
  {
    id: 39,
    number: "JE/2026/0039",
    date: "2026-10-05",
    reference: "Office rent October",
    status: "POSTED",
    lines: [
      { account: "5220", label: "Rent October", debit: 12_000_000, credit: 0 },
      { account: "1120", label: "Bank transfer", debit: 0, credit: 12_000_000 },
    ],
  },
  {
    id: 38,
    number: "JE/2026/0038",
    date: "2026-10-02",
    reference: "INV-0101 payment",
    status: "POSTED",
    lines: [
      {
        account: "1120",
        label: "Payment received",
        debit: 24_000_000,
        credit: 0,
      },
      {
        account: "1130",
        label: "Clear receivable",
        debit: 0,
        credit: 24_000_000,
      },
    ],
  },
];

/** Revenue and expense for the five months before the sample period. */
export const monthlyHistory = [
  { month: "2026-05", revenue: 24_000_000, expense: 16_000_000 },
  { month: "2026-06", revenue: 28_000_000, expense: 18_000_000 },
  { month: "2026-07", revenue: 31_000_000, expense: 21_000_000 },
  { month: "2026-08", revenue: 27_000_000, expense: 19_000_000 },
  { month: "2026-09", revenue: 33_000_000, expense: 22_000_000 },
];

export const sumDebit = (lines: JournalLine[]) =>
  lines.reduce((total, l) => total + l.debit, 0);

const movement = (account: MockAccount, line: JournalLine) =>
  account.normalBalance === "DEBIT"
    ? line.debit - line.credit
    : line.credit - line.debit;

/** Balance on the account's normal side after the posted entries. */
export function closingBalance(
  account: MockAccount,
  entries: JournalEntry[],
): number {
  return entries
    .filter((e) => e.status === "POSTED")
    .flatMap((e) => e.lines)
    .filter((l) => l.account === account.code)
    .reduce((total, l) => total + movement(account, l), account.opening);
}

export type TrialBalanceRow = {
  account: MockAccount;
  debit: number;
  credit: number;
};

export function trialBalance(entries: JournalEntry[]): TrialBalanceRow[] {
  return mockAccounts.map((account) => {
    const balance = closingBalance(account, entries);
    const onNormalSide = balance >= 0;
    const debitSide = (account.normalBalance === "DEBIT") === onNormalSide;
    const amount = Math.abs(balance);
    return {
      account,
      debit: debitSide ? amount : 0,
      credit: debitSide ? 0 : amount,
    };
  });
}

export function summary(entries: JournalEntry[]) {
  const total = (type: AccountType) =>
    mockAccounts
      .filter((a) => a.type === type)
      .reduce((sum, a) => sum + closingBalance(a, entries), 0);

  const revenue = total("REVENUE");
  const expense = total("EXPENSE");
  return {
    assets: total("ASSET"),
    liabilities: total("LIABILITY"),
    equity: total("EQUITY"),
    revenue,
    expense,
    netIncome: revenue - expense,
  };
}

/** Last six months: history plus the current month from posted entries. */
export function monthlySeries(entries: JournalEntry[]) {
  const month = PERIOD_END.slice(0, 7);
  const posted = entries.filter(
    (e) => e.status === "POSTED" && e.date.startsWith(month),
  );
  const flow = (type: AccountType) =>
    posted
      .flatMap((e) => e.lines)
      .reduce((sum, l) => {
        const account = mockAccounts.find((a) => a.code === l.account);
        return account?.type === type ? sum + movement(account, l) : sum;
      }, 0);

  return [
    ...monthlyHistory,
    { month, revenue: flow("REVENUE"), expense: flow("EXPENSE") },
  ];
}
