import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { Alert, Button, Form, Input, InputNumber } from "antd";
import { useTranslation } from "react-i18next";
import { formatMoney, isBalanced, roundToCurrency } from "~/lib/format";
import { AccountSelect } from "./account-select";

type Account = components["schemas"]["model.AccountResponse"];
type CreatePostingRequest = components["schemas"]["model.CreatePostingRequest"];

export type PostingLineValue = {
  id?: number;
  accountId?: number;
  debit?: number;
  credit?: number;
  description?: string;
  baseAmount?: number;
};

export type PostingsEditorProps = {
  name?: string;
  accounts: Account[];
  currency?: string;
  exchangeRate?: number;
  disabled?: boolean;
};

/**
 * Converts form line values (with debit and credit fields) to API CreatePostingRequest array.
 */
export function postingLinesToApi(
  lines: PostingLineValue[] | undefined,
  exchangeRate = 1,
  currency = "IDR",
): CreatePostingRequest[] {
  if (!lines) return [];
  const rate = exchangeRate > 0 ? exchangeRate : 1;

  return lines
    .filter(
      (line) =>
        line.accountId && ((line.debit ?? 0) > 0 || (line.credit ?? 0) > 0),
    )
    .map((line) => {
      const isDebit = (line.debit ?? 0) > 0;
      const amount = isDebit ? (line.debit ?? 0) : (line.credit ?? 0);
      const roundedAmount = roundToCurrency(amount, currency);
      const baseAmount = roundToCurrency(roundedAmount * rate, "IDR");

      return {
        accountId: line.accountId as number,
        direction: (isDebit ? "DEBIT" : "CREDIT") as "DEBIT" | "CREDIT",
        amount: roundedAmount,
        baseAmount,
        description: line.description,
      };
    });
}

/**
 * Converts API postings or initial entry postings to UI form line values.
 */
export function apiToPostingLines(
  postings:
    | (components["schemas"]["model.PostingResponse"] | CreatePostingRequest)[]
    | undefined,
  exchangeRate = 1,
): PostingLineValue[] {
  if (!postings || postings.length === 0) {
    return [
      {
        accountId: undefined,
        debit: undefined,
        credit: undefined,
        description: "",
      },
      {
        accountId: undefined,
        debit: undefined,
        credit: undefined,
        description: "",
      },
    ];
  }

  const rate = exchangeRate > 0 ? exchangeRate : 1;

  return postings.map((p) => {
    const isDebit = p.direction === "DEBIT";
    const amt = p.amount ?? 0;
    const baseAmt = p.baseAmount ?? roundToCurrency(amt * rate, "IDR");

    return {
      id: "id" in p ? p.id : undefined,
      accountId: p.accountId,
      debit: isDebit ? amt : undefined,
      credit: !isDebit ? amt : undefined,
      description: p.description ?? "",
      baseAmount: baseAmt,
    };
  });
}

/**
 * Calculates totals and balance check for posting lines.
 */
export function calculatePostingsSummary(
  lines: PostingLineValue[] | undefined,
  currency = "IDR",
  exchangeRate = 1,
) {
  const safeLines = lines ?? [];
  const rate = exchangeRate > 0 ? exchangeRate : 1;

  let totalDebit = 0;
  let totalCredit = 0;
  let totalBaseDebit = 0;
  let totalBaseCredit = 0;
  let validLineCount = 0;

  for (const line of safeLines) {
    const d = line?.debit ?? 0;
    const c = line?.credit ?? 0;
    if (line?.accountId && (d > 0 || c > 0)) {
      validLineCount++;
    }
    totalDebit += d;
    totalCredit += c;
    totalBaseDebit += roundToCurrency(d * rate, "IDR");
    totalBaseCredit += roundToCurrency(c * rate, "IDR");
  }

  const roundedDebit = roundToCurrency(totalDebit, currency);
  const roundedCredit = roundToCurrency(totalCredit, currency);
  const balanced =
    validLineCount >= 2 &&
    isBalanced(roundedDebit, roundedCredit, currency) &&
    roundedDebit > 0;

  return {
    totalDebit: roundedDebit,
    totalCredit: roundedCredit,
    totalBaseDebit: roundToCurrency(totalBaseDebit, "IDR"),
    totalBaseCredit: roundToCurrency(totalBaseCredit, "IDR"),
    validLineCount,
    balanced,
  };
}

export function PostingsEditor({
  name = "postings",
  accounts,
  currency = "IDR",
  exchangeRate = 1,
  disabled = false,
}: PostingsEditorProps) {
  const { t, i18n } = useTranslation();
  const form = Form.useFormInstance();
  const lines =
    (Form.useWatch(name, form) as PostingLineValue[] | undefined) ?? [];

  const summary = calculatePostingsSummary(lines, currency, exchangeRate);
  const isMultiCurrency = currency.toUpperCase() !== "IDR";

  return (
    <div className="flex flex-col gap-4">
      <Form.List name={name}>
        {(fields, { add, remove }) => (
          <div className="flex flex-col gap-2">
            <div className="grid grid-cols-12 gap-2 text-xs font-semibold text-muted-foreground px-1">
              <div className="col-span-4">
                {t("fin.journal.account", "Account")}
              </div>
              <div className="col-span-2 text-right">
                {t("fin.journal.debit", "Debit")}
              </div>
              <div className="col-span-2 text-right">
                {t("fin.journal.credit", "Credit")}
              </div>
              {isMultiCurrency && (
                <div className="col-span-2 text-right">Base (IDR)</div>
              )}
              <div className={isMultiCurrency ? "col-span-1" : "col-span-3"}>
                {t("fin.journal.label", "Description")}
              </div>
              <div className="col-span-1 text-center" />
            </div>

            {fields.map((field) => {
              const currentLine = lines[field.name];
              const debit = currentLine?.debit ?? 0;
              const credit = currentLine?.credit ?? 0;
              const activeAmount = debit > 0 ? debit : credit;
              const calcBase =
                activeAmount > 0
                  ? roundToCurrency(
                      activeAmount * (exchangeRate > 0 ? exchangeRate : 1),
                      "IDR",
                    )
                  : 0;

              return (
                <div
                  key={field.key}
                  className="grid grid-cols-12 gap-2 items-center bg-card p-1.5 rounded border border-border"
                >
                  <div className="col-span-4">
                    <Form.Item
                      name={[field.name, "accountId"]}
                      className="m-0"
                      rules={[
                        {
                          required: true,
                          message: t("common.required", "Required"),
                        },
                      ]}
                    >
                      <AccountSelect
                        accounts={accounts}
                        disabled={disabled}
                        placeholder={t("fin.journal.account", "Select account")}
                      />
                    </Form.Item>
                  </div>

                  <div className="col-span-2">
                    <Form.Item name={[field.name, "debit"]} className="m-0">
                      <InputNumber
                        min={0}
                        controls={false}
                        disabled={disabled || credit > 0}
                        placeholder="0"
                        className="w-full text-right font-mono"
                        onChange={() => {
                          if (form && (lines[field.name]?.credit ?? 0) > 0) {
                            form.setFieldValue(
                              [name, field.name, "credit"],
                              undefined,
                            );
                          }
                        }}
                      />
                    </Form.Item>
                  </div>

                  <div className="col-span-2">
                    <Form.Item name={[field.name, "credit"]} className="m-0">
                      <InputNumber
                        min={0}
                        controls={false}
                        disabled={disabled || debit > 0}
                        placeholder="0"
                        className="w-full text-right font-mono"
                        onChange={() => {
                          if (form && (lines[field.name]?.debit ?? 0) > 0) {
                            form.setFieldValue(
                              [name, field.name, "debit"],
                              undefined,
                            );
                          }
                        }}
                      />
                    </Form.Item>
                  </div>

                  {isMultiCurrency && (
                    <div className="col-span-2 text-right font-mono text-sm text-muted-foreground px-2">
                      {calcBase > 0
                        ? formatMoney(calcBase, "IDR", i18n.language)
                        : "-"}
                    </div>
                  )}

                  <div
                    className={isMultiCurrency ? "col-span-1" : "col-span-3"}
                  >
                    <Form.Item
                      name={[field.name, "description"]}
                      className="m-0"
                    >
                      <Input
                        disabled={disabled}
                        placeholder={t("fin.journal.label", "Description")}
                      />
                    </Form.Item>
                  </div>

                  <div className="col-span-1 text-center">
                    {!disabled && (
                      <Button
                        type="text"
                        danger
                        icon={<MinusCircleOutlined />}
                        disabled={fields.length <= 2}
                        onClick={() => remove(field.name)}
                        aria-label={t("common.delete", "Delete line")}
                      />
                    )}
                  </div>
                </div>
              );
            })}

            {!disabled && (
              <div className="mt-2">
                <Button
                  type="dashed"
                  icon={<PlusOutlined />}
                  onClick={() =>
                    add({
                      accountId: undefined,
                      debit: undefined,
                      credit: undefined,
                      description: "",
                    })
                  }
                  block
                >
                  {t("fin.journal.addLine", "Add line")}
                </Button>
              </div>
            )}
          </div>
        )}
      </Form.List>

      <div className="rounded border border-border p-3 bg-muted/30 flex flex-col gap-2">
        <div className="flex justify-between items-center text-sm">
          <span className="font-medium text-muted-foreground">
            {t("fin.journal.debit", "Total Debit")}:
          </span>
          <span className="font-mono font-semibold">
            {formatMoney(summary.totalDebit, currency, i18n.language)}
          </span>
        </div>
        <div className="flex justify-between items-center text-sm">
          <span className="font-medium text-muted-foreground">
            {t("fin.journal.credit", "Total Credit")}:
          </span>
          <span className="font-mono font-semibold">
            {formatMoney(summary.totalCredit, currency, i18n.language)}
          </span>
        </div>

        {isMultiCurrency && (
          <div className="flex justify-between items-center text-xs text-muted-foreground border-t border-border pt-1">
            <span>Base Totals (IDR):</span>
            <span className="font-mono">
              Dr {formatMoney(summary.totalBaseDebit, "IDR", i18n.language)} /
              Cr {formatMoney(summary.totalBaseCredit, "IDR", i18n.language)}
            </span>
          </div>
        )}

        <div className="pt-2">
          {summary.balanced ? (
            <Alert
              type="success"
              showIcon
              message={t("fin.journal.balanced", "Balanced")}
              className="py-1 px-3 text-xs"
            />
          ) : (
            <Alert
              type="warning"
              showIcon
              message={
                summary.validLineCount < 2
                  ? t(
                      "fin.journal.mustBalance",
                      "Debit and credit must be equal, with at least two lines.",
                    )
                  : `${t("fin.journal.unbalanced", "Not balanced")} (Diff: ${formatMoney(
                      Math.abs(summary.totalDebit - summary.totalCredit),
                      currency,
                      i18n.language,
                    )})`
              }
              className="py-1 px-3 text-xs"
            />
          )}
        </div>
      </div>
    </div>
  );
}
