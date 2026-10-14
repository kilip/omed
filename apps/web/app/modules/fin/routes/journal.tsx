import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import {
  Alert,
  App,
  Button,
  DatePicker,
  Form,
  type FormInstance,
  Input,
  InputNumber,
  Select,
  Table,
  type TableProps,
  Tag,
} from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { RecordModal } from "~/components/RecordModal";
import { formatDate, formatIDR } from "~/lib/format";
import {
  accountName,
  type JournalEntry,
  type JournalLine,
  mockAccounts,
  mockEntries,
  sumCredit,
  sumDebit,
} from "../mock";

type LineValue = { account?: string; debit?: number; credit?: number };

type FormValues = {
  date: { format: (pattern: string) => string };
  reference: string;
  lines: LineValue[];
};

function LinesEditor({ form }: { form: FormInstance<FormValues> }) {
  const { t, i18n } = useTranslation();
  const lines = Form.useWatch("lines", form) as LineValue[] | undefined;

  const debit = (lines ?? []).reduce((sum, l) => sum + (l?.debit ?? 0), 0);
  const credit = (lines ?? []).reduce((sum, l) => sum + (l?.credit ?? 0), 0);
  const balanced = debit === credit && debit > 0;

  return (
    <>
      <Form.List name="lines">
        {(fields, { add, remove }) => (
          <div className="flex flex-col gap-2">
            {fields.map((field) => (
              <div key={field.key} className="flex flex-wrap items-start gap-2">
                <Form.Item
                  name={[field.name, "account"]}
                  className="m-0 min-w-[160px] flex-[2_1_160px]"
                  rules={[{ required: true }]}
                >
                  <Select
                    showSearch
                    optionFilterProp="label"
                    placeholder={t("fin.journal.account")}
                    options={mockAccounts.map((a) => ({
                      value: a.code,
                      label: `${a.code} ${a.name}`,
                    }))}
                  />
                </Form.Item>
                <Form.Item
                  name={[field.name, "debit"]}
                  className="m-0 w-[110px]"
                >
                  <InputNumber
                    min={0}
                    controls={false}
                    placeholder={t("fin.journal.debit")}
                    style={{ width: "100%" }}
                  />
                </Form.Item>
                <Form.Item
                  name={[field.name, "credit"]}
                  className="m-0 w-[110px]"
                >
                  <InputNumber
                    min={0}
                    controls={false}
                    placeholder={t("fin.journal.credit")}
                    style={{ width: "100%" }}
                  />
                </Form.Item>
                <Button
                  type="text"
                  icon={<MinusCircleOutlined />}
                  aria-label={t("common.delete")}
                  disabled={fields.length <= 2}
                  onClick={() => remove(field.name)}
                />
              </div>
            ))}
            <Button
              type="dashed"
              icon={<PlusOutlined />}
              onClick={() => add({})}
            >
              {t("fin.journal.addLine")}
            </Button>
          </div>
        )}
      </Form.List>
      <div className="mt-3 flex flex-wrap items-center justify-between gap-2 text-[13px]">
        <span className="num">
          {t("fin.journal.debit")} {formatIDR(debit, i18n.language)}
          {" · "}
          {t("fin.journal.credit")} {formatIDR(credit, i18n.language)}
        </span>
        <Tag color={balanced ? "success" : "warning"}>
          {balanced ? t("fin.journal.balanced") : t("fin.journal.unbalanced")}
        </Tag>
      </div>
    </>
  );
}

export default function Journal() {
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();
  const [entries, setEntries] = useState<JournalEntry[]>(mockEntries);
  const [selectedId, setSelectedId] = useState<number | undefined>(
    mockEntries[0]?.id,
  );
  const [creating, setCreating] = useState(false);

  const money = (value: number) => formatIDR(value, i18n.language);
  const selected = entries.find((e) => e.id === selectedId);

  const create = async (values: FormValues) => {
    const lines: JournalLine[] = values.lines
      .filter((l) => l.account && (l.debit || l.credit))
      .map((l) => ({
        account: l.account ?? "",
        label: values.reference,
        debit: l.debit ?? 0,
        credit: l.credit ?? 0,
      }));

    if (
      lines.length < 2 ||
      sumDebit(lines) !== sumCredit(lines) ||
      sumDebit(lines) === 0
    ) {
      message.error(t("fin.journal.mustBalance"));
      return false;
    }

    const id = Math.max(0, ...entries.map((e) => e.id)) + 1;
    const entry: JournalEntry = {
      id,
      number: `JE/2026/${String(id).padStart(4, "0")}`,
      date: values.date.format("YYYY-MM-DD"),
      reference: values.reference,
      status: "DRAFT",
      lines,
    };
    setEntries([entry, ...entries]);
    setSelectedId(id);
    message.success(t("common.saved"));
    return true;
  };

  const listColumns: TableProps<JournalEntry>["columns"] = [
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

  const lineColumns: TableProps<JournalLine>["columns"] = [
    {
      title: t("fin.journal.account"),
      render: (_, l) => `${l.account} ${accountName(l.account)}`,
    },
    { title: t("fin.journal.label"), dataIndex: "label" },
    {
      title: t("fin.journal.debit"),
      align: "right",
      render: (_, l) =>
        l.debit ? <span className="num">{money(l.debit)}</span> : "",
    },
    {
      title: t("fin.journal.credit"),
      align: "right",
      render: (_, l) =>
        l.credit ? <span className="num">{money(l.credit)}</span> : "",
    },
  ];

  return (
    <Page
      title={t("fin.menu.journal")}
      actions={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreating(true)}
        >
          {t("fin.journal.new")}
        </Button>
      }
    >
      <Alert type="info" showIcon message={t("common.provisionalNotice")} />

      <div className="flex flex-wrap items-start gap-3">
        <Panel strong className="flex-[3_1_480px]">
          <Table<JournalEntry>
            size="small"
            rowKey="id"
            columns={listColumns}
            dataSource={entries}
            pagination={false}
            scroll={{ x: "max-content" }}
            rowSelection={{
              type: "radio",
              selectedRowKeys: selectedId === undefined ? [] : [selectedId],
              onChange: (keys) => setSelectedId(Number(keys[0])),
            }}
          />
        </Panel>

        {selected ? (
          <Panel
            strong
            className="flex-[2_1_380px]"
            title={selected.number}
            extra={
              <Tag color={selected.status === "POSTED" ? "success" : "default"}>
                {t(`fin.journal.${selected.status}`)}
              </Tag>
            }
          >
            <div
              className="mb-2 text-[13px]"
              style={{ color: "var(--omed-text-secondary)" }}
            >
              {formatDate(selected.date, i18n.language)} · {selected.reference}
            </div>
            <Table<JournalLine>
              size="small"
              rowKey={(l) => `${l.account}-${l.debit}-${l.credit}`}
              columns={lineColumns}
              dataSource={selected.lines}
              pagination={false}
              scroll={{ x: "max-content" }}
              summary={() => (
                <Table.Summary.Row>
                  <Table.Summary.Cell index={0} colSpan={2}>
                    <strong>{t("common.total")}</strong>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={2} align="right">
                    <strong className="num">
                      {money(sumDebit(selected.lines))}
                    </strong>
                  </Table.Summary.Cell>
                  <Table.Summary.Cell index={3} align="right">
                    <strong className="num">
                      {money(sumCredit(selected.lines))}
                    </strong>
                  </Table.Summary.Cell>
                </Table.Summary.Row>
              )}
            />
          </Panel>
        ) : null}
      </div>

      <RecordModal<FormValues>
        variant="drawer"
        open={creating}
        title={t("fin.journal.new")}
        initialValues={{ lines: [{}, {}] }}
        onClose={() => setCreating(false)}
        onSubmit={create}
      >
        {(form) => (
          <>
            <Form.Item
              name="date"
              label={t("common.date")}
              rules={[{ required: true }]}
            >
              <DatePicker style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item
              name="reference"
              label={t("fin.journal.reference")}
              rules={[{ required: true }]}
            >
              <Input />
            </Form.Item>
            <LinesEditor form={form} />
          </>
        )}
      </RecordModal>
    </Page>
  );
}
