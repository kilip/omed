import type { components } from "@omed/openapi";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CashflowDetail, summarizeCategories } from "./cashflow-detail";

type CashFlowItem = components["schemas"]["model.CashFlowItemResponse"];
type CashFlowLine = components["schemas"]["model.CashFlowLineResponse"];

vi.mock("~/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("~/lib/api")>();
  return {
    ...actual,
    api: {
      GET: vi.fn(),
      POST: vi.fn(),
      DELETE: vi.fn(),
    },
    unwrapOne: vi.fn(async (promise) => {
      const res = await promise;
      return res.data?.data;
    }),
    unwrapVoid: vi.fn(async () => undefined),
  };
});

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, defaultValue?: unknown) =>
        typeof defaultValue === "string" ? defaultValue : key,
      i18n: { language: "en" },
    }),
  };
});

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useRevalidator: () => ({ revalidate: vi.fn(), state: "idle" }),
    Link: ({ to, children }: { to: string; children: React.ReactNode }) => (
      <a href={to}>{children}</a>
    ),
  };
});

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQueryClient: () => ({
      invalidateQueries: vi.fn(),
    }),
    useSuspenseQuery: vi.fn().mockReturnValue({ data: [] }),
  };
});

vi.mock("../hooks", () => ({
  useFinPermissions: () => ({
    canEditTransaction: true,
    canDeleteTransaction: true,
    canPostEntry: true,
  }),
}));

describe("CashflowDetail and summarizeCategories", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("summarizeCategories", () => {
    it("returns first account and more count for owner income items", () => {
      const items: CashFlowLine[] = [
        {
          id: 1,
          account: {
            id: 4130,
            code: "4130",
            name: "Consulting & Advisory Income",
          },
          quantity: "1",
          unitPrice: "1000000.00",
          amount: "1000000.00",
        },
        {
          id: 2,
          account: { id: 4110, code: "4110", name: "Service Revenue" },
          quantity: "6",
          unitPrice: "500000.00",
          amount: "3000000.00",
        },
      ];

      const summary = summarizeCategories(items);
      expect(summary.first).toBe("4130 - Consulting & Advisory Income");
      expect(summary.more).toBe(1);
    });

    it("returns more 1 for 6710 twice (RF1)", () => {
      const items: CashFlowLine[] = [
        {
          id: 10,
          account: { id: 6710, code: "6710", name: "Office Supplies" },
          quantity: "2",
          unitPrice: "55000.00",
          amount: "110000.00",
        },
        {
          id: 11,
          account: { id: 6710, code: "6710", name: "Office Supplies" },
          quantity: "5",
          unitPrice: "3000.00",
          amount: "15000.00",
        },
      ];

      const summary = summarizeCategories(items);
      expect(summary.first).toBe("6710 - Office Supplies");
      expect(summary.more).toBe(1);
    });

    it("returns undefined first and more 0 for empty items", () => {
      const summary = summarizeCategories([]);
      expect(summary.first).toBeUndefined();
      expect(summary.more).toBe(0);
    });
  });

  describe("CashflowDetail actions and rendering", () => {
    const draftItem: CashFlowItem = {
      id: 501,
      number: "TRX-2026-0050",
      transactionDate: "2026-01-20",
      type: "INCOME",
      entryStatus: "DRAFT",
      entryId: 123,
      amount: "4000000.00",
      currency: "IDR",
      description: "Website Project Consulting",
      reference: "inv-123456",
      paymentAccount: {
        id: 21,
        code: "1121",
        name: "Bank Kaltim",
        type: "ASSET",
      },
      items: [
        {
          id: 1,
          account: {
            id: 4130,
            code: "4130",
            name: "Consulting & Advisory Income",
          },
          quantity: "1",
          unitPrice: "1000000.00",
          amount: "1000000.00",
        },
        {
          id: 2,
          account: { id: 4110, code: "4110", name: "Service Revenue" },
          quantity: "6",
          unitPrice: "500000.00",
          amount: "3000000.00",
        },
      ],
      attachments: [],
    };

    const postedItem: CashFlowItem = {
      ...draftItem,
      id: 502,
      number: "TRX-2026-0051",
      entryStatus: "POSTED",
    };

    it("contains Edit, Posting, and Delete when item is DRAFT", () => {
      const html = renderToString(
        <MemoryRouter>
          <CashflowDetail wid={1} flow="INCOME" item={draftItem} />
        </MemoryRouter>,
      );

      expect(html).toContain("Edit");
      expect(html).toContain("Posting");
      expect(html).toContain("Delete");
      expect(html).toContain("TRX-2026-0050");
      expect(html).toContain("inv-123456");
      expect(html).toContain("Bank Kaltim");
    });

    it("contains none of Edit, Posting, or Delete when item is POSTED", () => {
      const html = renderToString(
        <MemoryRouter>
          <CashflowDetail wid={1} flow="INCOME" item={postedItem} />
        </MemoryRouter>,
      );

      expect(html).not.toContain("Edit");
      expect(html).not.toContain("Posting");
      expect(html).not.toContain("Delete");
      expect(html).toContain("TRX-2026-0051");
    });
  });
});
