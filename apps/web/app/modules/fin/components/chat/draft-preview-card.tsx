import {
  CheckOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { Button, Input, InputNumber, Popconfirm, Select, Tag } from "antd";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { formatMoney } from "~/lib/format";
import type { DraftProposal } from "../../hooks/use-agent-chat";

type Account = components["schemas"]["model.AccountResponse"];
type DraftItem = components["schemas"]["model.DraftItem"];
type DraftJournalEntry = components["schemas"]["model.DraftJournalEntry"];

export interface DraftPreviewCardProps {
  proposal: DraftProposal;
  status?: "proposed" | "confirmed" | "discarded" | string;
  transactionId?: number;
  onConfirm?: (editedProposal?: DraftProposal) => void | Promise<void>;
  onDiscard?: () => void | Promise<void>;
  isConfirming?: boolean;
  isDiscarding?: boolean;
  accounts?: Account[];
  wid?: number;
}

export function DraftPreviewCard({
  proposal,
  status = "proposed",
  transactionId,
  onConfirm,
  onDiscard,
  isConfirming = false,
  isDiscarding = false,
  accounts = [],
}: DraftPreviewCardProps) {
  const { t, i18n } = useTranslation();
  const lang = i18n.language || "en";

  const [isEditing, setIsEditing] = useState(false);
  const [hasEdited, setHasEdited] = useState(false);
  const [draft, setDraft] = useState<DraftProposal>(() => ({
    ...proposal,
    items: proposal.items
      ? proposal.items.map((i, idx) => ({ ...i, _key: `item-${idx}` }))
      : [],
    entries: proposal.entries
      ? proposal.entries.map((e, idx) => ({ ...e, _key: `entry-${idx}` }))
      : [],
  }));

  const kind = draft.kind || proposal.kind || "expense";
  const isJournal = kind === "journal";
  const isIncome = kind === "income";
  const isExpense = kind === "expense" || (!isJournal && !isIncome);

  const isConfirmed = status === "confirmed";
  const isDiscarded = status === "discarded";
  const isReadOnly = isConfirmed || isDiscarded;

  const accountMap = useMemo(() => {
    const map = new Map<number, Account>();
    for (const a of accounts) {
      if (a.id) map.set(a.id, a);
    }
    return map;
  }, [accounts]);

  const postableAccountOptions = useMemo(() => {
    return accounts
      .filter((a) => a.isPostable && a.isActive)
      .map((a) => ({
        value: a.id as number,
        label: `${a.code} - ${a.name}`,
      }));
  }, [accounts]);

  const getAccountLabel = (accountId?: number) => {
    if (!accountId) return "-";
    const acc = accountMap.get(accountId);
    if (acc) return `${acc.code} - ${acc.name}`;
    return String(accountId);
  };

  const handleStartEdit = () => {
    setIsEditing(true);
  };

  const handleCancelEdit = () => {
    setDraft({
      ...proposal,
      items: proposal.items
        ? proposal.items.map((i, idx) => ({ ...i, _key: `item-${idx}` }))
        : [],
      entries: proposal.entries
        ? proposal.entries.map((e, idx) => ({ ...e, _key: `entry-${idx}` }))
        : [],
    });
    setHasEdited(false);
    setIsEditing(false);
  };

  const handleSaveEdit = () => {
    setIsEditing(false);
    setHasEdited(true);
  };

  const handleItemChange = (
    index: number,
    field: keyof DraftItem,
    value: unknown,
  ) => {
    const items = [...(draft.items || [])];
    const item = { ...items[index], [field]: value };

    // Auto-calculate amount if quantity or unitPrice changed
    if (field === "quantity" || field === "unitPrice") {
      const q = Number.parseFloat(String(item.quantity ?? "0")) || 0;
      const p = Number.parseFloat(String(item.unitPrice ?? "0")) || 0;
      if (q > 0 && p > 0) {
        item.amount = (q * p).toFixed(2);
      }
    }

    items[index] = item;

    // Recalculate total
    const total = items.reduce((acc, curr) => {
      const amt = Number.parseFloat(String(curr.amount ?? "0")) || 0;
      return acc + amt;
    }, 0);

    setDraft((prev) => ({
      ...prev,
      items,
      totalAmount: total > 0 ? total.toFixed(2) : prev.totalAmount,
    }));
    setHasEdited(true);
  };

  const handleAddItem = () => {
    const items = [
      ...(draft.items || []),
      {
        accountId: undefined,
        description: "",
        quantity: "1",
        unitPrice: "0",
        amount: "0",
        _key: `item-new-${Date.now()}-${Math.random()}`,
      },
    ];
    setDraft((prev) => ({ ...prev, items }));
    setHasEdited(true);
  };

  const handleRemoveItem = (index: number) => {
    const items = (draft.items || []).filter((_, i) => i !== index);
    const total = items.reduce((acc, curr) => {
      const amt = Number.parseFloat(String(curr.amount ?? "0")) || 0;
      return acc + amt;
    }, 0);
    setDraft((prev) => ({
      ...prev,
      items,
      totalAmount: total > 0 ? total.toFixed(2) : prev.totalAmount,
    }));
    setHasEdited(true);
  };

  const handleEntryChange = (
    index: number,
    field: keyof DraftJournalEntry,
    value: unknown,
  ) => {
    const entries = [...(draft.entries || [])];
    entries[index] = { ...entries[index], [field]: value };

    // Recalculate total amount from debit lines
    const totalDebit = entries
      .filter((e) => e.direction?.toUpperCase() === "DEBIT")
      .reduce((acc, curr) => {
        const amt = Number.parseFloat(String(curr.amount ?? "0")) || 0;
        return acc + amt;
      }, 0);

    setDraft((prev) => ({
      ...prev,
      entries,
      totalAmount: totalDebit > 0 ? totalDebit.toFixed(2) : prev.totalAmount,
    }));
    setHasEdited(true);
  };

  const handleAddEntry = () => {
    const entries = [
      ...(draft.entries || []),
      {
        accountId: undefined,
        direction: "DEBIT",
        amount: "0",
        description: "",
        _key: `entry-new-${Date.now()}-${Math.random()}`,
      },
    ];
    setDraft((prev) => ({ ...prev, entries }));
    setHasEdited(true);
  };

  const handleRemoveEntry = (index: number) => {
    const entries = (draft.entries || []).filter((_, i) => i !== index);
    setDraft((prev) => ({ ...prev, entries }));
    setHasEdited(true);
  };

  // Journal balance check
  const journalBalance = useMemo(() => {
    if (!isJournal) return { debit: 0, credit: 0, diff: 0, balanced: true };
    const debit = (draft.entries || [])
      .filter((e) => e.direction?.toUpperCase() === "DEBIT")
      .reduce(
        (sum, e) => sum + (Number.parseFloat(String(e.amount ?? "0")) || 0),
        0,
      );
    const credit = (draft.entries || [])
      .filter((e) => e.direction?.toUpperCase() === "CREDIT")
      .reduce(
        (sum, e) => sum + (Number.parseFloat(String(e.amount ?? "0")) || 0),
        0,
      );
    const diff = Math.abs(debit - credit);
    return {
      debit,
      credit,
      diff,
      balanced: diff < 0.005,
    };
  }, [isJournal, draft.entries]);

  return (
    <div
      data-testid="draft-preview-card"
      className="rounded-lg border border-[var(--omed-border)] bg-[var(--omed-surface)] overflow-hidden shadow-xs my-2 text-sm"
    >
      {/* Header bar */}
      <div className="flex flex-wrap items-center justify-between gap-2 px-4 py-2.5 bg-[var(--omed-surface-muted)] border-b border-[var(--omed-border-soft)]">
        <div className="flex items-center gap-2">
          {isExpense && (
            <Tag color="orange" className="font-medium m-0">
              {t("fin.chat.expenseDraft", "Expense Draft")}
            </Tag>
          )}
          {isIncome && (
            <Tag color="green" className="font-medium m-0">
              {t("fin.chat.incomeDraft", "Income Draft")}
            </Tag>
          )}
          {isJournal && (
            <Tag color="purple" className="font-medium m-0">
              {t("fin.chat.journalDraft", "Journal Draft")}
            </Tag>
          )}

          {isConfirmed && (
            <Tag color="success" className="font-medium m-0">
              {t("fin.chat.confirmed", "Confirmed")}
            </Tag>
          )}
          {isDiscarded && (
            <Tag className="font-medium m-0">
              {t("fin.chat.discarded", "Discarded")}
            </Tag>
          )}
          {!isReadOnly && (
            <Tag color="processing" className="font-medium m-0">
              {t("fin.chat.proposed", "Draft Proposal")}
            </Tag>
          )}
        </div>

        {isConfirmed && transactionId && (
          <Link
            to={`/fin/transactions/${transactionId}`}
            className="text-xs font-semibold hover:underline flex items-center gap-1"
            style={{ color: "var(--omed-primary-text)" }}
          >
            {t("fin.chat.viewTransaction", "View Transaction")} #{transactionId}{" "}
            &rarr;
          </Link>
        )}
      </div>

      {/* Main card body */}
      <div className="p-4 space-y-4">
        {/* Basic metadata */}
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 text-xs">
          <div>
            <div className="text-[var(--omed-text-secondary)] font-medium mb-1">
              {t("fin.chat.date", "Date")}
            </div>
            {isEditing ? (
              <input
                aria-label={t("fin.chat.date", "Date")}
                type="date"
                value={draft.transactionDate || ""}
                onChange={(e) => {
                  setDraft((prev) => ({
                    ...prev,
                    transactionDate: e.target.value,
                  }));
                  setHasEdited(true);
                }}
                className="w-full px-2 py-1 border border-[var(--omed-border)] rounded bg-[var(--omed-surface)] text-[var(--omed-text)]"
              />
            ) : (
              <div className="font-medium text-[var(--omed-text)]">
                {draft.transactionDate || "-"}
              </div>
            )}
          </div>

          <div>
            <div className="text-[var(--omed-text-secondary)] font-medium mb-1">
              {t("fin.chat.reference", "Reference")}
            </div>
            {isEditing ? (
              <Input
                aria-label={t("fin.chat.reference", "Reference")}
                size="small"
                value={draft.reference || ""}
                onChange={(e) => {
                  setDraft((prev) => ({ ...prev, reference: e.target.value }));
                  setHasEdited(true);
                }}
              />
            ) : (
              <div className="font-medium text-[var(--omed-text)]">
                {draft.reference || "-"}
              </div>
            )}
          </div>

          {!isJournal && (
            <div>
              <div className="text-[var(--omed-text-secondary)] font-medium mb-1">
                {t("fin.chat.paymentAccount", "Payment Account")}
              </div>
              {isEditing ? (
                <Select
                  aria-label={t("fin.chat.paymentAccount", "Payment Account")}
                  size="small"
                  className="w-full"
                  showSearch
                  optionFilterProp="label"
                  value={draft.paymentAccountId}
                  onChange={(val) => {
                    setDraft((prev) => ({ ...prev, paymentAccountId: val }));
                    setHasEdited(true);
                  }}
                  options={postableAccountOptions}
                  placeholder={t("fin.chat.paymentAccount", "Payment Account")}
                />
              ) : (
                <div className="font-medium text-[var(--omed-text)] truncate">
                  {getAccountLabel(draft.paymentAccountId)}
                </div>
              )}
            </div>
          )}

          <div>
            <div className="text-[var(--omed-text-secondary)] font-medium mb-1">
              {t("fin.chat.totalAmount", "Total Amount")}
            </div>
            <div
              className="font-semibold text-sm"
              style={{ color: "var(--omed-primary-text)" }}
            >
              {formatMoney(
                draft.totalAmount || 0,
                draft.currency || "IDR",
                lang,
              )}
            </div>
          </div>
        </div>

        {/* Description row */}
        <div>
          <div className="text-xs text-[var(--omed-text-secondary)] font-medium mb-1">
            {t("fin.chat.description", "Description")}
          </div>
          {isEditing ? (
            <Input
              aria-label={t("fin.chat.description", "Description")}
              size="small"
              value={draft.description || ""}
              onChange={(e) => {
                setDraft((prev) => ({ ...prev, description: e.target.value }));
                setHasEdited(true);
              }}
            />
          ) : (
            <div className="text-xs text-[var(--omed-text)]">
              {draft.description || "-"}
            </div>
          )}
        </div>

        {/* Multi-item breakdown for Expense / Income */}
        {!isJournal && (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-[var(--omed-text)]">
                {t("fin.chat.items", "Items")}
              </span>
              {isEditing && (
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={handleAddItem}
                >
                  {t("fin.chat.addItem", "Add Item")}
                </Button>
              )}
            </div>

            <div className="overflow-x-auto rounded border border-[var(--omed-border-soft)]">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="bg-[var(--omed-surface-muted)] text-[var(--omed-text-secondary)] border-b border-[var(--omed-border-soft)]">
                    <th className="py-2 px-3 font-medium">
                      {t("fin.chat.account", "Account")}
                    </th>
                    <th className="py-2 px-3 font-medium">
                      {t("fin.chat.description", "Description")}
                    </th>
                    <th className="py-2 px-3 font-medium w-16 text-right">
                      {t("fin.chat.quantity", "Qty")}
                    </th>
                    <th className="py-2 px-3 font-medium w-28 text-right">
                      {t("fin.chat.unitPrice", "Unit Price")}
                    </th>
                    <th className="py-2 px-3 font-medium w-28 text-right">
                      {t("fin.chat.amount", "Amount")}
                    </th>
                    {isEditing && <th className="py-2 px-2 w-10 text-center" />}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--omed-border-soft)]">
                  {(draft.items || []).map((item, idx) => (
                    <tr
                      key={
                        (item as { _key?: string })._key ??
                        `item-${item.accountId ?? idx}`
                      }
                      className="hover:bg-[var(--omed-surface-muted)]/50"
                    >
                      <td className="py-2 px-3">
                        {isEditing ? (
                          <Select
                            size="small"
                            className="w-full min-w-[160px]"
                            showSearch
                            optionFilterProp="label"
                            value={item.accountId}
                            onChange={(val) =>
                              handleItemChange(idx, "accountId", val)
                            }
                            options={postableAccountOptions}
                          />
                        ) : (
                          <span className="font-medium text-[var(--omed-text)]">
                            {getAccountLabel(item.accountId)}
                          </span>
                        )}
                      </td>
                      <td className="py-2 px-3">
                        {isEditing ? (
                          <Input
                            size="small"
                            value={item.description || ""}
                            onChange={(e) =>
                              handleItemChange(
                                idx,
                                "description",
                                e.target.value,
                              )
                            }
                          />
                        ) : (
                          <span className="text-[var(--omed-text-secondary)]">
                            {item.description || "-"}
                          </span>
                        )}
                      </td>
                      <td className="py-2 px-3 text-right">
                        {isEditing ? (
                          <InputNumber
                            size="small"
                            min={0}
                            className="w-16"
                            value={Number(item.quantity ?? 1)}
                            onChange={(val) =>
                              handleItemChange(
                                idx,
                                "quantity",
                                String(val ?? 0),
                              )
                            }
                          />
                        ) : (
                          <span>{item.quantity ?? "1"}</span>
                        )}
                      </td>
                      <td className="py-2 px-3 text-right">
                        {isEditing ? (
                          <InputNumber
                            size="small"
                            min={0}
                            className="w-24"
                            value={Number(item.unitPrice ?? 0)}
                            onChange={(val) =>
                              handleItemChange(
                                idx,
                                "unitPrice",
                                String(val ?? 0),
                              )
                            }
                          />
                        ) : (
                          <span>
                            {formatMoney(
                              item.unitPrice || 0,
                              draft.currency || "IDR",
                              lang,
                            )}
                          </span>
                        )}
                      </td>
                      <td className="py-2 px-3 text-right font-medium">
                        {isEditing ? (
                          <InputNumber
                            size="small"
                            min={0}
                            className="w-24"
                            value={Number(item.amount ?? 0)}
                            onChange={(val) =>
                              handleItemChange(idx, "amount", String(val ?? 0))
                            }
                          />
                        ) : (
                          <span>
                            {formatMoney(
                              item.amount || 0,
                              draft.currency || "IDR",
                              lang,
                            )}
                          </span>
                        )}
                      </td>
                      {isEditing && (
                        <td className="py-2 px-2 text-center">
                          <Button
                            type="text"
                            danger
                            size="small"
                            icon={<DeleteOutlined />}
                            onClick={() => handleRemoveItem(idx)}
                          />
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Journal Entries breakdown for Journal variant */}
        {isJournal && (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-[var(--omed-text)]">
                {t("fin.chat.entries", "Journal Entries")}
              </span>
              {isEditing && (
                <Button
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={handleAddEntry}
                >
                  {t("fin.chat.addEntry", "Add Entry")}
                </Button>
              )}
            </div>

            <div className="overflow-x-auto rounded border border-[var(--omed-border-soft)]">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="bg-[var(--omed-surface-muted)] text-[var(--omed-text-secondary)] border-b border-[var(--omed-border-soft)]">
                    <th className="py-2 px-3 font-medium">
                      {t("fin.chat.account", "Account")}
                    </th>
                    <th className="py-2 px-3 font-medium w-24">
                      {t("fin.chat.direction", "Direction")}
                    </th>
                    <th className="py-2 px-3 font-medium w-32 text-right">
                      {t("fin.chat.amount", "Amount")}
                    </th>
                    <th className="py-2 px-3 font-medium">
                      {t("fin.chat.description", "Description")}
                    </th>
                    {isEditing && <th className="py-2 px-2 w-10 text-center" />}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[var(--omed-border-soft)]">
                  {(draft.entries || []).map((entry, idx) => {
                    const isDebit = entry.direction?.toUpperCase() === "DEBIT";
                    return (
                      <tr
                        key={
                          (entry as { _key?: string })._key ??
                          `entry-${entry.accountId ?? idx}`
                        }
                        className="hover:bg-[var(--omed-surface-muted)]/50"
                      >
                        <td className="py-2 px-3">
                          {isEditing ? (
                            <Select
                              size="small"
                              className="w-full min-w-[160px]"
                              showSearch
                              optionFilterProp="label"
                              value={entry.accountId}
                              onChange={(val) =>
                                handleEntryChange(idx, "accountId", val)
                              }
                              options={postableAccountOptions}
                            />
                          ) : (
                            <span className="font-medium text-[var(--omed-text)]">
                              {getAccountLabel(entry.accountId)}
                            </span>
                          )}
                        </td>
                        <td className="py-2 px-3">
                          {isEditing ? (
                            <Select
                              size="small"
                              value={entry.direction?.toUpperCase() || "DEBIT"}
                              onChange={(val) =>
                                handleEntryChange(idx, "direction", val)
                              }
                              options={[
                                {
                                  value: "DEBIT",
                                  label: t("fin.chat.debit", "Debit"),
                                },
                                {
                                  value: "CREDIT",
                                  label: t("fin.chat.credit", "Credit"),
                                },
                              ]}
                            />
                          ) : (
                            <Tag
                              color={isDebit ? "blue" : "default"}
                              className="m-0 text-[10px]"
                            >
                              {isDebit
                                ? t("fin.chat.debit", "Debit")
                                : t("fin.chat.credit", "Credit")}
                            </Tag>
                          )}
                        </td>
                        <td className="py-2 px-3 text-right font-medium">
                          {isEditing ? (
                            <InputNumber
                              size="small"
                              min={0}
                              className="w-28"
                              value={Number(entry.amount ?? 0)}
                              onChange={(val) =>
                                handleEntryChange(
                                  idx,
                                  "amount",
                                  String(val ?? 0),
                                )
                              }
                            />
                          ) : (
                            <span>
                              {formatMoney(
                                entry.amount || 0,
                                draft.currency || "IDR",
                                lang,
                              )}
                            </span>
                          )}
                        </td>
                        <td className="py-2 px-3">
                          {isEditing ? (
                            <Input
                              size="small"
                              value={entry.description || ""}
                              onChange={(e) =>
                                handleEntryChange(
                                  idx,
                                  "description",
                                  e.target.value,
                                )
                              }
                            />
                          ) : (
                            <span className="text-[var(--omed-text-secondary)]">
                              {entry.description || "-"}
                            </span>
                          )}
                        </td>
                        {isEditing && (
                          <td className="py-2 px-2 text-center">
                            <Button
                              type="text"
                              danger
                              size="small"
                              icon={<DeleteOutlined />}
                              onClick={() => handleRemoveEntry(idx)}
                            />
                          </td>
                        )}
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {/* Balance state */}
            <div className="flex items-center justify-between text-xs px-2 py-1 bg-[var(--omed-surface-muted)] rounded">
              <span className="text-[var(--omed-text-secondary)]">
                {t("fin.chat.debit", "Debit")}:{" "}
                <span className="font-medium text-[var(--omed-text)]">
                  {formatMoney(
                    journalBalance.debit,
                    draft.currency || "IDR",
                    lang,
                  )}
                </span>{" "}
                | {t("fin.chat.credit", "Credit")}:{" "}
                <span className="font-medium text-[var(--omed-text)]">
                  {formatMoney(
                    journalBalance.credit,
                    draft.currency || "IDR",
                    lang,
                  )}
                </span>
              </span>
              <span>
                {journalBalance.balanced ? (
                  <Tag color="success" className="m-0">
                    {t("fin.chat.balanced", "Balanced")}
                  </Tag>
                ) : (
                  <Tag color="error" className="m-0">
                    {t("fin.chat.unbalanced", {
                      diff: formatMoney(
                        journalBalance.diff,
                        draft.currency || "IDR",
                        lang,
                      ),
                    })}
                  </Tag>
                )}
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Footer action buttons */}
      {!isReadOnly && (
        <div className="flex items-center justify-end gap-2 px-4 py-3 bg-[var(--omed-surface-muted)] border-t border-[var(--omed-border-soft)]">
          {isEditing ? (
            <>
              <Button size="small" onClick={handleCancelEdit}>
                {t("fin.chat.cancel", "Cancel")}
              </Button>
              <Button
                type="primary"
                size="small"
                icon={<CheckOutlined />}
                onClick={handleSaveEdit}
              >
                {t("fin.chat.saveChanges", "Save changes")}
              </Button>
            </>
          ) : (
            <>
              <Button
                size="small"
                icon={<EditOutlined />}
                onClick={handleStartEdit}
              >
                {t("fin.chat.edit", "Edit")}
              </Button>
              <Popconfirm
                title={t(
                  "fin.chat.discardConfirm",
                  "Are you sure you want to discard this draft proposal?",
                )}
                onConfirm={onDiscard}
              >
                <Button
                  size="small"
                  danger
                  icon={<DeleteOutlined />}
                  loading={isDiscarding}
                >
                  {t("fin.chat.discard", "Discard")}
                </Button>
              </Popconfirm>
              <Button
                type="primary"
                size="small"
                icon={<CheckOutlined />}
                loading={isConfirming}
                onClick={() => onConfirm?.(hasEdited ? draft : undefined)}
              >
                {t("fin.chat.confirm", "Confirm")}
              </Button>
            </>
          )}
        </div>
      )}
    </div>
  );
}
