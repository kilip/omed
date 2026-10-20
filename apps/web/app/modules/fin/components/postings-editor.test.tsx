import { describe, expect, it } from "vitest";
import {
  apiToPostingLines,
  calculatePostingsSummary,
  type PostingLineValue,
  postingLinesToApi,
} from "./postings-editor";

describe("postings-editor helpers", () => {
  describe("postingLinesToApi", () => {
    it("converts debit and credit form lines to API direction and positive amount", () => {
      const formLines: PostingLineValue[] = [
        {
          accountId: 1,
          debit: 150000,
          credit: undefined,
          description: "Office supplies",
        },
        { accountId: 2, debit: undefined, credit: 150000, description: "Cash" },
      ];

      const apiPostings = postingLinesToApi(formLines, 1, "IDR");
      expect(apiPostings).toEqual([
        {
          accountId: 1,
          direction: "DEBIT",
          amount: 150000,
          baseAmount: 150000,
          description: "Office supplies",
        },
        {
          accountId: 2,
          direction: "CREDIT",
          amount: 150000,
          baseAmount: 150000,
          description: "Cash",
        },
      ]);
    });

    it("applies exchange rate correctly to baseAmount", () => {
      const formLines: PostingLineValue[] = [
        { accountId: 10, debit: 100, credit: undefined },
        { accountId: 20, debit: undefined, credit: 100 },
      ];

      const apiPostings = postingLinesToApi(formLines, 16000, "USD");
      expect(apiPostings).toEqual([
        {
          accountId: 10,
          direction: "DEBIT",
          amount: 100,
          baseAmount: 1600000,
          description: undefined,
        },
        {
          accountId: 20,
          direction: "CREDIT",
          amount: 100,
          baseAmount: 1600000,
          description: undefined,
        },
      ]);
    });

    it("filters out empty or zero amount lines", () => {
      const formLines: PostingLineValue[] = [
        { accountId: 1, debit: 0, credit: 0 },
        { accountId: 2, debit: 500, credit: undefined },
        { accountId: undefined, debit: 500, credit: undefined },
      ];

      const apiPostings = postingLinesToApi(formLines, 1, "IDR");
      expect(apiPostings.length).toBe(1);
      expect(apiPostings[0]?.accountId).toBe(2);
    });
  });

  describe("apiToPostingLines", () => {
    it("converts API posting responses to form debit/credit lines", () => {
      const apiPostings = [
        {
          id: 101,
          accountId: 5,
          direction: "DEBIT" as const,
          amount: 250000,
          baseAmount: 250000,
          description: "Expense",
        },
        {
          id: 102,
          accountId: 6,
          direction: "CREDIT" as const,
          amount: 250000,
          baseAmount: 250000,
          description: "Bank",
        },
      ];

      const lines = apiToPostingLines(apiPostings);
      expect(lines).toEqual([
        {
          id: 101,
          accountId: 5,
          debit: 250000,
          credit: undefined,
          description: "Expense",
          baseAmount: 250000,
        },
        {
          id: 102,
          accountId: 6,
          debit: undefined,
          credit: 250000,
          description: "Bank",
          baseAmount: 250000,
        },
      ]);
    });

    it("returns default two empty lines when input is empty", () => {
      const lines = apiToPostingLines([]);
      expect(lines.length).toBe(2);
      expect(lines[0]?.accountId).toBeUndefined();
      expect(lines[1]?.accountId).toBeUndefined();
    });
  });

  describe("calculatePostingsSummary", () => {
    it("reports balanced when debit equals credit and at least 2 valid lines", () => {
      const lines: PostingLineValue[] = [
        { accountId: 1, debit: 100000, credit: 0 },
        { accountId: 2, debit: 0, credit: 100000 },
      ];

      const summary = calculatePostingsSummary(lines, "IDR");
      expect(summary.totalDebit).toBe(100000);
      expect(summary.totalCredit).toBe(100000);
      expect(summary.balanced).toBe(true);
      expect(summary.validLineCount).toBe(2);
    });

    it("reports unbalanced when debit differs from credit", () => {
      const lines: PostingLineValue[] = [
        { accountId: 1, debit: 100000, credit: 0 },
        { accountId: 2, debit: 0, credit: 80000 },
      ];

      const summary = calculatePostingsSummary(lines, "IDR");
      expect(summary.totalDebit).toBe(100000);
      expect(summary.totalCredit).toBe(80000);
      expect(summary.balanced).toBe(false);
    });

    it("reports unbalanced when fewer than 2 valid lines even if amounts balance to zero", () => {
      const summary = calculatePostingsSummary([], "IDR");
      expect(summary.balanced).toBe(false);
      expect(summary.validLineCount).toBe(0);
    });

    it("correctly calculates base totals with foreign currency and exchange rate", () => {
      const lines: PostingLineValue[] = [
        { accountId: 1, debit: 50, credit: 0 },
        { accountId: 2, debit: 0, credit: 50 },
      ];

      const summary = calculatePostingsSummary(lines, "USD", 16000);
      expect(summary.totalDebit).toBe(50);
      expect(summary.totalCredit).toBe(50);
      expect(summary.totalBaseDebit).toBe(800000);
      expect(summary.totalBaseCredit).toBe(800000);
      expect(summary.balanced).toBe(true);
    });
  });
});
