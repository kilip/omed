import type { components } from "@omed/openapi";
import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { Empty, Skeleton, Table, type TableProps, Tag } from "antd";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Link, useSearchParams } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { formatMoney } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import { AccountSelect, MoneyCell } from "../components";
import { accountLedgerQueryOptions, accountsQueryOptions } from "../queries";
import type { Route } from "./+types/ledger";

type Account = components["schemas"]["model.AccountResponse"];
type Posting = components["schemas"]["model.PostingResponse"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  await getQueryClient().ensureQueryData(accountsQueryOptions(wid));
  return { wid };
}

export default function GeneralLedger({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const [params, setParams] = useSearchParams();

  const accountParam = params.get("account") ?? params.get("accountId") ?? "";
  const pageParam = Number(params.get("page")) || 1;
  const limitParam = Number(params.get("limit")) || 20;

  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));

  // Determine active selected account
  const selectedAccount = useMemo<Account | undefined>(() => {
    if (!accountParam) {
      // Default to the first active postable account if available
      return accounts.find((a) => a.isPostable && a.isActive);
    }
    const idNum = Number(accountParam);
    if (!Number.isNaN(idNum) && idNum > 0) {
      const byId = accounts.find((a) => a.id === idNum);
      if (byId) return byId;
    }
    return accounts.find((a) => a.code === accountParam);
  }, [accounts, accountParam]);

  const selectedAccountId = selectedAccount?.id;

  const { data: ledger, isLoading: isLedgerLoading } = useQuery({
    ...accountLedgerQueryOptions(
      wid,
      selectedAccountId ?? 0,
      pageParam,
      limitParam,
    ),
    enabled: !!selectedAccountId,
  });

  const handleAccountChange = (val: number | null | undefined) => {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (val) {
          next.set("account", String(val));
          next.delete("accountId");
        } else {
          next.delete("account");
          next.delete("accountId");
        }
        next.set("page", "1");
        return next;
      },
      { replace: true },
    );
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set("page", String(page));
        next.set("limit", String(pageSize));
        return next;
      },
      { replace: true },
    );
  };

  const currency = selectedAccount?.currency || "IDR";
  const normalBalance = ledger?.normalBalance || selectedAccount?.normalBalance;

  const columns: TableProps<Posting>["columns"] = [
    {
      title: t("fin.ledger.postingId", "Posting #"),
      dataIndex: "id",
      width: 100,
      render: (id: number) => <span className="font-mono text-xs">#{id}</span>,
    },
    {
      title: t("fin.ledger.entry", "Entry #"),
      dataIndex: "entryId",
      width: 140,
      render: (entryId: number) => (
        <Link
          to={`/fin/journal/${entryId}`}
          className="font-medium text-primary hover:underline"
        >
          {t("fin.journal.number", "Entry")} #{entryId}
        </Link>
      ),
    },
    {
      title: t("fin.journal.label", "Description"),
      dataIndex: "description",
      render: (desc: string) => desc || "-",
    },
    {
      title: t("fin.journal.debit", "Debit"),
      align: "right",
      width: 150,
      render: (_, r) =>
        r.direction === "DEBIT" ? (
          <MoneyCell amount={r.amount} currency={currency} />
        ) : (
          ""
        ),
    },
    {
      title: t("fin.journal.credit", "Credit"),
      align: "right",
      width: 150,
      render: (_, r) =>
        r.direction === "CREDIT" ? (
          <MoneyCell amount={r.amount} currency={currency} />
        ) : (
          ""
        ),
    },
    {
      title: t("fin.ledger.balance", "Balance"),
      align: "right",
      width: 160,
      render: (_, r) => (
        <MoneyCell
          amount={r.runningBalance ?? r.balance}
          currency={currency}
          className="font-semibold"
        />
      ),
    },
  ];

  const postings = ledger?.postings ?? [];
  const totalItems = ledger?.total ?? 0;

  const totalDebit = postings.reduce(
    (sum, r) => (r.direction === "DEBIT" ? sum + (r.amount ?? 0) : sum),
    0,
  );
  const totalCredit = postings.reduce(
    (sum, r) => (r.direction === "CREDIT" ? sum + (r.amount ?? 0) : sum),
    0,
  );

  return (
    <Page title={t("fin.menu.ledger")}>
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-3">
          <AccountSelect
            accounts={accounts}
            value={selectedAccountId}
            onChange={handleAccountChange}
            placeholder={t("fin.ledger.selectAccountPlaceholder")}
            style={{ minWidth: 320 }}
            allowClear={false}
          />

          {normalBalance && (
            <Tag color="blue">
              {t("fin.ledger.normalBalance", "Normal")}:{" "}
              {t(`fin.normal.${normalBalance}`, normalBalance)}
            </Tag>
          )}
        </div>

        {selectedAccount && ledger && (
          <div className="flex items-center gap-4 text-sm">
            <div>
              <span className="text-secondary mr-2">
                {t("fin.ledger.balance", "Balance")}:
              </span>
              <strong className="num text-base font-semibold">
                {formatMoney(ledger.balance ?? 0, currency, i18n.language)}
              </strong>
            </div>
            {ledger.baseBalance !== undefined && currency !== "IDR" && (
              <div>
                <span className="text-secondary mr-2">
                  {t("fin.ledger.baseBalance", "Base balance")}:
                </span>
                <strong className="num text-base font-semibold">
                  {formatMoney(ledger.baseBalance, "IDR", i18n.language)}
                </strong>
              </div>
            )}
          </div>
        )}
      </div>

      {!selectedAccountId ? (
        <Panel strong>
          <Empty
            description={t(
              "fin.ledger.selectAccountPrompt",
              "Please select an account to view its general ledger postings.",
            )}
          />
        </Panel>
      ) : isLedgerLoading && !ledger ? (
        <Panel strong>
          <Skeleton active paragraph={{ rows: 6 }} />
        </Panel>
      ) : (
        <Panel strong>
          <Table<Posting>
            size="small"
            rowKey="id"
            columns={columns}
            dataSource={postings}
            pagination={{
              current: pageParam,
              pageSize: limitParam,
              total: totalItems,
              showSizeChanger: true,
              pageSizeOptions: ["10", "20", "50", "100"],
              onChange: handlePageChange,
            }}
            scroll={{ x: "max-content" }}
            locale={{
              emptyText: (
                <Empty
                  description={t(
                    "fin.ledger.noPostings",
                    "No postings found for this account.",
                  )}
                />
              ),
            }}
            summary={() =>
              postings.length > 0 ? (
                <Table.Summary.Row>
                  <Table.Summary.Cell index={0} colSpan={3}>
                    <strong>{t("common.total")}</strong>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={3} align="right">
                    <strong className="num">
                      {formatMoney(totalDebit, currency, i18n.language)}
                    </strong>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={4} align="right">
                    <strong className="num">
                      {formatMoney(totalCredit, currency, i18n.language)}
                    </strong>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={5} align="right">
                    <strong className="num">
                      {formatMoney(
                        ledger?.balance ?? 0,
                        currency,
                        i18n.language,
                      )}
                    </strong>
                  </Table.Summary.Cell>
                </Table.Summary.Row>
              ) : null
            }
          />
        </Panel>
      )}
    </Page>
  );
}
