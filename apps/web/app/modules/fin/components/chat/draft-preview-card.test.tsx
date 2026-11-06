import type React from "react";
import { renderToString } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DraftProposal } from "../../hooks/use-agent-chat";
import { DraftPreviewCard } from "./draft-preview-card";

interface MockButtonProps {
  children?: React.ReactNode;
  disabled?: boolean;
  danger?: boolean;
  onClick?: () => void;
  "aria-label"?: string;
  loading?: boolean;
}

interface MockPopconfirmProps {
  children?: React.ReactNode;
  title?: React.ReactNode;
  onConfirm?: () => void;
}

interface MockSelectProps {
  value?: unknown;
  onChange?: (val: unknown) => void;
  options?: { value: unknown; label: React.ReactNode }[];
  "aria-label"?: string;
}

interface MockInputProps {
  value?: string;
  onChange?: (e: { target: { value: string } }) => void;
  "aria-label"?: string;
}

interface MockInputNumberProps {
  value?: number;
  onChange?: (val: number | null) => void;
}

let buttonPropsList: MockButtonProps[] = [];
let popconfirmPropsList: MockPopconfirmProps[] = [];
let selectPropsList: MockSelectProps[] = [];
let inputPropsList: MockInputProps[] = [];
let inputNumberPropsList: MockInputNumberProps[] = [];

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>();
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, defaultValue?: unknown) => {
        if (typeof defaultValue === "string") return defaultValue;
        if (
          typeof defaultValue === "object" &&
          defaultValue !== null &&
          "defaultValue" in defaultValue
        ) {
          const opts = defaultValue as { defaultValue?: unknown };
          if (typeof opts.defaultValue === "string") {
            return opts.defaultValue;
          }
        }
        return key;
      },
      i18n: { language: "en" },
    }),
  };
});

vi.mock("react-router", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router")>();
  return {
    ...actual,
    Link: ({
      to,
      children,
      className,
    }: {
      to: string;
      children: React.ReactNode;
      className?: string;
    }) => (
      <a href={to} className={className}>
        {children}
      </a>
    ),
  };
});

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  return {
    ...actual,
    Button: (props: MockButtonProps) => {
      buttonPropsList.push(props);
      return (
        <button
          type="button"
          disabled={props.disabled}
          onClick={props.onClick}
          data-danger={props.danger}
          aria-label={props["aria-label"]}
        >
          {props.children}
        </button>
      );
    },
    Popconfirm: (props: MockPopconfirmProps) => {
      popconfirmPropsList.push(props);
      return <div data-testid="popconfirm">{props.children}</div>;
    },
    Select: (props: MockSelectProps) => {
      selectPropsList.push(props);
      return (
        <select
          value={String(props.value ?? "")}
          onChange={(e) => props.onChange?.(e.target.value)}
          aria-label={props["aria-label"]}
        >
          {props.options?.map((opt) => (
            <option key={String(opt.value)} value={String(opt.value)}>
              {opt.label}
            </option>
          ))}
        </select>
      );
    },
    Input: (props: MockInputProps) => {
      inputPropsList.push(props);
      return (
        <input
          value={props.value ?? ""}
          onChange={props.onChange}
          aria-label={props["aria-label"]}
        />
      );
    },
    InputNumber: (props: MockInputNumberProps) => {
      inputNumberPropsList.push(props);
      return (
        <input
          type="number"
          value={props.value ?? 0}
          onChange={(e) => props.onChange?.(Number(e.target.value))}
        />
      );
    },
    Tag: ({
      children,
      color,
      className,
    }: {
      children: React.ReactNode;
      color?: string;
      className?: string;
    }) => (
      <span className={`ant-tag ${color || ""} ${className || ""}`}>
        {children}
      </span>
    ),
  };
});

describe("DraftPreviewCard", () => {
  beforeEach(() => {
    buttonPropsList = [];
    popconfirmPropsList = [];
    selectPropsList = [];
    inputPropsList = [];
    inputNumberPropsList = [];
  });

  const mockAccounts = [
    {
      id: 101,
      code: "101",
      name: "Cash on Hand",
      isPostable: true,
      isActive: true,
    },
    {
      id: 102,
      code: "102",
      name: "Bank Account",
      isPostable: true,
      isActive: true,
    },
    {
      id: 201,
      code: "201",
      name: "Office Supplies",
      isPostable: true,
      isActive: true,
    },
    {
      id: 202,
      code: "202",
      name: "Stationery",
      isPostable: true,
      isActive: true,
    },
    {
      id: 401,
      code: "401",
      name: "Sales Revenue",
      isPostable: true,
      isActive: true,
    },
  ];

  describe("Expense preview card rendering", () => {
    it("renders expense draft with multiple items and calculated total", () => {
      const expenseProposal: DraftProposal = {
        kind: "expense",
        transactionDate: "2026-03-01",
        reference: "RCP-001",
        description: "Store supplies purchase",
        paymentAccountId: 101,
        currency: "IDR",
        totalAmount: "70000.00",
        items: [
          {
            accountId: 201,
            description: "Printer paper",
            quantity: "2",
            unitPrice: "25000.00",
            amount: "50000.00",
          },
          {
            accountId: 202,
            description: "Pens and markers",
            quantity: "1",
            unitPrice: "20000.00",
            amount: "20000.00",
          },
        ],
      };

      const html = renderToString(
        <DraftPreviewCard
          proposal={expenseProposal}
          status="proposed"
          accounts={mockAccounts}
        />,
      );

      expect(html).toContain("Expense Draft");
      expect(html).toContain("RCP-001");
      expect(html).toContain("Store supplies purchase");
      expect(html).toContain("101 - Cash on Hand");
      expect(html).toContain("Printer paper");
      expect(html).toContain("Pens and markers");
      expect(html).toContain("201 - Office Supplies");
      expect(html).toContain("202 - Stationery");
    });
  });

  describe("Income preview card rendering", () => {
    it("renders income draft with payment account and line items", () => {
      const incomeProposal: DraftProposal = {
        kind: "income",
        transactionDate: "2026-03-02",
        reference: "INV-999",
        description: "Client consulting service",
        paymentAccountId: 102,
        currency: "IDR",
        totalAmount: "5000000.00",
        items: [
          {
            accountId: 401,
            description: "Consulting fee",
            quantity: "1",
            unitPrice: "5000000.00",
            amount: "5000000.00",
          },
        ],
      };

      const html = renderToString(
        <DraftPreviewCard
          proposal={incomeProposal}
          status="proposed"
          accounts={mockAccounts}
        />,
      );

      expect(html).toContain("Income Draft");
      expect(html).toContain("INV-999");
      expect(html).toContain("Client consulting service");
      expect(html).toContain("102 - Bank Account");
      expect(html).toContain("Consulting fee");
      expect(html).toContain("401 - Sales Revenue");
    });
  });

  describe("Journal preview card rendering", () => {
    it("renders journal draft with balanced Dr and Cr entries", () => {
      const journalProposal: DraftProposal = {
        kind: "journal",
        transactionDate: "2026-03-03",
        reference: "ATM-001",
        description: "ATM Cash withdrawal from Bank",
        currency: "IDR",
        totalAmount: "2000000.00",
        entries: [
          {
            accountId: 101,
            direction: "DEBIT",
            amount: "2000000.00",
            description: "Cash on Hand",
          },
          {
            accountId: 102,
            direction: "CREDIT",
            amount: "2000000.00",
            description: "Bank Account",
          },
        ],
      };

      const html = renderToString(
        <DraftPreviewCard
          proposal={journalProposal}
          status="proposed"
          accounts={mockAccounts}
        />,
      );

      expect(html).toContain("Journal Draft");
      expect(html).toContain("ATM-001");
      expect(html).toContain("ATM Cash withdrawal from Bank");
      expect(html).toContain("101 - Cash on Hand");
      expect(html).toContain("102 - Bank Account");
      expect(html).toContain("Debit");
      expect(html).toContain("Credit");
      expect(html).toContain("Balanced");
    });
  });

  describe("Confirmed and discarded states", () => {
    it("renders confirmed state with link to transaction and hides action buttons", () => {
      const proposal: DraftProposal = {
        kind: "expense",
        transactionDate: "2026-03-01",
        totalAmount: "50000.00",
      };

      const html = renderToString(
        <DraftPreviewCard
          proposal={proposal}
          status="confirmed"
          transactionId={123}
          accounts={mockAccounts}
        />,
      );

      expect(html).toContain("Confirmed");
      expect(html).toContain("/fin/transactions/123");
      expect(html).toContain("View Transaction");

      // Edit, Confirm, Discard buttons should not be present
      const editBtn = buttonPropsList.find((b) =>
        String(b.children).includes("Edit"),
      );
      const confirmBtn = buttonPropsList.find((b) =>
        String(b.children).includes("Confirm"),
      );
      expect(editBtn).toBeUndefined();
      expect(confirmBtn).toBeUndefined();
    });

    it("renders discarded state with muted tag and hides action buttons", () => {
      const proposal: DraftProposal = {
        kind: "expense",
        transactionDate: "2026-03-01",
        totalAmount: "50000.00",
      };

      const html = renderToString(
        <DraftPreviewCard
          proposal={proposal}
          status="discarded"
          accounts={mockAccounts}
        />,
      );

      expect(html).toContain("Discarded");

      const editBtn = buttonPropsList.find((b) =>
        String(b.children).includes("Edit"),
      );
      const confirmBtn = buttonPropsList.find((b) =>
        String(b.children).includes("Confirm"),
      );
      expect(editBtn).toBeUndefined();
      expect(confirmBtn).toBeUndefined();
    });
  });

  describe("Actions and mutations", () => {
    it("triggers onConfirm when Confirm button is clicked", () => {
      const onConfirm = vi.fn();
      const proposal: DraftProposal = {
        kind: "expense",
        transactionDate: "2026-03-01",
        totalAmount: "70000.00",
      };

      renderToString(
        <DraftPreviewCard
          proposal={proposal}
          status="proposed"
          onConfirm={onConfirm}
          accounts={mockAccounts}
        />,
      );

      const confirmBtn = buttonPropsList.find((b) =>
        String(b.children).includes("Confirm"),
      );
      expect(confirmBtn).toBeDefined();

      confirmBtn?.onClick?.();
      expect(onConfirm).toHaveBeenCalledTimes(1);
    });

    it("triggers onDiscard when Discard Popconfirm confirms", () => {
      const onDiscard = vi.fn();
      const proposal: DraftProposal = {
        kind: "expense",
        transactionDate: "2026-03-01",
        totalAmount: "70000.00",
      };

      renderToString(
        <DraftPreviewCard
          proposal={proposal}
          status="proposed"
          onDiscard={onDiscard}
          accounts={mockAccounts}
        />,
      );

      expect(popconfirmPropsList.length).toBeGreaterThan(0);
      const popconfirm = popconfirmPropsList[0];
      expect(popconfirm).toBeDefined();

      popconfirm.onConfirm?.();
      expect(onDiscard).toHaveBeenCalledTimes(1);
    });

    it("toggles into inline edit mode when Edit button is clicked", () => {
      const proposal: DraftProposal = {
        kind: "expense",
        transactionDate: "2026-03-01",
        description: "Office supplies",
        totalAmount: "70000.00",
        items: [
          {
            accountId: 201,
            description: "Paper",
            quantity: "1",
            unitPrice: "70000.00",
            amount: "70000.00",
          },
        ],
      };

      renderToString(
        <DraftPreviewCard
          proposal={proposal}
          status="proposed"
          accounts={mockAccounts}
        />,
      );

      const editBtn = buttonPropsList.find((b) =>
        String(b.children).includes("Edit"),
      );
      expect(editBtn).toBeDefined();

      // Trigger edit mode
      editBtn?.onClick?.();
    });
  });
});
