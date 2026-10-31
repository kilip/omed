import { PaperClipOutlined, PlusOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useInfiniteQuery, useSuspenseQuery } from "@tanstack/react-query";
import {
  Button,
  DatePicker,
  Empty,
  Select,
  Table,
  type TableProps,
  Tag,
} from "antd";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
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
import {
  EntryStatusTag,
  groupPaymentAccounts,
  MoneyCell,
  summarizeCategories,
} from "../components";
import { useFinPermissions } from "../hooks";
import {
  accountsQueryOptions,
  type CashFlowFilters,
  expensesInfiniteQueryOptions,
} from "../queries";
import type { Route } from "./+types/expense";

type CashFlowItem = components["schemas"]["model.CashFlowItemResponse"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureInfiniteQueryData(expensesInfiniteQueryOptions(wid, {})),
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(permissionsQueryOptions(wid)),
  ]);
  return { wid };
}

export default function ExpenseRoute({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const { canCreateTransaction } = useFinPermissions(wid);

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));

  const [queryText, setQueryText] = useState("");
  const [paymentAccountFilter, setPaymentAccountFilter] = useState<
    number | undefined
  >();
  const [categoryAccountFilter, setCategoryAccountFilter] = useState<
    number | undefined
  >();
  const [statusFilter, setStatusFilter] = useState<
    "DRAFT" | "POSTED" | undefined
  >();
  const [dateRange, setDateRange] = useState<[string, string] | undefined>();

  const filters = useMemo<CashFlowFilters>(() => {
    return {
      query: queryText.trim() || undefined,
      paymentAccountId: paymentAccountFilter,
      categoryAccountId: categoryAccountFilter,
      status: statusFilter,
      dateFrom: dateRange?.[0],
      dateTo: dateRange?.[1],
    };
  }, [
    queryText,
    paymentAccountFilter,
    categoryAccountFilter,
    statusFilter,
    dateRange,
  ]);

  const { data, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useInfiniteQuery(expensesInfiniteQueryOptions(wid, filters));

  const expenses = useMemo(() => {
    return data?.pages.flatMap((page) => page.items) ?? [];
  }, [data]);

  const paymentGroups = useMemo(
    () => groupPaymentAccounts(accounts),
    [accounts],
  );

  const expenseCategoryOptions = useMemo(() => {
    return accounts
      .filter(
        (a) => a.isActive !== false && a.isPostable && a.type === "EXPENSE",
      )
      .map((a) => ({
        value: a.id as number,
        label: `${a.code} - ${a.name}`,
      }));
  }, [accounts]);

  const facets = useMemo<SearchFacet[]>(() => {
    const items: SearchFacet[] = [];
    if (paymentAccountFilter) {
      const a = accounts.find((item) => item.id === paymentAccountFilter);
      items.push({
        key: `pay-${paymentAccountFilter}`,
        label: t("fin.expense.filterPaymentAccount", {
          account: a ? `${a.code} - ${a.name}` : String(paymentAccountFilter),
        }),
        onRemove: () => setPaymentAccountFilter(undefined),
      });
    }
    if (categoryAccountFilter) {
      const a = accounts.find((item) => item.id === categoryAccountFilter);
      items.push({
        key: `cat-${categoryAccountFilter}`,
        label: t("fin.expense.filterCategory", {
          account: a ? `${a.code} - ${a.name}` : String(categoryAccountFilter),
        }),
        onRemove: () => setCategoryAccountFilter(undefined),
      });
    }
    if (dateRange) {
      items.push({
        key: `date-${dateRange[0]}-${dateRange[1]}`,
        label: t("fin.expense.filterDate", {
          range: `${dateRange[0]} ~ ${dateRange[1]}`,
        }),
        onRemove: () => setDateRange(undefined),
      });
    }
    if (statusFilter) {
      items.push({
        key: `status-${statusFilter}`,
        label: t("fin.cashflow.filterStatus", {
          status:
            statusFilter === "DRAFT"
              ? t("fin.entry.status.draft", "Draft")
              : t("fin.entry.status.posted", "Posted"),
        }),
        onRemove: () => setStatusFilter(undefined),
      });
    }
    return items;
  }, [
    paymentAccountFilter,
    categoryAccountFilter,
    statusFilter,
    dateRange,
    accounts,
    t,
  ]);

  const columns: TableProps<CashFlowItem>["columns"] = [
    {
      title: t("fin.expense.date", "Date"),
      dataIndex: "transactionDate",
      key: "transactionDate",
      render: (val) => formatDate(val, i18n.language),
    },
    {
      title: t("fin.transactions.number", "Number"),
      dataIndex: "number",
      key: "number",
      render: (num, record) => (
        <Link
          to={`/fin/expense/${record.id}`}
          onClick={(e) => e.stopPropagation()}
          className="font-semibold text-primary hover:underline"
        >
          {num || `#${record.id}`}
        </Link>
      ),
    },
    {
      title: t("fin.cashflow.status", "Status"),
      dataIndex: "entryStatus",
      key: "entryStatus",
      render: (status) => <EntryStatusTag status={status} />,
    },
    {
      title: t("fin.expense.paymentAccount", "Payment Account"),
      dataIndex: "paymentAccount",
      key: "paymentAccount",
      render: (_, record) =>
        record.paymentAccount
          ? `${record.paymentAccount.code} - ${record.paymentAccount.name}`
          : "-",
    },
    {
      title: t("fin.expense.expenseCategory", "Expense Category"),
      dataIndex: "items",
      key: "items",
      render: (_, record) => {
        const summary = summarizeCategories(record.items ?? []);
        if (!summary.first) return "-";
        return summary.more > 0
          ? `${summary.first} (+${summary.more})`
          : summary.first;
      },
    },
    {
      title: t("fin.expense.description", "Description"),
      dataIndex: "description",
      key: "description",
      render: (text) => text || "-",
    },
    {
      title: t("fin.expense.reference", "Reference"),
      dataIndex: "reference",
      key: "reference",
      render: (text) => text || "-",
    },
    {
      title: t("fin.expense.amount", "Amount"),
      dataIndex: "amount",
      key: "amount",
      align: "right",
      render: (val, record) => (
        <MoneyCell amount={val ?? 0} currency={record.currency ?? "IDR"} />
      ),
    },
    {
      title: t("fin.expense.receiptPicker", "Attachments"),
      dataIndex: "attachments",
      key: "attachments",
      align: "center",
      render: (_, record) => {
        const count = record.attachments?.length ?? 0;
        return count > 0 ? (
          <Tag icon={<PaperClipOutlined />}>{count}</Tag>
        ) : (
          "-"
        );
      },
    },
  ];

  const handleDateRangeChange = (
    dates: [Dayjs | null, Dayjs | null] | null,
  ) => {
    if (dates?.[0] && dates[1]) {
      setDateRange([
        dates[0].format("YYYY-MM-DD"),
        dates[1].format("YYYY-MM-DD"),
      ]);
    } else {
      setDateRange(undefined);
    }
  };

  const dateRangeValues: [Dayjs, Dayjs] | null = dateRange
    ? [dayjs(dateRange[0]), dayjs(dateRange[1])]
    : null;

  return (
    <Page
      title={t("fin.menu.expense", "Expense")}
      search={{
        value: queryText,
        onChange: setQueryText,
        placeholder: t("common.search", "Search..."),
        facets,
      }}
      actions={
        canCreateTransaction ? (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate("/fin/expense/new")}
          >
            {t("fin.expense.new", "New Expense")}
          </Button>
        ) : undefined
      }
      count={expenses.length}
    >
      <Panel strong>
        {/* Filter bar */}
        <div
          className="flex flex-wrap items-center gap-2 mb-3"
          data-testid="cashflow-filter-bar"
        >
          <Select
            placeholder={t("fin.cashflow.status", "Status")}
            allowClear
            className="min-w-[120px]"
            value={statusFilter}
            onChange={setStatusFilter}
            options={[
              { value: "DRAFT", label: t("fin.entry.status.draft", "Draft") },
              {
                value: "POSTED",
                label: t("fin.entry.status.posted", "Posted"),
              },
            ]}
          />
          <Select
            placeholder={t("fin.expense.paymentAccount", "Payment Account")}
            allowClear
            showSearch
            className="min-w-[180px]"
            value={paymentAccountFilter}
            onChange={setPaymentAccountFilter}
            options={paymentGroups}
          />
          <Select
            placeholder={t("fin.expense.expenseCategory", "Expense Category")}
            allowClear
            showSearch
            className="min-w-[180px]"
            value={categoryAccountFilter}
            onChange={setCategoryAccountFilter}
            options={expenseCategoryOptions}
          />
          <DatePicker.RangePicker
            value={dateRangeValues}
            onChange={handleDateRangeChange}
          />
        </div>

        {expenses.length === 0 ? (
          <Empty
            description={t("fin.expense.emptyDesc", "No expense records found")}
            className="py-12"
          />
        ) : (
          <>
            <Table<CashFlowItem>
              size="small"
              rowKey="id"
              columns={columns}
              dataSource={expenses}
              pagination={false}
              scroll={{ x: "max-content" }}
              onRow={(record) => ({
                onClick: () => navigate(`/fin/expense/${record.id}`),
                className: "cursor-pointer hover:bg-muted/50",
              })}
            />

            {hasNextPage && (
              <div className="flex justify-center p-4 border-t border-border">
                <Button
                  onClick={() => fetchNextPage()}
                  loading={isFetchingNextPage}
                >
                  {t("common.loadMore", "Load more")}
                </Button>
              </div>
            )}
          </>
        )}
      </Panel>
    </Page>
  );
}
