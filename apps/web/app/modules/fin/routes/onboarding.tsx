import {
  ArrowLeftOutlined,
  ArrowRightOutlined,
  CheckCircleFilled,
} from "@ant-design/icons";
import { useQueryClient } from "@tanstack/react-query";
import { App, Button, Divider, Steps, Typography } from "antd";
import dayjs from "dayjs";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { redirect, useNavigate } from "react-router";
import { Page } from "~/components/Page";
import { Panel } from "~/components/Panel";
import { api, unwrapOne } from "~/lib/api";
import { setActiveDecimalStyle } from "~/lib/format";
import { getQueryClient } from "~/lib/query-client";
import { requireWorkspaceId } from "~/lib/workspace";
import {
  CoaTemplateFields,
  CurrencyFormatFields,
  FiscalPeriodFields,
} from "../components";
import {
  accountKeys,
  finSettingsQueryOptions,
  periodKeys,
  settingKeys,
} from "../queries";

const { Title, Paragraph } = Typography;

export async function clientLoader() {
  const wid = await requireWorkspaceId();
  const settings = await getQueryClient().ensureQueryData(
    finSettingsQueryOptions(wid),
  );
  if (settings.onboarding) {
    return redirect("/fin");
  }
  return { wid };
}

export function shouldRevalidate() {
  return false;
}

export interface OnboardingProps {
  loaderData: { wid: number };
}

export default function Onboarding({ loaderData }: OnboardingProps) {
  const { wid } = loaderData;
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { message } = App.useApp();
  const [pending, setPending] = useState(false);
  const [currentStep, setCurrentStep] = useState(0);

  const isEn = i18n.language.startsWith("en");
  const [baseCurrency, setBaseCurrency] = useState(isEn ? "USD" : "IDR");
  const [decimalStyle, setDecimalStyle] = useState<"dot" | "comma">(
    isEn ? "dot" : "comma",
  );

  const currentYear = new Date().getFullYear();
  const [fiscalYear, setFiscalYear] = useState<number>(currentYear);
  const [periodName, setPeriodName] = useState<string>(`FY ${currentYear}`);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>([
    dayjs(`${currentYear}-01-01`),
    dayjs(`${currentYear}-12-31`),
  ]);
  const [description, setDescription] = useState<string>("");

  const [coaMode, setCoaMode] = useState<"template" | "blank">("template");
  const [template, setTemplate] = useState("freelancer");
  const [lang, setLang] = useState<"id" | "en">(isEn ? "en" : "id");

  const handleNext = () => {
    if (!periodName.trim()) {
      message.error(`${t("fin.onboarding.periodName")} required`);
      return;
    }
    if (!dateRange?.[0] || !dateRange[1]) {
      message.error(`${t("fin.onboarding.periodDates")} required`);
      return;
    }
    setCurrentStep(1);
  };

  const handleFinish = async () => {
    setPending(true);
    try {
      // 1. Base currency
      try {
        await unwrapOne(
          api.PUT("/workspaces/{wid}/fin/settings/base-currency", {
            params: { path: { wid } },
            body: { baseCurrency },
          }),
        );
      } catch (err: unknown) {
        const status = (err as { status?: number })?.status;
        if (status !== 409) {
          throw err;
        }
      }

      // 2. Seed CoA if template mode
      if (coaMode === "template") {
        try {
          await unwrapOne(
            api.POST("/workspaces/{wid}/fin/accounts/seed", {
              params: { path: { wid } },
              body: { template, lang },
            }),
          );
        } catch (err: unknown) {
          const status = (err as { status?: number })?.status;
          if (status !== 409) {
            throw err;
          }
        }
      }

      // 3. Create initial fiscal period
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
      } catch (err: unknown) {
        const status = (err as { status?: number; code?: string })?.status;
        const code = (err as { status?: number; code?: string })?.code;
        if (status !== 409 && code !== "PERIOD_OVERLAP") {
          throw err;
        }
      }

      // 4. Update settings (decimalStyle & onboarding)
      const updatedSettings = await unwrapOne(
        api.PUT("/workspaces/{wid}/fin/settings", {
          params: { path: { wid } },
          body: { decimalStyle, onboarding: true },
        }),
      );

      // 5. Update local state & invalidations
      setActiveDecimalStyle(decimalStyle);
      queryClient.setQueryData(settingKeys.all(wid), updatedSettings);
      await queryClient.invalidateQueries({ queryKey: settingKeys.all(wid) });
      await queryClient.invalidateQueries({ queryKey: accountKeys.all(wid) });
      await queryClient.invalidateQueries({ queryKey: periodKeys.all(wid) });
      message.success(t("fin.onboarding.success"));
      navigate("/fin");
    } catch (error) {
      message.error(error instanceof Error ? error.message : t("common.error"));
    } finally {
      setPending(false);
    }
  };

  return (
    <Page title={t("fin.onboarding.title")}>
      <div className="mx-auto flex w-full max-w-4xl flex-col gap-6 py-2">
        <div>
          <Title level={4} style={{ marginBottom: 4 }}>
            {t("fin.onboarding.title")}
          </Title>
          <Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {t("fin.onboarding.subtitle")}
          </Paragraph>
        </div>

        <Steps
          current={currentStep}
          items={[
            {
              title: t("fin.onboarding.step1Title"),
              content: t("fin.onboarding.step1Desc"),
            },
            {
              title: t("fin.onboarding.step2Title"),
              content: t("fin.onboarding.step2Desc"),
            },
          ]}
        />

        {currentStep === 0 && (
          <div className="flex flex-col gap-6">
            <Panel strong title={t("fin.onboarding.step1Title")}>
              <div className="flex flex-col gap-6">
                <CurrencyFormatFields
                  wid={wid}
                  baseCurrency={baseCurrency}
                  onBaseCurrencyChange={setBaseCurrency}
                  decimalStyle={decimalStyle}
                  onDecimalStyleChange={setDecimalStyle}
                />
                <Divider style={{ margin: "12px 0" }} />
                <div>
                  <Title level={5} style={{ marginBottom: 4 }}>
                    {t("fin.onboarding.fiscalPeriodTitle")}
                  </Title>
                  <Paragraph
                    type="secondary"
                    style={{ marginBottom: 16, fontSize: "12px" }}
                  >
                    {t("fin.onboarding.fiscalPeriodDesc")}
                  </Paragraph>
                  <FiscalPeriodFields
                    fiscalYear={fiscalYear}
                    onFiscalYearChange={(year) => {
                      setFiscalYear(year);
                      setDateRange([
                        dayjs(`${year}-01-01`),
                        dayjs(`${year}-12-31`),
                      ]);
                      setPeriodName(`FY ${year}`);
                    }}
                    periodName={periodName}
                    onPeriodNameChange={setPeriodName}
                    dateRange={dateRange}
                    onDateRangeChange={setDateRange}
                    description={description}
                    onDescriptionChange={setDescription}
                  />
                </div>
              </div>
            </Panel>

            <div className="flex items-center justify-end gap-3">
              <Button
                type="primary"
                icon={<ArrowRightOutlined />}
                onClick={handleNext}
              >
                {t("fin.onboarding.next")}
              </Button>
            </div>
          </div>
        )}

        {currentStep === 1 && (
          <div className="flex flex-col gap-6">
            <Panel strong title={t("fin.onboarding.step2Title")}>
              <CoaTemplateFields
                wid={wid}
                coaMode={coaMode}
                onCoaModeChange={setCoaMode}
                template={template}
                onTemplateChange={setTemplate}
                lang={lang}
                onLangChange={setLang}
              />
            </Panel>

            <div className="flex items-center justify-between gap-3">
              <Button
                icon={<ArrowLeftOutlined />}
                onClick={() => setCurrentStep(0)}
              >
                {t("fin.onboarding.back")}
              </Button>
              <Button
                type="primary"
                icon={<CheckCircleFilled />}
                loading={pending}
                onClick={handleFinish}
              >
                {t("fin.onboarding.finish")}
              </Button>
            </div>
          </div>
        )}
      </div>
    </Page>
  );
}
