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
import { Chatter, type ChatterMessage } from "~/components/Chatter";
import { FormView } from "~/components/FormView";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { RecordModal } from "~/components/RecordModal";
import { StatusBar } from "~/components/StatusBar";
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
  const [openId, setOpenId] = useState<number | undefined>();
  const [creating, setCreating] = useState(false);
  const [messages, setMessages] = useState<Record<number, ChatterMessage[]>>(
    {},
  );

  const money = (value: number) => formatIDR(value, i18n.language);
  const opened = entries.find((e) => e.id === openId);

  const addMessage = (
    entryId: number,
    kind: ChatterMessage["kind"],
    body: string,
  ) =>
    setMessages((all) => {
      const current = all[entryId] ?? [];
      const id = Math.max(0, ...current.map((m) => m.id)) + 1;
      const message: ChatterMessage = {
        id,
        kind,
        body,
        author: t("chatter.you"),
        at: new Date().toLocaleString(i18n.language),
      };
      return { ...all, [entryId]: [message, ...current] };
    });

  const setStatus = (entry: JournalEntry, status: JournalEntry["status"]) => {
    setEntries((all) =>
      all.map((e) => (e.id === entry.id ? { ...e, status } : e)),
    );
    addMessage(
      entry.id,
      "tracking",
      t(
        status === "POSTED"
          ? "fin.journal.trackPosted"
          : "fin.journal.trackDraft",
      ),
    );
  };

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
    setOpenId(id);
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

  const newButton = (
    <Button
      type="primary"
      icon={<PlusOutlined />}
      onClick={() => setCreating(true)}
    >
      {t("fin.journal.new")}
    </Button>
  );

  return (
    <Page
      title={opened ? opened.number : t("fin.menu.journal")}
      trail={
        opened
          ? [
              {
                title: t("fin.menu.journal"),
                onClick: () => setOpenId(undefined),
              },
            ]
          : []
      }
      count={opened ? undefined : entries.length}
      actions={newButton}
    >
      <Alert type="info" showIcon message={t("common.provisionalNotice")} />

      {opened ? (
        <FormView
          title={opened.number}
          buttons={
            opened.status === "DRAFT" ? (
              <Button
                type="primary"
                onClick={() => setStatus(opened, "POSTED")}
              >
                {t("fin.journal.post")}
              </Button>
            ) : (
              <Button onClick={() => setStatus(opened, "DRAFT")}>
                {t("fin.journal.resetDraft")}
              </Button>
            )
          }
          statusbar={
            <StatusBar
              label={t("common.status")}
              current={opened.status}
              steps={[
                { value: "DRAFT", label: t("fin.journal.DRAFT") },
                { value: "POSTED", label: t("fin.journal.POSTED") },
              ]}
            />
          }
          chatter={
            <Chatter
              messages={messages[opened.id] ?? []}
              onPost={(kind, body) => addMessage(opened.id, kind, body)}
            />
          }
        >
          <div
            className="mb-2 text-[13px]"
            style={{ color: "var(--omed-text-secondary)" }}
          >
            {formatDate(opened.date, i18n.language)} · {opened.reference}
          </div>
          <Table<JournalLine>
            size="small"
            rowKey={(l) => `${l.account}-${l.debit}-${l.credit}`}
            columns={lineColumns}
            dataSource={opened.lines}
            pagination={false}
            scroll={{ x: "max-content" }}
            summary={() => (
              <Table.Summary.Row>
                <Table.Summary.Cell index={0} colSpan={2}>
                  <strong>{t("common.total")}</strong>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={2} align="right">
                  <strong className="num">
                    {money(sumDebit(opened.lines))}
                  </strong>
                </Table.Summary.Cell>
                <Table.Summary.Cell index={3} align="right">
                  <strong className="num">
                    {money(sumCredit(opened.lines))}
                  </strong>
                </Table.Summary.Cell>
              </Table.Summary.Row>
            )}
          />
        </FormView>
      ) : (
        <Panel strong>
          <Table<JournalEntry>
            size="small"
            rowKey="id"
            columns={listColumns}
            dataSource={entries}
            pagination={false}
            scroll={{ x: "max-content" }}
            onRow={(entry) => ({
              onClick: () => setOpenId(entry.id),
              style: { cursor: "pointer" },
            })}
          />
        </Panel>
      )}

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
