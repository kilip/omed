import {
  DeleteOutlined,
  EditOutlined,
  LockOutlined,
  PlusOutlined,
  UnlockOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  Button,
  DatePicker,
  Empty,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Table,
  type TableProps,
  Tag,
  Tooltip,
  Typography,
} from "antd";
import dayjs, { type Dayjs } from "dayjs";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { RecordModal } from "~/components/RecordModal";
import { api, unwrapOne, unwrapVoid } from "~/lib/api";
import { getQueryClient } from "~/lib/query-client";
import { useAction } from "~/lib/use-action";
import { requireWorkspaceId } from "~/lib/workspace";
import { periodsQueryOptions } from "../queries";
import type { Route } from "./+types/periods";

const { Text } = Typography;

type Period = components["schemas"]["model.PeriodResponse"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  await getQueryClient().ensureQueryData(periodsQueryOptions(wid));
  return { wid };
}

type FormValues = {
  name: string;
  fiscalYear: number;
  dateRange: [Dayjs, Dayjs];
  description?: string;
};

export function getFiscalYears(periods: Period[]): number[] {
  const years = Array.from(
    new Set(
      periods
        .map((p) => p.fiscalYear)
        .filter((y): y is number => y !== undefined),
    ),
  );
  return years.sort((a, b) => b - a);
}

export function filterPeriods(
  periods: Period[],
  statusFilter: "ALL" | "OPEN" | "CLOSED",
  fiscalYearFilter: number | "ALL",
  searchQuery: string,
): Period[] {
  return periods.filter((p) => {
    if (statusFilter !== "ALL" && p.status !== statusFilter) return false;
    if (fiscalYearFilter !== "ALL" && p.fiscalYear !== fiscalYearFilter)
      return false;
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      const matchName = p.name?.toLowerCase().includes(q);
      const matchDesc = p.description?.toLowerCase().includes(q);
      const matchYear = String(p.fiscalYear).includes(q);
      if (!matchName && !matchDesc && !matchYear) return false;
    }
    return true;
  });
}

export default function Periods({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { data: periods = [] } = useSuspenseQuery(periodsQueryOptions(wid));
  const { t } = useTranslation();
  const { run, pending } = useAction();

  const [editing, setEditing] = useState<Period | "new" | null>(null);
  const [statusFilter, setStatusFilter] = useState<"ALL" | "OPEN" | "CLOSED">(
    "ALL",
  );
  const [fiscalYearFilter, setFiscalYearFilter] = useState<number | "ALL">(
    "ALL",
  );
  const [searchQuery, setSearchQuery] = useState("");

  const fiscalYears = useMemo(() => getFiscalYears(periods), [periods]);

  const filteredPeriods = useMemo(
    () => filterPeriods(periods, statusFilter, fiscalYearFilter, searchQuery),
    [periods, statusFilter, fiscalYearFilter, searchQuery],
  );

  const savePeriod = async (values: FormValues) => {
    const payload = {
      name: values.name.trim(),
      fiscalYear: values.fiscalYear,
      startDate: values.dateRange[0].format("YYYY-MM-DD"),
      endDate: values.dateRange[1].format("YYYY-MM-DD"),
      description: values.description?.trim() || undefined,
    };

    if (editing === "new") {
      return run(
        () =>
          unwrapOne(
            api.POST("/workspaces/{wid}/fin/periods", {
              params: { path: { wid } },
              body: payload,
            }),
          ),
        "fin.periods.createSuccess",
      );
    }

    if (editing && typeof editing === "object" && editing.id) {
      const periodId = editing.id;
      return run(
        () =>
          unwrapOne(
            api.PUT("/workspaces/{wid}/fin/periods/{id}", {
              params: { path: { wid, id: periodId } },
              body: payload,
            }),
          ),
        "fin.periods.updateSuccess",
      );
    }

    return false;
  };

  const closePeriod = (p: Period) => {
    if (!p.id) return Promise.resolve(false);
    const periodId = p.id;
    return run(
      () =>
        unwrapOne(
          api.POST("/workspaces/{wid}/fin/periods/{id}/close", {
            params: { path: { wid, id: periodId } },
          }),
        ),
      "fin.periods.closeSuccess",
    );
  };

  const reopenPeriod = (p: Period) => {
    if (!p.id) return Promise.resolve(false);
    const periodId = p.id;
    return run(
      () =>
        unwrapOne(
          api.POST("/workspaces/{wid}/fin/periods/{id}/reopen", {
            params: { path: { wid, id: periodId } },
          }),
        ),
      "fin.periods.reopenSuccess",
    );
  };

  const deletePeriod = (p: Period) => {
    if (!p.id) return Promise.resolve(false);
    const periodId = p.id;
    return run(
      () =>
        unwrapVoid(
          api.DELETE("/workspaces/{wid}/fin/periods/{id}", {
            params: { path: { wid, id: periodId } },
          }),
        ),
      "fin.periods.deleteSuccess",
    );
  };

  const currentYear = new Date().getFullYear();
  const initialFormValues: Partial<FormValues> =
    editing && editing !== "new"
      ? {
          name: editing.name,
          fiscalYear: editing.fiscalYear,
          dateRange: [
            dayjs(editing.startDate as unknown as string),
            dayjs(editing.endDate as unknown as string),
          ],
          description: editing.description ?? "",
        }
      : {
          name: `FY ${currentYear}`,
          fiscalYear: currentYear,
          dateRange: [
            dayjs(`${currentYear}-01-01`),
            dayjs(`${currentYear}-12-31`),
          ],
          description: "",
        };

  const columns: TableProps<Period>["columns"] = [
    {
      title: t("fin.periods.name"),
      dataIndex: "name",
      key: "name",
      render: (name: string, record: Period) => (
        <div className="flex flex-col">
          <Text strong>{name}</Text>
          {record.description && (
            <Text type="secondary" className="text-xs">
              {record.description}
            </Text>
          )}
        </div>
      ),
    },
    {
      title: t("fin.periods.fiscalYear"),
      dataIndex: "fiscalYear",
      key: "fiscalYear",
      width: 130,
    },
    {
      title: t("fin.periods.dateRange"),
      key: "dateRange",
      width: 240,
      render: (_: unknown, record: Period) => (
        <Text className="font-mono text-xs">
          {record.startDate as unknown as string} ~{" "}
          {record.endDate as unknown as string}
        </Text>
      ),
    },
    {
      title: t("fin.periods.status"),
      dataIndex: "status",
      key: "status",
      width: 120,
      render: (status: string) => (
        <Tag color={status === "OPEN" ? "green" : "default"}>
          {status === "OPEN"
            ? t("fin.periods.statusOpen")
            : t("fin.periods.statusClosed")}
        </Tag>
      ),
    },
    {
      title: t("common.actions"),
      key: "actions",
      width: 140,
      align: "right",
      render: (_: unknown, record: Period) => {
        const isOpen = record.status === "OPEN";
        return (
          <Space size="small">
            {isOpen && (
              <Tooltip title={t("fin.periods.edit")}>
                <Button
                  type="text"
                  size="small"
                  icon={<EditOutlined />}
                  aria-label={t("fin.periods.edit")}
                  onClick={() => setEditing(record)}
                />
              </Tooltip>
            )}

            {isOpen ? (
              <Popconfirm
                title={t("fin.periods.closeConfirm")}
                okText={t("fin.periods.close")}
                cancelText={t("common.cancel")}
                okButtonProps={{ danger: true }}
                onConfirm={() => closePeriod(record)}
              >
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<LockOutlined />}
                  aria-label={t("fin.periods.close")}
                />
              </Popconfirm>
            ) : (
              <Popconfirm
                title={t("fin.periods.reopenConfirm")}
                okText={t("fin.periods.reopen")}
                cancelText={t("common.cancel")}
                onConfirm={() => reopenPeriod(record)}
              >
                <Button
                  type="text"
                  size="small"
                  icon={<UnlockOutlined />}
                  aria-label={t("fin.periods.reopen")}
                />
              </Popconfirm>
            )}

            <Popconfirm
              title={t("fin.periods.deleteConfirm")}
              okText={t("common.delete")}
              cancelText={t("common.cancel")}
              okButtonProps={{ danger: true }}
              onConfirm={() => deletePeriod(record)}
            >
              <Button
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
                aria-label={t("common.delete")}
              />
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  return (
    <Page
      title={t("fin.periods.title")}
      actions={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setEditing("new")}
        >
          {t("fin.periods.new")}
        </Button>
      }
    >
      <Panel>
        {/* Filters */}
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <Input.Search
            allowClear
            placeholder={t("common.search")}
            className="w-full max-w-xs"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          <div className="flex flex-wrap items-center gap-3">
            <Select
              className="w-36"
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { value: "ALL", label: t("fin.periods.allStatuses") },
                { value: "OPEN", label: t("fin.periods.statusOpen") },
                { value: "CLOSED", label: t("fin.periods.statusClosed") },
              ]}
            />
            {fiscalYears.length > 0 && (
              <Select
                className="w-36"
                value={fiscalYearFilter}
                onChange={setFiscalYearFilter}
                options={[
                  { value: "ALL", label: t("fin.periods.allYears") },
                  ...fiscalYears.map((yr) => ({
                    value: yr,
                    label: String(yr),
                  })),
                ]}
              />
            )}
          </div>
        </div>

        {/* Table */}
        <Table<Period>
          rowKey="id"
          columns={columns}
          dataSource={filteredPeriods}
          loading={pending}
          pagination={
            filteredPeriods.length > 10
              ? { pageSize: 10, showSizeChanger: true }
              : false
          }
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={t("fin.periods.emptyTitle")}
              >
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setEditing("new")}
                >
                  {t("fin.periods.new")}
                </Button>
              </Empty>
            ),
          }}
        />
      </Panel>

      {/* Create / Edit Modal */}
      {editing && (
        <RecordModal<FormValues>
          open
          title={
            editing === "new" ? t("fin.periods.new") : t("fin.periods.edit")
          }
          initialValues={initialFormValues}
          submitting={pending}
          onClose={() => setEditing(null)}
          onSubmit={savePeriod}
        >
          {(form) => (
            <>
              <Form.Item
                name="fiscalYear"
                label={t("fin.periods.fiscalYear")}
                rules={[{ required: true }]}
              >
                <InputNumber
                  className="w-full"
                  min={2000}
                  max={2100}
                  onChange={(val) => {
                    if (val && editing === "new") {
                      form.setFieldsValue({
                        name: `FY ${val}`,
                        dateRange: [
                          dayjs(`${val}-01-01`),
                          dayjs(`${val}-12-31`),
                        ],
                      });
                    }
                  }}
                />
              </Form.Item>

              <Form.Item
                name="name"
                label={t("fin.periods.name")}
                rules={[{ required: true, whitespace: true }]}
              >
                <Input placeholder={t("fin.setup.periodNamePlaceholder")} />
              </Form.Item>

              <Form.Item
                name="dateRange"
                label={t("fin.periods.dateRange")}
                rules={[{ required: true }]}
              >
                <DatePicker.RangePicker className="w-full" />
              </Form.Item>

              <Form.Item
                name="description"
                label={t("fin.periods.description")}
              >
                <Input.TextArea rows={3} />
              </Form.Item>
            </>
          )}
        </RecordModal>
      )}
    </Page>
  );
}
