import { PlusOutlined, SaveOutlined } from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import {
  App,
  Button,
  Col,
  Divider,
  Modal,
  Row,
  Space,
  Table,
  type TableProps,
  Tag,
  Typography,
} from "antd";
import dayjs, { type Dayjs } from "dayjs";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapOne } from "~/lib/api";
import { type DecimalStyle, setActiveDecimalStyle } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  AccountSelect,
  CoaTemplateFields,
  CurrencyFormatFields,
  FiscalPeriodFields,
} from "../components";
import {
  accountKeys,
  accountsQueryOptions,
  finSettingsQueryOptions,
  periodKeys,
  periodsQueryOptions,
  recentEntriesQueryOptions,
  settingKeys,
} from "../queries";

const { Text, Title, Paragraph } = Typography;

type Period = components["schemas"]["model.PeriodResponse"];

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const queryClient = getQueryClient();
  await Promise.all([
    queryClient.ensureQueryData(finSettingsQueryOptions(wid)),
    queryClient.ensureQueryData(periodsQueryOptions(wid)),
    queryClient.ensureQueryData(accountsQueryOptions(wid)),
    queryClient.ensureQueryData(recentEntriesQueryOptions(wid, 1, "POSTED")),
  ]);
  return { wid };
}

export interface SettingsProps {
  loaderData: { wid: number };
}

export default function SettingsRoute({ loaderData }: SettingsProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const { message } = App.useApp();
  const queryClient = useQueryClient();

  const { data: settings } = useSuspenseQuery(finSettingsQueryOptions(wid));
  const { data: periods = [] } = useSuspenseQuery(periodsQueryOptions(wid));
  const { data: accounts = [] } = useSuspenseQuery(accountsQueryOptions(wid));
  const { data: postedEntries = [] } = useSuspenseQuery(
    recentEntriesQueryOptions(wid, 1, "POSTED"),
  );

  const hasPostedEntries = postedEntries.length > 0;

  // General & Format State
  const [baseCurrency, setBaseCurrency] = useState(
    settings?.baseCurrency || "USD",
  );
  const [decimalStyle, setDecimalStyle] = useState<DecimalStyle>(
    (settings?.decimalStyle as DecimalStyle) || "comma",
  );
  const [savingGeneral, setSavingGeneral] = useState(false);

  // Period Modal State
  const [periodModalOpen, setPeriodModalOpen] = useState(false);
  const currentYear = new Date().getFullYear();
  const [fiscalYear, setFiscalYear] = useState(currentYear);
  const [periodName, setPeriodName] = useState(`FY ${currentYear}`);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([
    dayjs(`${currentYear}-01-01`),
    dayjs(`${currentYear}-12-31`),
  ]);
  const [periodDescription, setPeriodDescription] = useState("");
  const [savingPeriod, setSavingPeriod] = useState(false);

  // CoA Seeding State (if accounts are empty)
  const [coaMode, setCoaMode] = useState<"template" | "blank">("template");
  const [template, setTemplate] = useState("freelancer");
  const [templateLang, setTemplateLang] = useState<"id" | "en">(
    i18n.language.startsWith("en") ? "en" : "id",
  );
  const [seedingCoa, setSeedingCoa] = useState(false);

  // FX Accounts State
  const [fxGainAccountId, setFxGainAccountId] = useState<number | undefined>(
    settings?.fxGainAccountId,
  );
  const [fxLossAccountId, setFxLossAccountId] = useState<number | undefined>(
    settings?.fxLossAccountId,
  );
  const [fxUnrealizedGainAccountId, setFxUnrealizedGainAccountId] = useState<
    number | undefined
  >(settings?.fxUnrealizedGainAccountId);
  const [fxUnrealizedLossAccountId, setFxUnrealizedLossAccountId] = useState<
    number | undefined
  >(settings?.fxUnrealizedLossAccountId);
  const [fxRoundingAccountId, setFxRoundingAccountId] = useState<
    number | undefined
  >(settings?.fxRoundingAccountId);
  const [savingFx, setSavingFx] = useState(false);

  const handleSaveGeneral = async () => {
    try {
      setSavingGeneral(true);
      if (!hasPostedEntries && baseCurrency !== settings?.baseCurrency) {
        await unwrapOne(
          api.PUT("/workspaces/{wid}/fin/settings/base-currency", {
            params: { path: { wid } },
            body: { baseCurrency },
          }),
        );
      }
      await unwrapOne(
        api.PUT("/workspaces/{wid}/fin/settings", {
          params: { path: { wid } },
          body: { decimalStyle, onboarding: true },
        }),
      );
      setActiveDecimalStyle(decimalStyle);
      await queryClient.invalidateQueries({ queryKey: settingKeys.all(wid) });
      message.success(t("fin.settings.saveSuccess"));
    } catch (err) {
      message.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSavingGeneral(false);
    }
  };

  const handleCreatePeriod = async () => {
    try {
      setSavingPeriod(true);
      await unwrapOne(
        api.POST("/workspaces/{wid}/fin/periods", {
          params: { path: { wid } },
          body: {
            name: periodName.trim(),
            fiscalYear,
            startDate: dateRange[0].format("YYYY-MM-DD"),
            endDate: dateRange[1].format("YYYY-MM-DD"),
            description: periodDescription.trim() || undefined,
          },
        }),
      );
      await queryClient.invalidateQueries({ queryKey: periodKeys.all(wid) });
      message.success(t("fin.settings.createPeriodSuccess"));
      setPeriodModalOpen(false);
      setPeriodName(`FY ${fiscalYear}`);
      setPeriodDescription("");
    } catch (err) {
      message.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSavingPeriod(false);
    }
  };

  const handleSeedTemplate = async () => {
    try {
      setSeedingCoa(true);
      await unwrapOne(
        api.POST("/workspaces/{wid}/fin/accounts/seed", {
          params: { path: { wid } },
          body: { template, lang: templateLang },
        }),
      );
      await queryClient.invalidateQueries({ queryKey: accountKeys.all(wid) });
      await queryClient.invalidateQueries({ queryKey: settingKeys.all(wid) });
      message.success(t("fin.settings.seedTemplateSuccess"));
    } catch (err) {
      message.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSeedingCoa(false);
    }
  };

  const handleSaveFx = async () => {
    try {
      setSavingFx(true);
      await unwrapOne(
        api.PUT("/workspaces/{wid}/fin/settings/fx-accounts", {
          params: { path: { wid } },
          body: {
            fxGainAccountId: fxGainAccountId || undefined,
            fxLossAccountId: fxLossAccountId || undefined,
            fxUnrealizedGainAccountId: fxUnrealizedGainAccountId || undefined,
            fxUnrealizedLossAccountId: fxUnrealizedLossAccountId || undefined,
            fxRoundingAccountId: fxRoundingAccountId || undefined,
          },
        }),
      );
      await queryClient.invalidateQueries({ queryKey: settingKeys.all(wid) });
      message.success(t("fin.settings.saveSuccess"));
    } catch (err) {
      message.error(err instanceof Error ? err.message : t("common.error"));
    } finally {
      setSavingFx(false);
    }
  };

  const periodColumns: TableProps<Period>["columns"] = [
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
      width: 120,
    },
    {
      title: t("fin.periods.dateRange"),
      key: "dateRange",
      width: 220,
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
      width: 110,
      render: (status: string) => (
        <Tag color={status === "OPEN" ? "green" : "default"}>
          {status === "OPEN"
            ? t("fin.periods.statusOpen")
            : t("fin.periods.statusClosed")}
        </Tag>
      ),
    },
  ];

  return (
    <Page title={t("fin.settings.title")}>
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 py-2">
        <div>
          <Title level={4} style={{ marginBottom: 4 }}>
            {t("fin.settings.title")}
          </Title>
          <Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {t("fin.settings.subtitle")}
          </Paragraph>
        </div>

        {/* Panel 1: Currency & Number Format */}
        <Panel strong title={t("fin.settings.currencyFormatTitle")}>
          <div className="flex flex-col gap-6">
            <Paragraph
              type="secondary"
              style={{ marginBottom: 0, fontSize: "13px" }}
            >
              {t("fin.settings.currencyFormatDesc")}
            </Paragraph>

            <CurrencyFormatFields
              wid={wid}
              baseCurrency={baseCurrency}
              onBaseCurrencyChange={setBaseCurrency}
              baseCurrencyDisabled={hasPostedEntries}
              decimalStyle={decimalStyle}
              onDecimalStyleChange={setDecimalStyle}
            />

            <div className="flex justify-end">
              <Button
                type="primary"
                icon={<SaveOutlined />}
                loading={savingGeneral}
                onClick={handleSaveGeneral}
              >
                {t("fin.settings.saveChanges")}
              </Button>
            </div>
          </div>
        </Panel>

        {/* Panel 2: Fiscal Periods */}
        <Panel strong title={t("fin.settings.fiscalPeriodsTitle")}>
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <Paragraph
                type="secondary"
                style={{ marginBottom: 0, fontSize: "13px" }}
              >
                {t("fin.settings.fiscalPeriodsDesc")}
              </Paragraph>
              <Space>
                <Link to="/fin/periods">
                  <Button type="link">
                    {t("fin.settings.allPeriodsLink")}
                  </Button>
                </Link>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setPeriodModalOpen(true)}
                >
                  {t("fin.settings.newPeriod")}
                </Button>
              </Space>
            </div>

            <Table<Period>
              rowKey="id"
              dataSource={periods}
              columns={periodColumns}
              pagination={false}
              size="small"
            />
          </div>
        </Panel>

        {/* Panel 3: Chart of Accounts & FX Mappings */}
        <Panel strong title={t("fin.settings.coaFxTitle")}>
          <div className="flex flex-col gap-6">
            <Paragraph
              type="secondary"
              style={{ marginBottom: 0, fontSize: "13px" }}
            >
              {t("fin.settings.coaFxDesc")}
            </Paragraph>

            {accounts.length === 0 && (
              <div className="flex flex-col gap-4">
                <CoaTemplateFields
                  wid={wid}
                  coaMode={coaMode}
                  onCoaModeChange={setCoaMode}
                  template={template}
                  onTemplateChange={setTemplate}
                  lang={templateLang}
                  onLangChange={setTemplateLang}
                />
                {coaMode === "template" && (
                  <div className="flex justify-end">
                    <Button
                      type="primary"
                      loading={seedingCoa}
                      onClick={handleSeedTemplate}
                    >
                      {t("fin.settings.seedTemplateButton")}
                    </Button>
                  </div>
                )}
                <Divider style={{ margin: "12px 0" }} />
              </div>
            )}

            {/* FX Accounts Mapping */}
            <div className="flex flex-col gap-4">
              <Title level={5} style={{ margin: 0 }}>
                {t("fin.settings.fxAccountsTitle")}
              </Title>

              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12}>
                  <div className="flex flex-col gap-1">
                    <Text strong>{t("fin.settings.realizedGainAccount")}</Text>
                    <AccountSelect
                      accounts={accounts}
                      value={fxGainAccountId}
                      onChange={setFxGainAccountId}
                      placeholder={t("fin.settings.realizedGainAccount")}
                    />
                  </div>
                </Col>

                <Col xs={24} sm={12}>
                  <div className="flex flex-col gap-1">
                    <Text strong>{t("fin.settings.realizedLossAccount")}</Text>
                    <AccountSelect
                      accounts={accounts}
                      value={fxLossAccountId}
                      onChange={setFxLossAccountId}
                      placeholder={t("fin.settings.realizedLossAccount")}
                    />
                  </div>
                </Col>

                <Col xs={24} sm={12}>
                  <div className="flex flex-col gap-1">
                    <Text strong>
                      {t("fin.settings.unrealizedGainAccount")}
                    </Text>
                    <AccountSelect
                      accounts={accounts}
                      value={fxUnrealizedGainAccountId}
                      onChange={setFxUnrealizedGainAccountId}
                      placeholder={t("fin.settings.unrealizedGainAccount")}
                    />
                  </div>
                </Col>

                <Col xs={24} sm={12}>
                  <div className="flex flex-col gap-1">
                    <Text strong>
                      {t("fin.settings.unrealizedLossAccount")}
                    </Text>
                    <AccountSelect
                      accounts={accounts}
                      value={fxUnrealizedLossAccountId}
                      onChange={setFxUnrealizedLossAccountId}
                      placeholder={t("fin.settings.unrealizedLossAccount")}
                    />
                  </div>
                </Col>

                <Col xs={24} sm={12}>
                  <div className="flex flex-col gap-1">
                    <Text strong>{t("fin.settings.roundingAccount")}</Text>
                    <AccountSelect
                      accounts={accounts}
                      value={fxRoundingAccountId}
                      onChange={setFxRoundingAccountId}
                      placeholder={t("fin.settings.roundingAccount")}
                    />
                  </div>
                </Col>
              </Row>

              <div className="flex justify-end pt-2">
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  loading={savingFx}
                  onClick={handleSaveFx}
                >
                  {t("fin.settings.saveChanges")}
                </Button>
              </div>
            </div>
          </div>
        </Panel>

        {/* Modal: New Fiscal Period */}
        <Modal
          title={t("fin.settings.newPeriod")}
          open={periodModalOpen}
          onOk={handleCreatePeriod}
          confirmLoading={savingPeriod}
          onCancel={() => setPeriodModalOpen(false)}
          destroyOnHidden
        >
          <div className="py-4">
            <FiscalPeriodFields
              fiscalYear={fiscalYear}
              onFiscalYearChange={(year) => {
                setFiscalYear(year);
                setDateRange([dayjs(`${year}-01-01`), dayjs(`${year}-12-31`)]);
                setPeriodName(`FY ${year}`);
              }}
              periodName={periodName}
              onPeriodNameChange={setPeriodName}
              dateRange={dateRange}
              onDateRangeChange={setDateRange}
              description={periodDescription}
              onDescriptionChange={setPeriodDescription}
            />
          </div>
        </Modal>
      </div>
    </Page>
  );
}
