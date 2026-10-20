import { PlusOutlined } from "@ant-design/icons";
import type { components, paths } from "@omed/openapi";
import { useInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import { Button, Empty, type MenuProps, Table, type TableProps } from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import type { SearchFacet } from "~/components/SearchBar";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import { EntryStatusTag, MoneyCell } from "../components";
import { useFinPermissions } from "../hooks";
import {
  accountsQueryOptions,
  entriesInfiniteQueryOptions,
  periodsQueryOptions,
  transactionsInfiniteQueryOptions,
} from "../queries";
import type { Route } from "./+types/journal";

type Entry = components["schemas"]["model.EntryResponse"];
type SearchFilters = NonNullable<
  paths["/workspaces/{wid}/fin/entries/search"]["post"]["requestBody"]
>["content"]["application/json"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureInfiniteQueryData(entriesInfiniteQueryOptions(wid, {})),
    queryClient.ensureInfiniteQueryData(
      transactionsInfiniteQueryOptions(wid, {}),
    ),
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(periodsQueryOptions(wid)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid };
}

export default function JournalRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { canCreateEntry } = useFinPermissions(wid);

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: periods = [] } = useSuspenseQuery(periodsQueryOptions(wid));

  const [queryText, setQueryText] = useState("");
  const [statusFilter, setStatusFilter] = useState<
    SearchFilters["status"] | undefined
  >();
  const [periodFilter, setPeriodFilter] = useState<number | undefined>();
  const [accountFilter, setAccountFilter] = useState<number | undefined>();
  const [dateRange, setDateRange] = useState<[string, string] | undefined>();

  const filters = useMemo<SearchFilters>(() => {
    return {
      status: statusFilter,
      periodId: periodFilter,
      accountId: accountFilter,
      dateFrom: dateRange?.[0],
      dateTo: dateRange?.[1],
    };
  }, [statusFilter, periodFilter, accountFilter, dateRange]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteQuery(entriesInfiniteQueryOptions(wid, filters));

  const entries = useMemo(() => {
    const list = data?.pages.flatMap((page) => page.items) ?? [];
    if (!queryText.trim()) return list;
    const lower = queryText.toLowerCase().trim();
    return list.filter(
      (e) =>
        e.number?.toLowerCase().includes(lower) ||
        e.description?.toLowerCase().includes(lower),
    );
  }, [data, queryText]);

  // Compute total debit/credit amount for each entry from its postings
  const entryAmounts = useMemo(() => {
    const map = new Map<number, number>();
    for (const entry of entries) {
      if (!entry.id) continue;
      const totalDebit = (entry.postings ?? []).reduce(
        (sum, p) => (p.direction === "DEBIT" ? sum + (p.amount ?? 0) : sum),
        0,
      );
      map.set(entry.id, totalDebit);
    }
    return map;
  }, [entries]);

  const facets = useMemo<SearchFacet[]>(() => {
    const items: SearchFacet[] = [];
    if (statusFilter) {
      items.push({
        key: `status-${statusFilter}`,
        label: t("fin.journal.filterStatus", {
          status: t(`fin.journal.${statusFilter}`, statusFilter),
        }),
        onRemove: () => setStatusFilter(undefined),
      });
    }
    if (periodFilter) {
      const p = periods.find((item) => item.id === periodFilter);
      items.push({
        key: `period-${periodFilter}`,
        label: t("fin.journal.filterPeriod", {
          period: p?.name ?? String(periodFilter),
        }),
        onRemove: () => setPeriodFilter(undefined),
      });
    }
    if (accountFilter) {
      const a = accounts.find((item) => item.id === accountFilter);
      items.push({
        key: `account-${accountFilter}`,
        label: t("fin.journal.filterAccount", {
          account: a ? `${a.code} - ${a.name}` : String(accountFilter),
        }),
        onRemove: () => setAccountFilter(undefined),
      });
    }
    if (dateRange) {
      items.push({
        key: `date-${dateRange[0]}-${dateRange[1]}`,
        label: t("fin.journal.filterDate", {
          range: `${dateRange[0]} ~ ${dateRange[1]}`,
        }),
        onRemove: () => setDateRange(undefined),
      });
    }
    return items;
  }, [
    statusFilter,
    periodFilter,
    accountFilter,
    dateRange,
    periods,
    accounts,
    t,
  ]);

  const filterMenuItems: MenuProps["items"] = [
    {
      key: "status-group",
      label: t("fin.journal.status", "Status"),
      type: "group",
      children: [
        { key: "status:DRAFT", label: t("fin.journal.DRAFT", "Draft") },
        { key: "status:POSTED", label: t("fin.journal.POSTED", "Posted") },
        { key: "status:VOID", label: t("fin.journal.VOID", "Void") },
      ],
    },
    {
      key: "period-group",
      label: t("fin.journal.period", "Period"),
      type: "group",
      children: periods.slice(0, 5).map((p) => ({
        key: `period:${p.id}`,
        label: p.name,
      })),
    },
  ];

  const handleFilterSelect = (key: string) => {
    if (key.startsWith("status:")) {
      const val = key.replace("status:", "") as SearchFilters["status"];
      setStatusFilter((curr) => (curr === val ? undefined : val));
    } else if (key.startsWith("period:")) {
      const val = Number(key.replace("period:", ""));
      setPeriodFilter((curr) => (curr === val ? undefined : val));
    }
  };

  const columns: TableProps<Entry>["columns"] = [
    {
      title: t("fin.journal.number", "Number"),
      dataIndex: "number",
      key: "number",
      render: (num, record) => (
        <span className="font-semibold text-primary">
          {num || `#${record.id}`}
        </span>
      ),
    },
    {
      title: t("common.date", "Date"),
      dataIndex: "entryDate",
      key: "entryDate",
      render: (val) => formatDate(val, i18n.language),
    },
    {
      title: t("fin.journal.label", "Description"),
      dataIndex: "description",
      key: "description",
      render: (text) => text || "-",
    },
    {
      title: t("fin.journal.amount", "Amount"),
      key: "amount",
      align: "right",
      render: (_, record) => {
        const amt = record.id ? (entryAmounts.get(record.id) ?? 0) : 0;
        return <MoneyCell amount={amt} currency="IDR" />;
      },
    },
    {
      title: t("common.status", "Status"),
      dataIndex: "status",
      key: "status",
      render: (st) => <EntryStatusTag status={st} />,
    },
    {
      title: t("fin.journal.transaction", "Transaction"),
      key: "transaction",
      render: (_, record) => {
        if (!record.transactionId) return "-";
        return (
          <Link
            to={`/fin/transactions/${record.transactionId}`}
            onClick={(e) => e.stopPropagation()}
            className="text-xs text-primary hover:underline"
          >
            #{record.transactionId}
          </Link>
        );
      },
    },
  ];

  return (
    <Page
      title={t("fin.menu.journal", "Journal Entries")}
      search={{
        value: queryText,
        onChange: setQueryText,
        placeholder: t(
          "fin.journal.searchPlaceholder",
          "Search description or number...",
        ),
        facets,
        filterItems: filterMenuItems,
        onFilterSelect: handleFilterSelect,
      }}
      actions={
        canCreateEntry ? (
          <div className="flex gap-2">
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => navigate("/fin/journal/new")}
            >
              {t("fin.journal.new", "New entry")}
            </Button>
          </div>
        ) : undefined
      }
      count={entries.length}
    >
      <Panel strong>
        {entries.length === 0 ? (
          <Empty
            description={t("fin.journal.emptyDesc", "No journal entries found")}
            className="py-12"
          />
        ) : (
          <>
            <Table<Entry>
              size="small"
              rowKey="id"
              columns={columns}
              dataSource={entries}
              pagination={false}
              scroll={{ x: "max-content" }}
              onRow={(record) => ({
                onClick: () => navigate(`/fin/journal/${record.id}`),
                style: { cursor: "pointer" },
              })}
            />

            {hasNextPage && (
              <div className="flex justify-center p-4 border-t border-border">
                <Button
                  onClick={() => fetchNextPage()}
                  loading={isFetchingNextPage}
                >
                  {t("fin.journal.loadMore", "Load more")}
                </Button>
              </div>
            )}
          </>
        )}
      </Panel>
    </Page>
  );
}
