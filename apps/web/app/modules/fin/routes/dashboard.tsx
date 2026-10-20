import { RocketOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Button, Progress, Statistic, Table, type TableProps } from "antd";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { formatDate, formatMoney } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { EntryStatusTag } from "../components/entry-status-tag";
import { RevenueChart, type RevenuePoint } from "../components/revenue-chart";
import {
  accountsQueryOptions,
  financialSummaryQueryOptions,
  recentEntriesQueryOptions,
} from "../queries";
import type { Route } from "./+types/dashboard";

type Account = components["schemas"]["model.AccountResponse"];
type Entry = components["schemas"]["model.EntryResponse"];

const types = ["ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE"] as const;

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(financialSummaryQueryOptions(wid, 6)),
    queryClient.ensureQueryData(recentEntriesQueryOptions(wid, 5)),
  ]);
  return { wid };
}

export default function FinDashboard({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: summary } = useSuspenseQuery(
    financialSummaryQueryOptions(wid, 6),
  );
  const { data: recent = [] } = useSuspenseQuery(
    recentEntriesQueryOptions(wid, 5),
  );

  const baseCurrency = summary?.baseCurrency || "IDR";
  const money = (value: number) =>
    formatMoney(value, baseCurrency, i18n.language);

  const totals = summary?.totals ?? {
    assets: 0,
    liabilities: 0,
    equity: 0,
    revenue: 0,
    expense: 0,
    netIncome: 0,
  };
  const series: RevenuePoint[] = (summary?.monthly ?? []).map((m) => ({
    month: m.month ?? "",
    revenue: m.revenue ?? 0,
    expense: m.expense ?? 0,
  }));

  const countByType = (type: string) =>
    accounts.filter((a: Account) => a.type === type).length;

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
          (sum, p) => (p.direction === "DEBIT" ? sum + (p.amount ?? 0) : sum),
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
    <Page title={t("fin.menu.dashboard")}>
      {accounts.length === 0 && (
        <div
          className="surface flex flex-wrap items-center justify-between gap-4 p-4"
          style={{
            borderRadius: "var(--omed-radius-lg)",
            borderLeft: "4px solid var(--omed-primary)",
          }}
        >
          <div className="flex items-center gap-3">
            <RocketOutlined
              style={{ fontSize: 24, color: "var(--omed-primary)" }}
            />
            <div>
              <div
                className="font-semibold"
                style={{ color: "var(--omed-text)" }}
              >
                {t("fin.dashboard.setupBannerTitle")}
              </div>
              <div
                className="text-xs"
                style={{ color: "var(--omed-text-secondary)" }}
              >
                {t("fin.dashboard.setupBannerDesc")}
              </div>
            </div>
          </div>
          <Link to="/fin/setup">
            <Button type="primary">{t("fin.dashboard.setupAction")}</Button>
          </Link>
        </div>
      )}

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
        <Panel>
          <Statistic
            title={t("fin.dashboard.accounts")}
            value={accounts.length}
          />
        </Panel>
      </div>

      <div className="flex flex-wrap items-start gap-3">
        <Panel
          strong
          title={t("fin.dashboard.revenueVsExpense")}
          className="flex-[3_1_460px]"
        >
          <RevenueChart data={series} />
        </Panel>

        <Panel
          strong
          title={t("fin.dashboard.accountsByType")}
          className="flex-[2_1_280px]"
        >
          <div className="flex flex-col gap-3">
            {types.map((type) => (
              <div key={type}>
                <div className="flex justify-between text-[13px]">
                  <span>{t(`fin.types.${type}`)}</span>
                  <span className="num">{countByType(type)}</span>
                </div>
                <Progress
                  percent={
                    accounts.length
                      ? Math.round((countByType(type) / accounts.length) * 100)
                      : 0
                  }
                  showInfo={false}
                  size="small"
                />
              </div>
            ))}
          </div>
        </Panel>
      </div>

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
