import type { components } from "@omed/openapi";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Statistic, Table, type TableProps } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { formatDate, formatMoney } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { EntryStatusTag } from "../components/entry-status-tag";
import { RevenueChart, type RevenuePoint } from "../components/revenue-chart";
import {
  financialSummaryQueryOptions,
  recentEntriesQueryOptions,
} from "../queries";
import type { Route } from "./+types/dashboard";

type Entry = components["schemas"]["model.EntryResponse"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(financialSummaryQueryOptions(wid, 6)),
    queryClient.ensureQueryData(recentEntriesQueryOptions(wid, 5)),
  ]);
  return { wid };
}

export default function FinDashboard({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();

  const { data: summary } = useSuspenseQuery(
    financialSummaryQueryOptions(wid, 6),
  );
  const { data: recent = [] } = useSuspenseQuery(
    recentEntriesQueryOptions(wid, 5),
  );

  const baseCurrency = summary?.baseCurrency || "IDR";
  const money = (value: number | string) =>
    formatMoney(value, baseCurrency, i18n.language);

  const totals = summary?.totals ?? {
    assets: "0",
    liabilities: "0",
    equity: "0",
    revenue: "0",
    expense: "0",
    netIncome: "0",
  };
  const series: RevenuePoint[] = (summary?.monthly ?? []).map((m) => ({
    month: m.month ?? "",
    revenue:
      typeof m.revenue === "string"
        ? Number.parseFloat(m.revenue) || 0
        : (m.revenue ?? 0),
    expense:
      typeof m.expense === "string"
        ? Number.parseFloat(m.expense) || 0
        : (m.expense ?? 0),
  }));

  const columns: TableProps<Entry>["columns"] = [
    {
      title: t("fin.journal.number"),
      dataIndex: "number",
      render: (num, record) => (
        <span className="font-semibold text-primary">
          {num || `#${record.id}`}
        </span>
      ),
    },
    {
      title: t("common.date"),
      dataIndex: "entryDate",
      render: (val) => formatDate(val, i18n.language),
    },
    {
      title: t("fin.journal.reference"),
      dataIndex: "description",
      render: (text) => text || "-",
    },
    {
      title: t("fin.journal.amount"),
      align: "right",
      render: (_, record) => {
        const totalDebit = (record.postings ?? []).reduce(
          (sum, p) =>
            p.direction === "DEBIT"
              ? sum +
                (typeof p.amount === "string"
                  ? Number.parseFloat(p.amount) || 0
                  : (p.amount ?? 0))
              : sum,
          0,
        );
        return <span className="num">{money(totalDebit)}</span>;
      },
    },
    {
      title: t("common.status"),
      dataIndex: "status",
      render: (st) => <EntryStatusTag status={st} />,
    },
  ];

  return (
    <Page title={t("fin.menu.overview")}>
      <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
        <Panel>
          <Statistic
            title={t("fin.dashboard.assets")}
            value={totals.assets}
            formatter={(v) => money(Number(v))}
          />
        </Panel>
        <Panel>
          <Statistic
            title={t("fin.dashboard.liabilities")}
            value={totals.liabilities}
            formatter={(v) => money(Number(v))}
          />
        </Panel>
        <Panel>
          <Statistic
            title={t("fin.dashboard.netIncome")}
            value={totals.netIncome}
            formatter={(v) => money(Number(v))}
          />
        </Panel>
      </div>

      <Panel strong title={t("fin.dashboard.revenueVsExpense")}>
        <RevenueChart data={series} />
      </Panel>

      <Panel strong title={t("fin.dashboard.recentEntries")}>
        <Table<Entry>
          size="small"
          rowKey="id"
          columns={columns}
          dataSource={recent}
          pagination={false}
          scroll={{ x: "max-content" }}
          onRow={(record) => ({
            onClick: () => navigate(`/fin/journal/${record.id}`),
            style: { cursor: "pointer" },
          })}
        />
      </Panel>
    </Page>
  );
}
