import { useQuery } from "@tanstack/react-query";
import { Card, Col, Radio, Row, Tooltip, Typography } from "antd";
import { useTranslation } from "react-i18next";
import { formatMoney } from "~/lib/format";
import { currenciesQueryOptions } from "../queries";
import { CurrencySelect } from "./currency-select";

const { Text, Paragraph } = Typography;

export interface CurrencyFormatFieldsProps {
  wid: number;
  baseCurrency: string;
  onBaseCurrencyChange: (currency: string) => void;
  baseCurrencyDisabled?: boolean;
  decimalStyle: "dot" | "comma";
  onDecimalStyleChange: (style: "dot" | "comma") => void;
}

export function CurrencyFormatFields({
  wid,
  baseCurrency,
  onBaseCurrencyChange,
  baseCurrencyDisabled,
  decimalStyle,
  onDecimalStyleChange,
}: CurrencyFormatFieldsProps) {
  const { t } = useTranslation();
  const { data: currencies = [], isLoading } = useQuery(
    currenciesQueryOptions(wid),
  );

  const sampleAmount = 1234567.89;
  const dotPreview = formatMoney(sampleAmount, baseCurrency, "en", "dot");
  const commaPreview = formatMoney(sampleAmount, baseCurrency, "id", "comma");

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <div>
          <Text strong>{t("fin.onboarding.baseCurrency")}</Text>
          <Paragraph
            type="secondary"
            style={{ marginBottom: 8, fontSize: "12px" }}
          >
            {t("fin.onboarding.baseCurrencyDesc")}
          </Paragraph>
        </div>
        <div className="max-w-md">
          <Tooltip
            title={
              baseCurrencyDisabled
                ? t("fin.settings.currencyLockedTooltip")
                : undefined
            }
          >
            <div>
              <CurrencySelect
                currencies={currencies}
                value={baseCurrency}
                onChange={onBaseCurrencyChange}
                disabled={baseCurrencyDisabled}
                loading={isLoading}
                className="w-full"
              />
            </div>
          </Tooltip>
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <div>
          <Text strong>{t("fin.onboarding.decimalStyle")}</Text>
          <Paragraph
            type="secondary"
            style={{ marginBottom: 8, fontSize: "12px" }}
          >
            {t("fin.onboarding.decimalStyleDesc")}
          </Paragraph>
        </div>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12}>
            <Card
              hoverable
              className="cursor-pointer transition-all"
              style={{
                borderColor:
                  decimalStyle === "dot"
                    ? "var(--omed-primary)"
                    : "var(--omed-border)",
                borderWidth: 2,
                backgroundColor:
                  decimalStyle === "dot"
                    ? "var(--omed-tint)"
                    : "var(--omed-surface)",
              }}
              onClick={() => onDecimalStyleChange("dot")}
            >
              <div className="flex flex-col gap-1">
                <div className="flex items-center justify-between">
                  <Text strong style={{ color: "var(--omed-text)" }}>
                    {t("fin.onboarding.dotTitle")}
                  </Text>
                  <Radio
                    checked={decimalStyle === "dot"}
                    onChange={() => onDecimalStyleChange("dot")}
                  />
                </div>
                <div
                  className="font-mono text-base font-semibold"
                  style={{ color: "var(--omed-primary-text)" }}
                >
                  {dotPreview} (1,234,567.89)
                </div>
                <Text type="secondary" className="text-xs">
                  {t("fin.onboarding.dotDesc")}
                </Text>
              </div>
            </Card>
          </Col>
          <Col xs={24} sm={12}>
            <Card
              hoverable
              className="cursor-pointer transition-all"
              style={{
                borderColor:
                  decimalStyle === "comma"
                    ? "var(--omed-primary)"
                    : "var(--omed-border)",
                borderWidth: 2,
                backgroundColor:
                  decimalStyle === "comma"
                    ? "var(--omed-tint)"
                    : "var(--omed-surface)",
              }}
              onClick={() => onDecimalStyleChange("comma")}
            >
              <div className="flex flex-col gap-1">
                <div className="flex items-center justify-between">
                  <Text strong style={{ color: "var(--omed-text)" }}>
                    {t("fin.onboarding.commaTitle")}
                  </Text>
                  <Radio
                    checked={decimalStyle === "comma"}
                    onChange={() => onDecimalStyleChange("comma")}
                  />
                </div>
                <div
                  className="font-mono text-base font-semibold"
                  style={{ color: "var(--omed-primary-text)" }}
                >
                  {commaPreview} (1.234.567,89)
                </div>
                <Text type="secondary" className="text-xs">
                  {t("fin.onboarding.commaDesc")}
                </Text>
              </div>
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  );
}
