import { Alert, Table, type TableProps, Tag } from "antd";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { formatDate, formatIDR } from "~/lib/format";
import {
  type AccountType,
  mockEntries,
  PERIOD_END,
  type TrialBalanceRow,
  trialBalance,
} from "../mock";

const order: AccountType[] = [
  "ASSET",
  "LIABILITY",
  "EQUITY",
  "REVENUE",
  "EXPENSE",
];

type Row =
  | { key: string; kind: "group"; type: AccountType }
  | { key: string; kind: "account"; row: TrialBalanceRow };

export default function TrialBalance() {
  const { t, i18n } = useTranslation();
  const money = (value: number) => formatIDR(value, i18n.language);

  const rows = trialBalance(mockEntries);
  const totalDebit = rows.reduce((sum, r) => sum + r.debit, 0);
  const totalCredit = rows.reduce((sum, r) => sum + r.credit, 0);
  const balanced = totalDebit === totalCredit;

  const data: Row[] = order.flatMap((type) => [
    { key: `g-${type}`, kind: "group" as const, type },
    ...rows
      .filter((r) => r.account.type === type)
      .map((row) => ({
        key: row.account.code,
        kind: "account" as const,
        row,
      })),
  ]);

  const amount = (value: number) =>
    value ? <span className="num">{money(value)}</span> : "";

  const columns: TableProps<Row>["columns"] = [
    {
      title: t("fin.accounts.code"),
      width: 120,
      onCell: (r) => ({ colSpan: r.kind === "group" ? 4 : 1 }),
      render: (_, r) =>
        r.kind === "group" ? (
          <strong>{t(`fin.types.${r.type}`)}</strong>
        ) : (
          <code>{r.row.account.code}</code>
        ),
    },
    {
      title: t("common.name"),
      onCell: (r) => ({ colSpan: r.kind === "group" ? 0 : 1 }),
      render: (_, r) => (r.kind === "account" ? r.row.account.name : null),
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
      <Alert type="info" showIcon message={t("common.provisionalNotice")} />

      <div className="flex flex-wrap items-center gap-3">
        <span style={{ color: "var(--omed-text-secondary)" }}>
          {t("fin.trial.asOf", {
            date: formatDate(PERIOD_END, i18n.language),
          })}
        </span>
        <Tag color={balanced ? "success" : "error"}>
          {balanced ? t("fin.journal.balanced") : t("fin.journal.unbalanced")}
        </Tag>
      </div>

      <Panel strong>
        <Table<Row>
          size="small"
          rowKey="key"
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
