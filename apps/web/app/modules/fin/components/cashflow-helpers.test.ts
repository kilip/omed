import dayjs from "dayjs";
import { describe, expect, it } from "vitest";
import type { FinancialSuggestion } from "~/modules/agent/queries";
import {
  applyAnalysis,
  type CashflowItemValue,
  checkTotalMismatch,
  computeLineAmount,
  computeTotal,
  mergeItems,
  type PrefillScalar,
} from "./cashflow-helpers";

describe("cashflow-helpers line calculation", () => {
  it("computes integer line amount correctly", () => {
    expect(computeLineAmount(6, 500000, 0)).toBe(3000000);
  });

  it("computes decimal line amount with rounding to currency decimals", () => {
    expect(computeLineAmount(2.5, 3333, 0)).toBe(8333);
  });

  it("computes total of the owner expense items", () => {
    const items: CashflowItemValue[] = [
      { accountId: 6710, description: "telur", quantity: 2, unitPrice: 55000 },
      { accountId: 6710, description: "indomie", quantity: 5, unitPrice: 3000 },
    ];
    expect(computeTotal(items)).toBe(125000);
  });
});

describe("mergeItems", () => {
  it("replaces a form holding only the blank default row", () => {
    const existing: CashflowItemValue[] = [{ quantity: 1, unitPrice: 0 }];
    const incoming: NonNullable<FinancialSuggestion["items"]> = [
      {
        accountId: 6710,
        description: "telur",
        quantity: "2",
        unitPrice: "55000",
      },
    ];
    const result = mergeItems(existing, incoming);
    expect(result).toEqual([
      { accountId: 6710, description: "telur", quantity: 2, unitPrice: 55000 },
    ]);
  });

  it("appends to a form with real rows", () => {
    const existing: CashflowItemValue[] = [
      { accountId: 6710, description: "telur", quantity: 2, unitPrice: 55000 },
    ];
    const incoming: NonNullable<FinancialSuggestion["items"]> = [
      {
        accountId: 6720,
        description: "indomie",
        quantity: "5",
        unitPrice: "3000",
      },
    ];
    const result = mergeItems(existing, incoming);
    expect(result).toEqual([
      { accountId: 6710, description: "telur", quantity: 2, unitPrice: 55000 },
      { accountId: 6720, description: "indomie", quantity: 5, unitPrice: 3000 },
    ]);
  });

  it("skips an incoming item equal to an existing one (same account, case-insensitive desc, qty, price)", () => {
    const existing: CashflowItemValue[] = [
      {
        accountId: 6710,
        description: "Telur Ayam",
        quantity: 2,
        unitPrice: 55000,
      },
    ];
    const incoming: NonNullable<FinancialSuggestion["items"]> = [
      {
        accountId: 6710,
        description: "telur ayam",
        quantity: "2",
        unitPrice: "55000",
      },
      {
        accountId: 6720,
        description: "Minyak",
        quantity: "1",
        unitPrice: "14000",
      },
    ];
    const result = mergeItems(existing, incoming);
    expect(result).toEqual([
      {
        accountId: 6710,
        description: "Telur Ayam",
        quantity: 2,
        unitPrice: 55000,
      },
      {
        accountId: 6720,
        description: "Minyak",
        quantity: 1,
        unitPrice: 14000,
      },
    ]);
  });

  it("produces one row from two identical incoming items", () => {
    const existing: CashflowItemValue[] = [{ quantity: 1, unitPrice: 0 }];
    const incoming: NonNullable<FinancialSuggestion["items"]> = [
      {
        accountId: 6710,
        description: "kopi",
        quantity: "1",
        unitPrice: "10000",
      },
      {
        accountId: 6710,
        description: "KOPI",
        quantity: "1",
        unitPrice: "10000",
      },
    ];
    const result = mergeItems(existing, incoming);
    expect(result).toEqual([
      { accountId: 6710, description: "kopi", quantity: 1, unitPrice: 10000 },
    ]);
  });

  it("keeps incoming accountId: null unset and converts quantity and unitPrice from strings to numbers", () => {
    const existing: CashflowItemValue[] = [{ quantity: 1, unitPrice: 0 }];
    const incoming: NonNullable<FinancialSuggestion["items"]> = [
      {
        accountId: null as unknown as number,
        description: "Uncategorized Item",
        quantity: "3.5",
        unitPrice: "12500",
      },
    ];
    const result = mergeItems(existing, incoming);
    expect(result).toEqual([
      {
        accountId: undefined,
        description: "Uncategorized Item",
        quantity: 3.5,
        unitPrice: 12500,
      },
    ]);
  });
});

describe("applyAnalysis", () => {
  it("in default mode: fills empty scalars and untouched defaults, leaves touched scalars alone", () => {
    const current = {
      transactionDate: dayjs("2026-01-01"), // untouched default
      currency: "IDR", // untouched default
      exchangeRate: 1, // untouched default
      paymentAccountId: undefined, // empty
      reference: "USER-REF", // touched with user value
      description: undefined, // empty
      items: [{ quantity: 1, unitPrice: 0 }],
    };

    const touchedFields = new Set<PrefillScalar>(["reference"]);
    const isTouched = (field: PrefillScalar) => touchedFields.has(field);

    const suggestion: FinancialSuggestion = {
      transactionDate: "2026-03-15",
      currency: "USD",
      exchangeRate: "16000",
      paymentAccountId: 1120,
      reference: "INV-999",
      description: "Hosting payment",
      items: [
        {
          accountId: 6710,
          description: "VPS",
          quantity: "1",
          unitPrice: "16000",
        },
      ],
    };

    const result = applyAnalysis({
      current,
      isTouched,
      suggestion,
      mode: "default",
    });

    expect(result.values.transactionDate?.format("YYYY-MM-DD")).toBe(
      "2026-03-15",
    );
    expect(result.values.currency).toBe("USD");
    expect(result.values.exchangeRate).toBe(16000);
    expect(result.values.paymentAccountId).toBe(1120);
    expect(result.values.reference).toBeUndefined(); // touched, so left alone
    expect(result.values.description).toBe("Hosting payment");
    expect(result.values.items).toEqual([
      { accountId: 6710, description: "VPS", quantity: 1, unitPrice: 16000 },
    ]);

    expect(result.filledFields).toContain("transactionDate");
    expect(result.filledFields).toContain("currency");
    expect(result.filledFields).toContain("exchangeRate");
    expect(result.filledFields).toContain("paymentAccountId");
    expect(result.filledFields).toContain("description");
    expect(result.filledFields).toContain("items");
    expect(result.filledFields).not.toContain("reference");
  });

  it("does not include untouched scalar in filledFields if suggestion value matches current value", () => {
    const current = {
      transactionDate: dayjs("2026-03-15"),
      currency: "IDR",
      exchangeRate: 1,
      paymentAccountId: 1110,
      reference: undefined,
      description: undefined,
      items: [{ quantity: 1, unitPrice: 0 }],
    };

    const isTouched = () => false;

    const suggestion: FinancialSuggestion = {
      transactionDate: "2026-03-15", // identical date
      currency: "IDR", // identical currency
      exchangeRate: "1", // identical exchange rate
      paymentAccountId: 1110, // identical account
      reference: "INV-100", // new
      description: "Office coffee", // new
    };

    const result = applyAnalysis({
      current,
      isTouched,
      suggestion,
      mode: "default",
    });

    expect(result.filledFields).toContain("reference");
    expect(result.filledFields).toContain("description");
    expect(result.filledFields).not.toContain("transactionDate");
    expect(result.filledFields).not.toContain("currency");
    expect(result.filledFields).not.toContain("exchangeRate");
    expect(result.filledFields).not.toContain("paymentAccountId");
  });

  it("null suggestion fields change nothing in either mode", () => {
    const current = {
      transactionDate: dayjs("2026-01-01"),
      currency: "IDR",
      exchangeRate: 1,
      paymentAccountId: 1110,
      reference: "ORIG-REF",
      description: "Orig desc",
      items: [
        { accountId: 10, description: "item", quantity: 1, unitPrice: 100 },
      ],
    };

    const suggestion: FinancialSuggestion = {
      transactionDate: undefined,
      currency: undefined,
      exchangeRate: undefined,
      paymentAccountId: undefined,
      reference: undefined,
      description: undefined,
      items: undefined,
    };

    const defaultResult = applyAnalysis({
      current,
      isTouched: () => false,
      suggestion,
      mode: "default",
    });
    expect(defaultResult.filledFields).toEqual([]);
    expect(defaultResult.values).toEqual({});

    const overwriteResult = applyAnalysis({
      current,
      isTouched: () => true,
      suggestion,
      mode: "overwrite",
    });
    expect(overwriteResult.filledFields).toEqual([]);
    expect(overwriteResult.values).toEqual({});
  });

  it("payment_proof suggestion returns no items in either mode and filledFields does not contain items", () => {
    const current = {
      transactionDate: dayjs("2026-01-01"),
      items: [
        {
          accountId: 6710,
          description: "Existing item",
          quantity: 1,
          unitPrice: 50000,
        },
      ],
    };

    const suggestion: FinancialSuggestion = {
      documentType: "payment_proof",
      reference: "TRANSFER-123",
      items: [
        {
          accountId: 9999,
          description: "Ignored item",
          quantity: "1",
          unitPrice: "50000",
        },
      ],
    };

    const defaultRes = applyAnalysis({
      current,
      isTouched: () => false,
      suggestion,
      mode: "default",
    });
    expect(defaultRes.values.items).toBeUndefined();
    expect(defaultRes.filledFields).not.toContain("items");
    expect(defaultRes.values.reference).toBe("TRANSFER-123");

    const overwriteRes = applyAnalysis({
      current,
      isTouched: () => true,
      suggestion,
      mode: "overwrite",
    });
    expect(overwriteRes.values.items).toBeUndefined();
    expect(overwriteRes.filledFields).not.toContain("items");
    expect(overwriteRes.values.reference).toBe("TRANSFER-123");
  });

  it("in overwrite mode: applies every non-null scalar regardless of touched state, and replaces items", () => {
    const current = {
      transactionDate: dayjs("2026-01-01"),
      currency: "IDR",
      exchangeRate: 1,
      paymentAccountId: 1110,
      reference: "USER-REF",
      description: "User description",
      items: [
        {
          accountId: 6710,
          description: "Old item 1",
          quantity: 1,
          unitPrice: 1000,
        },
        {
          accountId: 6720,
          description: "Old item 2",
          quantity: 2,
          unitPrice: 2000,
        },
      ],
    };

    // User touched all fields
    const isTouched = () => true;

    const suggestion: FinancialSuggestion = {
      transactionDate: "2026-04-20",
      currency: "USD",
      exchangeRate: "16200",
      paymentAccountId: 1120,
      reference: "AI-REF",
      description: "AI description",
      items: [
        {
          accountId: 6730,
          description: "New AI item",
          quantity: "3",
          unitPrice: "5000",
        },
      ],
    };

    const result = applyAnalysis({
      current,
      isTouched,
      suggestion,
      mode: "overwrite",
    });

    expect(result.values.transactionDate?.format("YYYY-MM-DD")).toBe(
      "2026-04-20",
    );
    expect(result.values.currency).toBe("USD");
    expect(result.values.exchangeRate).toBe(16200);
    expect(result.values.paymentAccountId).toBe(1120);
    expect(result.values.reference).toBe("AI-REF");
    expect(result.values.description).toBe("AI description");
    expect(result.values.items).toEqual([
      {
        accountId: 6730,
        description: "New AI item",
        quantity: 3,
        unitPrice: 5000,
      },
    ]);

    expect(result.filledFields).toEqual(
      expect.arrayContaining([
        "transactionDate",
        "currency",
        "exchangeRate",
        "paymentAccountId",
        "reference",
        "description",
        "items",
      ]),
    );
  });
});

describe("checkTotalMismatch", () => {
  it("for payment_proof: compares documentTotal with formTotal", () => {
    const suggestion: FinancialSuggestion = {
      documentType: "payment_proof",
      documentTotal: "100000.00",
      currency: "IDR",
      items: [{ description: "Ignored", quantity: "1", unitPrice: "50000" }],
    };

    // Equal amounts -> null
    expect(
      checkTotalMismatch({
        suggestion,
        formTotal: 100000,
        formCurrency: "IDR",
      }),
    ).toBeNull();

    // Difference of 0.01 -> null
    expect(
      checkTotalMismatch({
        suggestion,
        formTotal: 100000.01,
        formCurrency: "IDR",
      }),
    ).toBeNull();

    // Difference of 0.02 -> returns { fileTotal, compared }
    expect(
      checkTotalMismatch({
        suggestion,
        formTotal: 100000.02,
        formCurrency: "IDR",
      }),
    ).toEqual({ fileTotal: 100000, compared: 100000.02 });
  });

  it("for other document types: compares documentTotal with sum of suggestion items", () => {
    const suggestion: FinancialSuggestion = {
      documentType: "receipt",
      documentTotal: "75000.00",
      currency: "IDR",
      items: [
        { description: "Item 1", quantity: "2", unitPrice: "25000" },
        { description: "Item 2", quantity: "1", unitPrice: "25000" },
      ],
    };

    // Item sum is 75000 -> matches documentTotal -> null
    expect(
      checkTotalMismatch({
        suggestion,
        formTotal: 999999, // formTotal is ignored for non-payment_proof
        formCurrency: "IDR",
      }),
    ).toBeNull();

    // Mismatched documentTotal
    const mismatchedSuggestion: FinancialSuggestion = {
      ...suggestion,
      documentTotal: "80000.00",
    };

    expect(
      checkTotalMismatch({
        suggestion: mismatchedSuggestion,
        formTotal: 999999,
        formCurrency: "IDR",
      }),
    ).toEqual({ fileTotal: 80000, compared: 75000 });
  });

  it("skips check (returns null) when suggestion currency differs from formCurrency", () => {
    const suggestion: FinancialSuggestion = {
      documentType: "payment_proof",
      documentTotal: "100.00",
      currency: "USD",
    };

    expect(
      checkTotalMismatch({
        suggestion,
        formTotal: 1600000,
        formCurrency: "IDR",
      }),
    ).toBeNull();
  });

  it("does not skip check when suggestion currency is null or undefined", () => {
    const suggestion: FinancialSuggestion = {
      documentType: "payment_proof",
      documentTotal: "50000.00",
      currency: undefined,
    };

    expect(
      checkTotalMismatch({
        suggestion,
        formTotal: 60000,
        formCurrency: "IDR",
      }),
    ).toEqual({ fileTotal: 50000, compared: 60000 });
  });

  it("returns null when documentTotal is missing or empty", () => {
    const suggestion: FinancialSuggestion = {
      documentType: "receipt",
      documentTotal: undefined,
      currency: "IDR",
      items: [{ quantity: "1", unitPrice: "1000" }],
    };

    expect(
      checkTotalMismatch({
        suggestion,
        formTotal: 1000,
        formCurrency: "IDR",
      }),
    ).toBeNull();
  });
});
