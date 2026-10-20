import { PlusOutlined } from "@ant-design/icons";
import type { components, paths } from "@omed/openapi";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button, Empty, type MenuProps, Table, type TableProps } from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import type { SearchFacet } from "~/components/SearchBar";
import { formatDate } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { permissionsQueryOptions } from "~/modules/core/queries";
import { useFinPermissions } from "../hooks";
import { transactionsInfiniteQueryOptions } from "../queries";
import type { Route } from "./+types/transactions";

type Transaction = components["schemas"]["model.TransactionResponse"];
type SearchFilters = NonNullable<
  paths["/workspaces/{wid}/fin/transactions/search"]["post"]["requestBody"]
>["content"]["application/json"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureInfiniteQueryData(
      transactionsInfiniteQueryOptions(wid, {}),
    ),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid };
}

export default function TransactionsRoute({
  loaderData,
}: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { canCreateTransaction } = useFinPermissions(wid);

  const [queryText, setQueryText] = useState("");
  const [sourceFilter, setSourceFilter] = useState<
    SearchFilters["source"] | undefined
  >();
  const [referenceFilter, setReferenceFilter] = useState<string | undefined>();
  const [dateRange, setDateRange] = useState<[string, string] | undefined>();

  const filters = useMemo<SearchFilters>(() => {
    return {
      query: queryText.trim() || undefined,
      source: sourceFilter,
      reference: referenceFilter,
      dateFrom: dateRange?.[0],
      dateTo: dateRange?.[1],
    };
  }, [queryText, sourceFilter, referenceFilter, dateRange]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteQuery(transactionsInfiniteQueryOptions(wid, filters));

  const transactions = useMemo(() => {
    return data?.pages.flatMap((page) => page.items) ?? [];
  }, [data]);

  const facets = useMemo<SearchFacet[]>(() => {
    const items: SearchFacet[] = [];
    if (sourceFilter) {
      items.push({
        key: `source-${sourceFilter}`,
        label: t("fin.transactions.filterSource", {
          source: t(`fin.transactions.sources.${sourceFilter}`, sourceFilter),
        }),
        onRemove: () => setSourceFilter(undefined),
      });
    }
    if (referenceFilter) {
      items.push({
        key: `ref-${referenceFilter}`,
        label: t("fin.transactions.filterReference", {
          ref: referenceFilter,
        }),
        onRemove: () => setReferenceFilter(undefined),
      });
    }
    if (dateRange) {
      items.push({
        key: `date-${dateRange[0]}-${dateRange[1]}`,
        label: t("fin.transactions.filterDate", {
          range: `${dateRange[0]} ~ ${dateRange[1]}`,
        }),
        onRemove: () => setDateRange(undefined),
      });
    }
    return items;
  }, [sourceFilter, referenceFilter, dateRange, t]);

  const filterMenuItems: MenuProps["items"] = [
    {
      key: "source-group",
      label: t("fin.transactions.source", "Source"),
      type: "group",
      children: [
        {
          key: "source:MANUAL",
          label: t("fin.transactions.sources.MANUAL", "Manual"),
        },
        {
          key: "source:IMPORT",
          label: t("fin.transactions.sources.IMPORT", "Import"),
        },
        {
          key: "source:SYSTEM",
          label: t("fin.transactions.sources.SYSTEM", "System"),
        },
      ],
    },
  ];

  const handleFilterSelect = (key: string) => {
    if (key.startsWith("source:")) {
      const val = key.replace("source:", "") as SearchFilters["source"];
      setSourceFilter((curr) => (curr === val ? undefined : val));
    }
  };

  const columns: TableProps<Transaction>["columns"] = [
    {
      title: t("fin.transactions.number", "Number"),
      dataIndex: "number",
      key: "number",
      render: (num, record) => (
        <span className="font-semibold text-primary">
          {num || `#${record.id}`}
        </span>
      ),
    },
    {
      title: t("fin.transactions.date", "Date"),
      dataIndex: "transactionDate",
      key: "transactionDate",
      render: (val) => formatDate(val, i18n.language),
    },
    {
      title: t("fin.transactions.reference", "Reference"),
      dataIndex: "reference",
      key: "reference",
      render: (text) => text || "-",
    },
    {
      title: t("fin.transactions.description", "Description"),
      dataIndex: "description",
      key: "description",
      render: (text) => text || "-",
    },
    {
      title: t("fin.transactions.currency", "Currency"),
      dataIndex: "currency",
      key: "currency",
      render: (cur) => cur || "IDR",
    },
    {
      title: t("fin.transactions.source", "Source"),
      dataIndex: "source",
      key: "source",
      render: (src) => (src ? t(`fin.transactions.sources.${src}`, src) : "-"),
    },
  ];

  return (
    <Page
      title={t("fin.menu.transactions", "Transactions")}
      search={{
        value: queryText,
        onChange: setQueryText,
        placeholder: t(
          "fin.transactions.description",
          "Search number or description...",
        ),
        facets,
        filterItems: filterMenuItems,
        onFilterSelect: handleFilterSelect,
      }}
      actions={
        canCreateTransaction ? (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate("/fin/transactions/new")}
          >
            {t("fin.transactions.new", "New transaction")}
          </Button>
        ) : undefined
      }
    >
      <Panel>
        <Table<Transaction>
          rowKey="id"
          columns={columns}
          dataSource={transactions}
          pagination={false}
          locale={{
            emptyText: (
              <Empty
                description={
                  <span>
                    {t("fin.transactions.emptyTitle", "No transactions found")}
                  </span>
                }
              />
            ),
          }}
          onRow={(record) => ({
            onClick: () => {
              if (record.id) {
                navigate(`/fin/transactions/${record.id}`);
              }
            },
            className: "cursor-pointer hover:bg-muted/50",
          })}
        />

        {hasNextPage && (
          <div className="flex justify-center p-4 border-t">
            <Button
              onClick={() => fetchNextPage()}
              loading={isFetchingNextPage}
            >
              {t("fin.journal.loadMore", "Load more")}
            </Button>
          </div>
        )}
      </Panel>
    </Page>
  );
}
