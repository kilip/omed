import { Alert, Select, Table, type TableProps } from "antd";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { formatDate, formatIDR } from "~/lib/format";
import {
  type LedgerRow,
  ledgerFor,
  mockAccounts,
  mockEntries,
  PERIOD_END,
  PERIOD_START,
} from "../mock";

export default function GeneralLedger() {
  const { t, i18n } = useTranslation();
  const [params, setParams] = useSearchParams();
  const code = params.get("account") ?? "1120";

  const money = (value: number) => formatIDR(value, i18n.language);
  const ledger = ledgerFor(code, mockEntries);

  const columns: TableProps<LedgerRow>["columns"] = [
    {
      title: t("common.date"),
      render: (_, r) => formatDate(r.date, i18n.language),
    },
    { title: t("fin.journal.number"), dataIndex: "number" },
    { title: t("fin.journal.label"), dataIndex: "label" },
    {
      title: t("fin.journal.debit"),
      align: "right",
      render: (_, r) =>
        r.debit ? <span className="num">{money(r.debit)}</span> : "",
    },
    {
      title: t("fin.journal.credit"),
      align: "right",
      render: (_, r) =>
        r.credit ? <span className="num">{money(r.credit)}</span> : "",
    },
    {
      title: t("fin.ledger.balance"),
      align: "right",
      render: (_, r) => <span className="num">{money(r.balance)}</span>,
    },
  ];

  const totalDebit = ledger.rows.reduce((sum, r) => sum + r.debit, 0);
  const totalCredit = ledger.rows.reduce((sum, r) => sum + r.credit, 0);

  return (
    <Page title={t("fin.menu.ledger")}>
      <Alert type="info" showIcon message={t("common.provisionalNotice")} />

      <div className="flex flex-wrap items-center gap-3">
        <Select
          showSearch
          optionFilterProp="label"
          aria-label={t("fin.journal.account")}
          style={{ minWidth: 260 }}
          value={code}
          options={mockAccounts.map((a) => ({
            value: a.code,
            label: `${a.code} ${a.name}`,
          }))}
          onChange={(value) => setParams({ account: value })}
        />
        <span style={{ color: "var(--omed-text-secondary)" }}>
          {formatDate(PERIOD_START, i18n.language)} –{" "}
          {formatDate(PERIOD_END, i18n.language)}
        </span>
      </div>

      <Panel strong>
        <Table<LedgerRow>
          size="small"
          rowKey={(r) => `${r.number}-${r.label}`}
          columns={columns}
          dataSource={ledger.rows}
          pagination={false}
          scroll={{ x: "max-content" }}
          title={() => (
            <div className="flex justify-between text-[13px]">
              <span>{t("fin.ledger.opening")}</span>
              <strong className="num">{money(ledger.opening)}</strong>
            </div>
          )}
          summary={() => (
            <>
              <Table.Summary.Row>
                <Table.Summary.Cell index={0} colSpan={3}>
                  <strong>{t("common.total")}</strong>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={3} align="right">
                  <strong className="num">{money(totalDebit)}</strong>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={4} align="right">
                  <strong className="num">{money(totalCredit)}</strong>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={5} align="right">
                  <strong className="num">{money(ledger.closing)}</strong>
                </Table.Summary.Cell>
              </Table.Summary.Row>
            </>
          )}
        />
      </Panel>
    </Page>
  );
}
