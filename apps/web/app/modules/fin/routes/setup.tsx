import {
  BankOutlined,
  CheckCircleFilled,
  CreditCardOutlined,
  DollarOutlined,
  RocketOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Input,
  InputNumber,
  Radio,
  Row,
  Segmented,
  Space,
  Spin,
  Table,
  type TableProps,
  Tag,
  Typography,
} from "antd";
import dayjs from "dayjs";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { redirect, useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { ApiError, api, unwrapList, unwrapOne } from "~/lib/api";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  accountKeys,
  periodKeys,
  templatePreviewQueryOptions,
} from "../queries";
import type { Route } from "./+types/setup";

type TemplateAccount =
  components["schemas"]["model.AccountTemplateItemResponse"];

const { Text, Title, Paragraph } = Typography;

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const accounts = await unwrapList(
    api.POST("/workspaces/{wid}/fin/accounts/search", {
      params: { path: { wid } },
      body: {},
    }),
  );
  if (accounts.length > 0) {
    return redirect("/fin/accounts");
  }
  return { wid };
}

export default function Setup({ loaderData }: Route.ComponentProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [pending, setPending] = useState(false);

  const [template, setTemplate] = useState("freelancer");
  const [lang, setLang] = useState<"id" | "en">(
    i18n.language.startsWith("en") ? "en" : "id",
  );

  const [searchQuery, setSearchQuery] = useState("");
  const [selectedType, setSelectedType] = useState<string>("ALL");

  const { data: previewData, isLoading: previewLoading } = useQuery(
    templatePreviewQueryOptions(wid, template, lang),
  );

  const currentYear = new Date().getFullYear();
  const [fiscalYear, setFiscalYear] = useState<number>(currentYear);
  const [periodName, setPeriodName] = useState<string>(`FY ${currentYear}`);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs(`${currentYear}-01-01`),
    dayjs(`${currentYear}-12-31`),
  ]);
  const [description, setDescription] = useState<string>("");

  const filteredAccounts = useMemo(() => {
    const list = previewData?.accounts ?? [];
    return list.filter((item) => {
      const matchType = selectedType === "ALL" || item.type === selectedType;
      const q = searchQuery.toLowerCase().trim();
      const matchQuery =
        !q ||
        item.code?.toLowerCase().includes(q) ||
        item.name?.toLowerCase().includes(q) ||
        item.description?.toLowerCase().includes(q);
      return matchType && matchQuery;
    });
  }, [previewData, selectedType, searchQuery]);

  const applyTemplate = async () => {
    if (!periodName.trim()) {
      message.error(`${t("fin.setup.periodName")} required`);
      return;
    }
    if (!dateRange?.[0] || !dateRange[1]) {
      message.error(`${t("fin.setup.periodDates")} required`);
      return;
    }

    setPending(true);
    try {
      const res = await unwrapOne(
        api.POST("/workspaces/{wid}/fin/accounts/seed", {
          params: { path: { wid } },
          body: { template, lang },
        }),
      );

      try {
        await unwrapOne(
          api.POST("/workspaces/{wid}/fin/periods", {
            params: { path: { wid } },
            body: {
              name: periodName.trim(),
              fiscalYear,
              startDate: dateRange[0].format("YYYY-MM-DD"),
              endDate: dateRange[1].format("YYYY-MM-DD"),
              description: description.trim() || undefined,
            },
          }),
        );
      } catch (err) {
        if (
          !(
            err instanceof ApiError &&
            (err.code === "PERIOD_OVERLAP" || err.status === 409)
          )
        ) {
          throw err;
        }
      }
      message.success(t("fin.setup.success", { count: res?.count ?? 50 }));
      queryClient.removeQueries({ queryKey: accountKeys.all(wid) });
      queryClient.removeQueries({ queryKey: periodKeys.all(wid) });
      navigate("/fin/accounts");
    } catch (error) {
      message.error(error instanceof Error ? error.message : t("common.error"));
    } finally {
      setPending(false);
    }
  };

  const previewColumns: TableProps<TemplateAccount>["columns"] = [
    {
      title: t("fin.accounts.code"),
      dataIndex: "code",
      key: "code",
      width: 100,
      render: (code: string) => (
        <code
          className="font-mono font-semibold"
          style={{ color: "var(--omed-primary-text)" }}
        >
          {code}
        </code>
      ),
    },
    {
      title: t("fin.accounts.name"),
      dataIndex: "name",
      key: "name",
      render: (name: string, record: TemplateAccount) => (
        <div className="flex flex-col">
          <span
            className={`font-medium ${!record.isPostable ? "text-base font-semibold" : ""}`}
          >
            {name}
          </span>
          {record.description && (
            <span className="text-xs text-muted-foreground">
              {record.description}
            </span>
          )}
        </div>
      ),
    },
    {
      title: t("fin.accounts.type"),
      dataIndex: "type",
      key: "type",
      width: 120,
      render: (type?: string) => {
        if (!type) return null;
        const colorMap: Record<string, string> = {
          ASSET: "blue",
          LIABILITY: "orange",
          EQUITY: "purple",
          REVENUE: "green",
          EXPENSE: "red",
        };
        return (
          <Tag color={colorMap[type] ?? "default"}>
            {t(`fin.types.${type}`)}
          </Tag>
        );
      },
    },
    {
      title: t("fin.accounts.postable"),
      dataIndex: "isPostable",
      key: "isPostable",
      width: 120,
      render: (isPostable?: boolean) =>
        isPostable ? (
          <Tag color="cyan">{t("fin.accounts.postable")}</Tag>
        ) : (
          <Tag>{t("fin.accounts.parent")}</Tag>
        ),
    },
  ];

  return (
    <Page title={t("fin.setup.title")}>
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6 py-2">
        <div>
          <Title level={4} style={{ marginBottom: 4 }}>
            {t("fin.setup.title")}
          </Title>
          <Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {t("fin.setup.subtitle")}
          </Paragraph>
        </div>

        {/* Template Selection */}
        <Panel strong title={t("fin.setup.template")}>
          <div className="grid gap-4">
            <Card
              hoverable
              className="relative cursor-pointer transition-all"
              style={{
                borderColor:
                  template === "freelancer"
                    ? "var(--omed-primary)"
                    : "var(--omed-border)",
                borderWidth: 2,
                backgroundColor:
                  template === "freelancer" ? "var(--omed-tint)" : undefined,
              }}
              onClick={() => setTemplate("freelancer")}
            >
              <div className="flex items-start justify-between gap-4">
                <div className="flex flex-col gap-2">
                  <div className="flex items-center gap-2">
                    <RocketOutlined
                      style={{ fontSize: 20, color: "var(--omed-primary)" }}
                    />
                    <Title level={5} style={{ margin: 0 }}>
                      {t("fin.setup.freelancerTitle")}
                    </Title>
                    <Tag color="orange">
                      {previewData?.accounts?.length ?? 50} Akun
                    </Tag>
                  </div>
                  <Paragraph type="secondary" style={{ marginBottom: 8 }}>
                    {t("fin.setup.freelancerDesc")}
                  </Paragraph>
                  <Space wrap size={[4, 8]}>
                    <Tag icon={<BankOutlined />} color="blue">
                      {t("fin.setup.tags.banks")}
                    </Tag>
                    <Tag icon={<CreditCardOutlined />} color="cyan">
                      {t("fin.setup.tags.ewallet")}
                    </Tag>
                    <Tag color="volcano">{t("fin.setup.tags.paylater")}</Tag>
                    <Tag color="magenta">{t("fin.setup.tags.personal")}</Tag>
                    <Tag icon={<DollarOutlined />} color="purple">
                      {t("fin.setup.tags.prive")}
                    </Tag>
                    <Tag color="red">{t("fin.setup.tags.tax")}</Tag>
                    <Tag color="green">{t("fin.setup.tags.saas")}</Tag>
                  </Space>
                </div>
                {template === "freelancer" && (
                  <CheckCircleFilled
                    style={{ fontSize: 24, color: "var(--omed-primary)" }}
                  />
                )}
              </div>
            </Card>
          </div>
        </Panel>

        {/* Language Selection */}
        <Panel strong title={t("fin.setup.language")}>
          <Radio.Group
            value={lang}
            onChange={(e) => setLang(e.target.value)}
            className="flex flex-col gap-3"
          >
            <Radio value="id">
              <span className="font-medium">{t("fin.setup.languages.id")}</span>
              <Text type="secondary" className="ml-2 text-xs">
                (mis. Kas Tunai, Bank BCA, Utang PayLater, Beban Hidup &
                Pribadi, Makanan & Minuman)
              </Text>
            </Radio>
            <Radio value="en">
              <span className="font-medium">{t("fin.setup.languages.en")}</span>
              <Text type="secondary" className="ml-2 text-xs">
                (e.g., Petty Cash, BCA Bank Account, PayLater Loans, Personal &
                Living Expenses, Food & Groceries)
              </Text>
            </Radio>
          </Radio.Group>
        </Panel>

        {/* Initial Period Configuration */}
        <Panel strong title={t("fin.setup.periodTitle")}>
          <Paragraph type="secondary" style={{ marginBottom: 16 }}>
            {t("fin.setup.periodDesc")}
          </Paragraph>
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12}>
              <div className="flex flex-col gap-1">
                <Text strong>{t("fin.setup.fiscalYear")}</Text>
                <InputNumber
                  aria-label={t("fin.setup.fiscalYear")}
                  className="w-full"
                  min={2000}
                  max={2100}
                  value={fiscalYear}
                  onChange={(val) => {
                    if (val) {
                      setFiscalYear(val);
                      setDateRange([
                        dayjs(`${val}-01-01`),
                        dayjs(`${val}-12-31`),
                      ]);
                      setPeriodName(`FY ${val}`);
                    }
                  }}
                />
              </div>
            </Col>
            <Col xs={24} sm={12}>
              <div className="flex flex-col gap-1">
                <Text strong>{t("fin.setup.periodName")}</Text>
                <Input
                  aria-label={t("fin.setup.periodName")}
                  value={periodName}
                  placeholder={t("fin.setup.periodNamePlaceholder")}
                  onChange={(e) => setPeriodName(e.target.value)}
                />
              </div>
            </Col>
            <Col xs={24} sm={12}>
              <div className="flex flex-col gap-1">
                <Text strong>{t("fin.setup.periodDates")}</Text>
                <DatePicker.RangePicker
                  aria-label={t("fin.setup.periodDates")}
                  className="w-full"
                  value={dateRange}
                  onChange={(dates) => {
                    if (dates?.[0] && dates[1]) {
                      setDateRange([dates[0], dates[1]]);
                    }
                  }}
                />
              </div>
            </Col>
            <Col xs={24} sm={12}>
              <div className="flex flex-col gap-1">
                <Text strong>{t("fin.setup.periodDescription")}</Text>
                <Input
                  aria-label={t("fin.setup.periodDescription")}
                  value={description}
                  placeholder={t("fin.setup.periodDescription")}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </div>
            </Col>
          </Row>
        </Panel>

        {/* Dynamic Preview of accounts from API */}
        <Panel strong title={t("fin.setup.preview")}>
          <div className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <Segmented
                value={selectedType}
                onChange={(val) => setSelectedType(String(val))}
                options={[
                  { label: t("fin.setup.filterAll"), value: "ALL" },
                  { label: t("fin.types.ASSET"), value: "ASSET" },
                  { label: t("fin.types.LIABILITY"), value: "LIABILITY" },
                  { label: t("fin.types.EQUITY"), value: "EQUITY" },
                  { label: t("fin.types.REVENUE"), value: "REVENUE" },
                  { label: t("fin.types.EXPENSE"), value: "EXPENSE" },
                ]}
              />
              <Input
                placeholder={t("fin.setup.previewSearchPlaceholder")}
                prefix={<SearchOutlined className="text-muted-foreground" />}
                className="w-full sm:w-64"
                allowClear
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>

            {previewLoading ? (
              <div className="flex justify-center py-8">
                <Spin />
              </div>
            ) : (
              <Table
                rowKey="code"
                columns={previewColumns}
                dataSource={filteredAccounts}
                pagination={{ pageSize: 8, showSizeChanger: false }}
                size="small"
              />
            )}
          </div>
        </Panel>

        {/* Actions */}
        <div className="flex items-center justify-end gap-3 pt-2">
          <Button onClick={() => navigate("/fin/accounts")}>
            {t("fin.setup.cancel")}
          </Button>
          <Button
            type="primary"
            icon={<CheckCircleFilled />}
            loading={pending}
            onClick={applyTemplate}
          >
            {t("fin.setup.apply")}
          </Button>
        </div>
      </div>
    </Page>
  );
}
