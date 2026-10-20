import type { components } from "@omed/openapi";
import { useQuery } from "@tanstack/react-query";
import { DatePicker, Table, type TableProps, Tag } from "antd";
import dayjs from "dayjs";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { formatMoney } from "~/lib/format";
import { requireWorkspaceId } from "~/lib/workspace";
import { trialBalanceQueryOptions } from "../queries";
import type { Route } from "./+types/trial-balance";

type AccountItem = components["schemas"]["model.TrialBalanceAccountItem"];

export type AccountType =
  | "ASSET"
  | "LIABILITY"
  | "EQUITY"
  | "REVENUE"
  | "EXPENSE";

const order: AccountType[] = [
  "ASSET",
  "LIABILITY",
  "EQUITY",
  "REVENUE",
  "EXPENSE",
];

type Row =
  | { key: string; kind: "group"; type: AccountType }
  | { key: string; kind: "account"; row: AccountItem };

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  return { wid };
}

export default function TrialBalance({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const [params, setParams] = useSearchParams();

  const asOfParam = params.get("asOf") || dayjs().format("YYYY-MM-DD");

  const { data: tb, isLoading } = useQuery(
    trialBalanceQueryOptions(wid, asOfParam),
  );

  const accounts = tb?.accounts ?? [];
  const totalDebit = tb?.totalDebit ?? 0;
  const totalCredit = tb?.totalCredit ?? 0;
  const balanced = tb?.isBalanced ?? false;
  const baseCurrency = tb?.baseCurrency ?? "IDR";

  const money = (value: number) =>
    formatMoney(value, baseCurrency, i18n.language);

  const data: Row[] = order.flatMap((type) => [
    { key: `g-${type}`, kind: "group" as const, type },
    ...accounts
      .filter((r) => r.type === type)
      .map((row) => ({
        key: row.accountCode || String(row.accountId),
        kind: "account" as const,
        row,
      })),
  ]);

  const amount = (value?: number) =>
    value ? <span className="num">{money(value)}</span> : "";

  const handleDateChange = (date: dayjs.Dayjs | null) => {
    if (!date) return;
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set("asOf", date.format("YYYY-MM-DD"));
        return next;
      },
      { replace: true },
    );
  };

  const columns: TableProps<Row>["columns"] = [
    {
      title: t("fin.accounts.code"),
      width: 120,
      onCell: (r) => ({ colSpan: r.kind === "group" ? 4 : 1 }),
      render: (_, r) =>
        r.kind === "group" ? (
          <strong>{t(`fin.types.${r.type}`)}</strong>
        ) : (
          <code>{r.row.accountCode}</code>
        ),
    },
    {
      title: t("common.name"),
      onCell: (r) => ({ colSpan: r.kind === "group" ? 0 : 1 }),
      render: (_, r) => (r.kind === "account" ? r.row.accountName : null),
    },
    {
      title: t("fin.journal.debit"),
      align: "right",
      onCell: (r) => ({ colSpan: r.kind === "group" ? 0 : 1 }),
      render: (_, r) => (r.kind === "account" ? amount(r.row.debit) : null),
    },
    {
      title: t("fin.journal.credit"),
      align: "right",
      onCell: (r) => ({ colSpan: r.kind === "group" ? 0 : 1 }),
      render: (_, r) => (r.kind === "account" ? amount(r.row.credit) : null),
    },
  ];

  return (
    <Page title={t("fin.menu.trialBalance")}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <span style={{ color: "var(--omed-text-secondary)" }}>
            {t("fin.trial.asOfLabel")}:
          </span>
          <DatePicker
            value={dayjs(asOfParam)}
            onChange={handleDateChange}
            allowClear={false}
          />
        </div>
        <Tag color={balanced ? "success" : "error"}>
          {balanced ? t("fin.journal.balanced") : t("fin.journal.unbalanced")}
        </Tag>
      </div>

      <Panel strong>
        <Table<Row>
          size="small"
          rowKey="key"
          loading={isLoading}
          columns={columns}
          dataSource={data}
          pagination={false}
          scroll={{ x: "max-content" }}
          summary={() => (
            <Table.Summary.Row>
              <Table.Summary.Cell index={0} colSpan={2}>
                <strong>{t("common.total")}</strong>
              </Table.Summary.Cell>
              <Table.Summary.Cell index={2} align="right">
                <strong className="num">{money(totalDebit)}</strong>
              </Table.Summary.Cell>
              <Table.Summary.Cell index={3} align="right">
                <strong className="num">{money(totalCredit)}</strong>
              </Table.Summary.Cell>
            </Table.Summary.Row>
          )}
        />
      </Panel>
    </Page>
  );
}
