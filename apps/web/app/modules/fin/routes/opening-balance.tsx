import {
  BankOutlined,
  CreditCardOutlined,
  DollarOutlined,
  HistoryOutlined,
  SaveOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  InputNumber,
  Row,
  Select,
  Space,
  Spin,
  Typography,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { redirect, useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapList, unwrapOne } from "~/lib/api";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  accountKeys,
  accountsQueryOptions,
  entryKeys,
  ledgerKeys,
  periodsQueryOptions,
} from "../queries";
import type { Route } from "./+types/opening-balance";

type Account = components["schemas"]["model.AccountResponse"];
type CreatePosting = components["schemas"]["model.CreatePostingRequest"];

const { Text, Title, Paragraph } = Typography;

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const accounts = await unwrapList(
    api.POST("/workspaces/{wid}/fin/accounts/search", {
      params: { path: { wid } },
      body: {},
    }),
  );
  if (accounts.length === 0) {
    return redirect("/fin/setup");
  }
  return { wid };
}

export default function OpeningBalance({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();

  const { data: accounts } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: periods } = useSuspenseQuery(periodsQueryOptions(wid));

  const openPeriods = useMemo(
    () => periods?.filter((p) => p.status === "OPEN") ?? [],
    [periods],
  );

  const [selectedPeriodId, setSelectedPeriodId] = useState<number | undefined>(
    openPeriods[0]?.id ?? periods?.[0]?.id,
  );

  const selectedPeriod = useMemo(
    () => periods?.find((p) => p.id === selectedPeriodId),
    [periods, selectedPeriodId],
  );

  // Filter relevant accounts
  const assetAccounts = useMemo(() => {
    return (accounts ?? []).filter(
      (a): a is Account & { id: number; code: string } =>
        Boolean(
          a.id &&
            a.code &&
            a.type === "ASSET" &&
            a.isPostable &&
            (a.code.startsWith("11") || a.code.startsWith("10")),
        ),
    );
  }, [accounts]);

  const liabilityAccounts = useMemo(() => {
    return (accounts ?? []).filter(
      (a): a is Account & { id: number; code: string } =>
        Boolean(
          a.id &&
            a.code &&
            a.type === "LIABILITY" &&
            a.isPostable &&
            (a.code.startsWith("21") || a.code.startsWith("20")),
        ),
    );
  }, [accounts]);

  const equityAccount = useMemo(() => {
    const list = accounts ?? [];
    return (
      list.find((a) => a.code === "3100" && a.isPostable && a.id) ||
      list.find((a) => a.type === "EQUITY" && a.isPostable && a.id)
    );
  }, [accounts]);

  const [assetBalances, setAssetBalances] = useState<Record<number, number>>(
    {},
  );
  const [liabilityBalances, setLiabilityBalances] = useState<
    Record<number, number>
  >({});
  const [existingEntryId, setExistingEntryId] = useState<number | null>(null);
  const [existingEntryNumber, setExistingEntryNumber] = useState<string | null>(
    null,
  );
  const [isSearchingEntry, setIsSearchingEntry] = useState(false);
  const [saving, setSaving] = useState(false);

  // Check if an opening balance entry already exists in the selected period
  useEffect(() => {
    if (!selectedPeriodId) return;
    let isCancelled = false;

    async function checkExistingEntry() {
      setIsSearchingEntry(true);
      try {
        const res = await unwrapList(
          api.POST("/workspaces/{wid}/fin/entries/search", {
            params: { path: { wid } },
            body: {
              periodId: selectedPeriodId,
              limit: 50,
            },
          }),
        );
        if (isCancelled) return;

        const openingEntry = res.find((entry) => {
          const desc = entry.description?.toLowerCase() ?? "";
          return (
            desc.includes("opening balance") || desc.includes("saldo awal")
          );
        });

        if (openingEntry?.postings) {
          setExistingEntryId(openingEntry.id ?? null);
          setExistingEntryNumber(openingEntry.number ?? null);

          const assets: Record<number, number> = {};
          const liabilities: Record<number, number> = {};

          for (const p of openingEntry.postings) {
            if (!p.accountId || p.amount === undefined) continue;
            const acc = accounts?.find((a) => a.id === p.accountId);
            if (!acc?.id) continue;
            const amt =
              typeof p.amount === "string"
                ? Number.parseFloat(p.amount) || 0
                : (p.amount ?? 0);
            if (acc.type === "ASSET" && p.direction === "DEBIT") {
              assets[acc.id] = amt;
            } else if (acc.type === "LIABILITY" && p.direction === "CREDIT") {
              liabilities[acc.id] = amt;
            }
          }
          setAssetBalances(assets);
          setLiabilityBalances(liabilities);
        } else {
          setExistingEntryId(null);
          setExistingEntryNumber(null);
        }
      } catch (err) {
        console.error("Failed to check existing opening entry:", err);
      } finally {
        if (!isCancelled) {
          setIsSearchingEntry(false);
        }
      }
    }

    checkExistingEntry();
    return () => {
      isCancelled = true;
    };
  }, [wid, selectedPeriodId, accounts]);

  const totalAssets = useMemo(() => {
    return Object.values(assetBalances).reduce(
      (acc, cur) => acc + (cur || 0),
      0,
    );
  }, [assetBalances]);

  const totalLiabilities = useMemo(() => {
    return Object.values(liabilityBalances).reduce(
      (acc, cur) => acc + (cur || 0),
      0,
    );
  }, [liabilityBalances]);

  const netEquity = useMemo(() => {
    return totalAssets - totalLiabilities;
  }, [totalAssets, totalLiabilities]);

  const handleSave = async () => {
    if (!selectedPeriod?.id || !selectedPeriod?.startDate) {
      message.error(t("fin.openingBalance.selectPeriod"));
      return;
    }
    if (!equityAccount?.id) {
      message.error(
        "Modal Pemilik (Owner's Capital) account not found in Chart of Accounts.",
      );
      return;
    }

    const hasAnyBalance = totalAssets > 0 || totalLiabilities > 0;
    if (!hasAnyBalance) {
      message.warning(t("fin.openingBalance.emptyWarning"));
      return;
    }

    const periodId = selectedPeriod.id;
    const periodStartDate = selectedPeriod.startDate;

    setSaving(true);
    try {
      // Void old entry if it already exists
      if (existingEntryId) {
        try {
          await unwrapOne(
            api.POST("/workspaces/{wid}/fin/entries/{id}/void", {
              params: { path: { wid, id: existingEntryId } },
            }),
          );
        } catch (voidErr) {
          console.warn(
            "Could not void existing entry, proceeding to create new entry:",
            voidErr,
          );
        }
      }

      // Build postings
      const postings: CreatePosting[] = [];

      // Asset debit postings
      for (const [accIdStr, amount] of Object.entries(assetBalances)) {
        if (amount > 0) {
          const accId = Number(accIdStr);
          const acc = accounts?.find((a) => a.id === accId);
          postings.push({
            accountId: accId,
            amount: amount.toString(),
            direction: "DEBIT",
            description: `Saldo Awal ${acc?.name ?? ""}`.trim(),
          });
        }
      }

      // Liability credit postings
      for (const [accIdStr, amount] of Object.entries(liabilityBalances)) {
        if (amount > 0) {
          const accId = Number(accIdStr);
          const acc = accounts?.find((a) => a.id === accId);
          postings.push({
            accountId: accId,
            amount: amount.toString(),
            direction: "CREDIT",
            description: `Saldo Awal ${acc?.name ?? ""}`.trim(),
          });
        }
      }

      // Net equity balancing posting
      if (netEquity > 0) {
        postings.push({
          accountId: equityAccount.id,
          amount: netEquity.toString(),
          direction: "CREDIT",
          description: "Saldo Awal Modal Pemilik",
        });
      } else if (netEquity < 0) {
        postings.push({
          accountId: equityAccount.id,
          amount: Math.abs(netEquity).toString(),
          direction: "DEBIT",
          description: "Saldo Awal Modal Pemilik",
        });
      }

      if (postings.length < 2) {
        message.warning(t("fin.openingBalance.emptyWarning"));
        return;
      }

      // Create transaction with inline entry
      const txRes = await unwrapOne(
        api.POST("/workspaces/{wid}/fin/transactions", {
          params: { path: { wid } },
          body: {
            transactionDate: periodStartDate,
            description: "Opening Balance / Saldo Awal",
            currency: "IDR",
            exchangeRate: "1",
            source: "MANUAL",
            entry: {
              periodId,
              entryDate: periodStartDate,
              description: "Opening Balance / Saldo Awal",
              postings,
            },
          },
        }),
      );

      const createdEntry = txRes?.entries?.[0];

      // Post entry
      if (createdEntry?.id) {
        await unwrapOne(
          api.POST("/workspaces/{wid}/fin/entries/{id}/post", {
            params: { path: { wid, id: createdEntry.id } },
          }),
        );
      }

      message.success(t("fin.openingBalance.saveSuccess"));
      queryClient.removeQueries({ queryKey: accountKeys.all(wid) });
      queryClient.removeQueries({ queryKey: entryKeys.all(wid) });
      queryClient.removeQueries({ queryKey: ledgerKeys.all(wid) });

      setExistingEntryId(createdEntry?.id ?? null);
      setExistingEntryNumber(createdEntry?.number ?? null);
    } catch (err) {
      message.error(
        err instanceof Error ? err.message : "Error saving opening balance",
      );
    } finally {
      setSaving(false);
    }
  };

  return (
    <Page title={t("fin.openingBalance.title")}>
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6 py-2">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <Title level={4} style={{ marginBottom: 4 }}>
              {t("fin.openingBalance.title")}
            </Title>
            <Paragraph type="secondary" style={{ marginBottom: 0 }}>
              {t("fin.openingBalance.subtitle")}
            </Paragraph>
          </div>
          <Button onClick={() => navigate("/fin/accounts")}>
            {t("fin.setup.toAccounts")}
          </Button>
        </div>

        {/* Period Selector */}
        <Panel strong title={t("fin.openingBalance.period")}>
          <div className="flex flex-col gap-3">
            <Select
              className="w-full sm:w-80"
              value={selectedPeriodId}
              placeholder={t("fin.openingBalance.selectPeriod")}
              onChange={(val) => setSelectedPeriodId(val)}
              options={(periods ?? []).map((p) => ({
                label: `${p.name} (${p.startDate} - ${p.endDate}) [${p.status}]`,
                value: p.id,
              }))}
            />

            {existingEntryNumber && (
              <Alert
                type="info"
                showIcon
                icon={<HistoryOutlined />}
                message={t("fin.openingBalance.reentryNotice", {
                  number: existingEntryNumber,
                })}
              />
            )}
          </div>
        </Panel>

        {isSearchingEntry ? (
          <div className="flex justify-center py-8">
            <Spin />
          </div>
        ) : (
          <>
            {/* Section 1: Liquid Assets (Cash, Bank, E-Wallet) */}
            <Panel strong title={t("fin.openingBalance.assetsTitle")}>
              <Paragraph type="secondary" style={{ marginBottom: 16 }}>
                {t("fin.openingBalance.assetsDesc")}
              </Paragraph>
              <div className="flex flex-col gap-3">
                {assetAccounts.map((account) => (
                  <div
                    key={account.id}
                    className="flex flex-wrap items-center justify-between gap-4 rounded-md border p-3"
                    style={{
                      borderColor: "var(--omed-border-soft)",
                      backgroundColor: "var(--omed-surface-muted)",
                    }}
                  >
                    <div className="flex items-center gap-3">
                      {account.code.startsWith("111") ? (
                        <DollarOutlined
                          style={{ fontSize: 18, color: "var(--omed-primary)" }}
                        />
                      ) : (
                        <BankOutlined
                          style={{ fontSize: 18, color: "var(--omed-primary)" }}
                        />
                      )}
                      <div className="flex flex-col">
                        <Space>
                          <code className="font-mono font-semibold">
                            {account.code}
                          </code>
                          <span className="font-medium">{account.name}</span>
                        </Space>
                        {account.description && (
                          <span className="text-xs text-muted-foreground">
                            {account.description}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <InputNumber
                        aria-label={`balance-${account.code}`}
                        className="w-48"
                        min={0}
                        placeholder="0"
                        prefix="Rp"
                        value={assetBalances[account.id]}
                        onChange={(val) =>
                          setAssetBalances((prev) => ({
                            ...prev,
                            [account.id]: val ?? 0,
                          }))
                        }
                      />
                    </div>
                  </div>
                ))}
              </div>
            </Panel>

            {/* Section 2: Current Liabilities & PayLater */}
            <Panel strong title={t("fin.openingBalance.liabilitiesTitle")}>
              <Paragraph type="secondary" style={{ marginBottom: 16 }}>
                {t("fin.openingBalance.liabilitiesDesc")}
              </Paragraph>
              <div className="flex flex-col gap-3">
                {liabilityAccounts.map((account) => (
                  <div
                    key={account.id}
                    className="flex flex-wrap items-center justify-between gap-4 rounded-md border p-3"
                    style={{
                      borderColor: "var(--omed-border-soft)",
                      backgroundColor: "var(--omed-surface-muted)",
                    }}
                  >
                    <div className="flex items-center gap-3">
                      <CreditCardOutlined
                        style={{ fontSize: 18, color: "#fa541c" }}
                      />
                      <div className="flex flex-col">
                        <Space>
                          <code className="font-mono font-semibold">
                            {account.code}
                          </code>
                          <span className="font-medium">{account.name}</span>
                        </Space>
                        {account.description && (
                          <span className="text-xs text-muted-foreground">
                            {account.description}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <InputNumber
                        aria-label={`balance-${account.code}`}
                        className="w-48"
                        min={0}
                        placeholder="0"
                        prefix="Rp"
                        value={liabilityBalances[account.id]}
                        onChange={(val) =>
                          setLiabilityBalances((prev) => ({
                            ...prev,
                            [account.id]: val ?? 0,
                          }))
                        }
                      />
                    </div>
                  </div>
                ))}
              </div>
            </Panel>

            {/* Section 3: Summary of Equity */}
            <Panel strong title={t("fin.openingBalance.summaryTitle")}>
              <Paragraph type="secondary" style={{ marginBottom: 16 }}>
                {t("fin.openingBalance.netEquityDesc")}
              </Paragraph>
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={8}>
                  <Card
                    size="small"
                    style={{ backgroundColor: "var(--omed-surface-muted)" }}
                  >
                    <Text type="secondary">
                      {t("fin.openingBalance.totalAssets")}
                    </Text>
                    <Title
                      level={4}
                      style={{ margin: "4px 0 0", color: "#1890ff" }}
                    >
                      Rp {totalAssets.toLocaleString("id-ID")}
                    </Title>
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card
                    size="small"
                    style={{ backgroundColor: "var(--omed-surface-muted)" }}
                  >
                    <Text type="secondary">
                      {t("fin.openingBalance.totalLiabilities")}
                    </Text>
                    <Title
                      level={4}
                      style={{ margin: "4px 0 0", color: "#fa541c" }}
                    >
                      Rp {totalLiabilities.toLocaleString("id-ID")}
                    </Title>
                  </Card>
                </Col>
                <Col xs={24} sm={8}>
                  <Card
                    size="small"
                    style={{ backgroundColor: "var(--omed-tint)" }}
                  >
                    <Text type="secondary">
                      {t("fin.openingBalance.netEquity")}
                    </Text>
                    <Title
                      level={4}
                      style={{
                        margin: "4px 0 0",
                        color: "var(--omed-primary)",
                      }}
                    >
                      Rp {netEquity.toLocaleString("id-ID")}
                    </Title>
                  </Card>
                </Col>
              </Row>
            </Panel>

            {/* Actions */}
            <div className="flex items-center justify-end gap-3 pt-2">
              <Button onClick={() => navigate("/fin/accounts")}>
                {t("fin.setup.cancel")}
              </Button>
              <Button
                type="primary"
                icon={<SaveOutlined />}
                loading={saving}
                onClick={handleSave}
              >
                {t("fin.openingBalance.save")}
              </Button>
            </div>
          </>
        )}
      </div>
    </Page>
  );
}
