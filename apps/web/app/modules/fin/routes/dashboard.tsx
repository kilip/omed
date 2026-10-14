import type { components } from "@omed/openapi";
import { Progress, Statistic, Table, type TableProps, Tag } from "antd";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapList } from "~/lib/api";
import { formatDate, formatIDR } from "~/lib/format";
import { requireWorkspaceId } from "~/lib/workspace";
import { RevenueChart } from "../components/revenue-chart";
import {
  type JournalEntry,
  mockEntries,
  monthlySeries,
  sumDebit,
  summary,
} from "../mock";
import type { Route } from "./+types/dashboard";

type Account = components["schemas"]["model.AccountResponse"];

const types = ["ASSET", "LIABILITY", "EQUITY", "REVENUE", "EXPENSE"] as const;

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const accounts = await unwrapList(
    api.POST("/workspaces/{wid}/fin/accounts/search", {
      params: { path: { wid } },
      body: {},
    }),
  );
  return { accounts };
}

export default function FinDashboard({ loaderData }: Route.ComponentProps) {
  const { accounts } = loaderData;
  const { t, i18n } = useTranslation();
  const money = (value: number) => formatIDR(value, i18n.language);

  const totals = summary(mockEntries);
  const series = monthlySeries(mockEntries);
  const recent = [...mockEntries]
    .sort((a, b) => b.date.localeCompare(a.date) || b.id - a.id)
    .slice(0, 5);

  const countByType = (type: string) =>
    accounts.filter((a: Account) => a.type === type).length;

  const provisional = <Tag variant="filled">{t("common.provisional")}</Tag>;

  const columns: TableProps<JournalEntry>["columns"] = [
    { title: t("fin.journal.number"), dataIndex: "number" },
    {
      title: t("common.date"),
      render: (_, e) => formatDate(e.date, i18n.language),
    },
    { title: t("fin.journal.reference"), dataIndex: "reference" },
    {
      title: t("fin.journal.amount"),
      align: "right",
      render: (_, e) => <span className="num">{money(sumDebit(e.lines))}</span>,
    },
    {
      title: t("common.status"),
      render: (_, e) => (
        <Tag color={e.status === "POSTED" ? "success" : "default"}>
          {t(`fin.journal.${e.status}`)}
        </Tag>
      ),
    },
  ];

  return (
    <Page title={t("fin.menu.overview")}>
      <div className="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
        <Panel extra={provisional}>
          <Statistic
            title={t("fin.dashboard.assets")}
            value={totals.assets}
            formatter={(v) => money(Number(v))}
          />
        </Panel>
        <Panel extra={provisional}>
          <Statistic
            title={t("fin.dashboard.liabilities")}
            value={totals.liabilities}
            formatter={(v) => money(Number(v))}
          />
        </Panel>
        <Panel extra={provisional}>
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
          extra={provisional}
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

      <Panel
        strong
        title={t("fin.dashboard.recentEntries")}
        extra={provisional}
      >
        <Table
          size="small"
          rowKey="id"
          columns={columns}
          dataSource={recent}
          pagination={false}
          scroll={{ x: "max-content" }}
        />
      </Panel>
    </Page>
  );
}
