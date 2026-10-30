import { Col, DatePicker, Input, InputNumber, Row, Typography } from "antd";
import type dayjs from "dayjs";
import { useTranslation } from "react-i18next";

const { Text } = Typography;

export interface FiscalPeriodFieldsProps {
  fiscalYear: number;
  onFiscalYearChange: (year: number) => void;
  periodName: string;
  onPeriodNameChange: (name: string) => void;
  dateRange: [dayjs.Dayjs, dayjs.Dayjs];
  onDateRangeChange: (dates: [dayjs.Dayjs, dayjs.Dayjs]) => void;
  description: string;
  onDescriptionChange: (desc: string) => void;
}

export function FiscalPeriodFields({
  fiscalYear,
  onFiscalYearChange,
  periodName,
  onPeriodNameChange,
  dateRange,
  onDateRangeChange,
  description,
  onDescriptionChange,
}: FiscalPeriodFieldsProps) {
  const { t } = useTranslation();

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} sm={12}>
        <div className="flex flex-col gap-1">
          <Text strong>{t("fin.onboarding.fiscalYear")}</Text>
          <InputNumber
            aria-label={t("fin.onboarding.fiscalYear")}
            className="w-full"
            min={2000}
            max={2100}
            value={fiscalYear}
            onChange={(val) => {
              if (val) {
                onFiscalYearChange(val);
              }
            }}
          />
        </div>
      </Col>
      <Col xs={24} sm={12}>
        <div className="flex flex-col gap-1">
          <Text strong>{t("fin.onboarding.periodName")}</Text>
          <Input
            aria-label={t("fin.onboarding.periodName")}
            value={periodName}
            placeholder={t("fin.onboarding.periodNamePlaceholder")}
            onChange={(e) => onPeriodNameChange(e.target.value)}
          />
        </div>
      </Col>
      <Col xs={24} sm={12}>
        <div className="flex flex-col gap-1">
          <Text strong>{t("fin.onboarding.periodDates")}</Text>
          <DatePicker.RangePicker
            aria-label={t("fin.onboarding.periodDates")}
            className="w-full"
            value={dateRange}
            onChange={(dates) => {
              if (dates?.[0] && dates[1]) {
                onDateRangeChange([dates[0], dates[1]]);
              }
            }}
          />
        </div>
      </Col>
      <Col xs={24} sm={12}>
        <div className="flex flex-col gap-1">
          <Text strong>{t("fin.onboarding.periodDescription")}</Text>
          <Input
            aria-label={t("fin.onboarding.periodDescription")}
            value={description}
            placeholder={t("fin.onboarding.periodDescription")}
            onChange={(e) => onDescriptionChange(e.target.value)}
          />
        </div>
      </Col>
    </Row>
  );
}
