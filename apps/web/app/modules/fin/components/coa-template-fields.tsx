import {
  BankOutlined,
  CreditCardOutlined,
  DollarOutlined,
  EditOutlined,
  RocketOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import type { components } from "@omed/openapi";
import { useQuery } from "@tanstack/react-query";
import {
  Card,
  Col,
  Input,
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
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { templatePreviewQueryOptions } from "../queries";

const { Text, Title, Paragraph } = Typography;

type TemplateAccount =
  components["schemas"]["model.AccountTemplateItemResponse"];

export interface CoaTemplateFieldsProps {
  wid: number;
  coaMode: "template" | "blank";
  onCoaModeChange: (mode: "template" | "blank") => void;
  template: string;
  onTemplateChange: (template: string) => void;
  lang: "id" | "en";
  onLangChange: (lang: "id" | "en") => void;
}

export function CoaTemplateFields({
  wid,
  coaMode,
  onCoaModeChange,
  template,
  onTemplateChange: _onTemplateChange,
  lang,
  onLangChange,
}: CoaTemplateFieldsProps) {
  const { t } = useTranslation();
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedType, setSelectedType] = useState<string>("ALL");

  const { data: previewData, isLoading: previewLoading } = useQuery(
    templatePreviewQueryOptions(wid, template, lang),
  );

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
    <div className="flex flex-col gap-6">
      {/* Mode Selection */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12}>
          <Card
            hoverable
            className="cursor-pointer transition-all"
            style={{
              borderColor:
                coaMode === "template"
                  ? "var(--omed-primary)"
                  : "var(--omed-border)",
              borderWidth: 2,
              backgroundColor:
                coaMode === "template"
                  ? "var(--omed-tint)"
                  : "var(--omed-surface)",
            }}
            onClick={() => onCoaModeChange("template")}
          >
            <div className="flex items-start justify-between gap-2">
              <div className="flex flex-col gap-1">
                <div className="flex items-center gap-2">
                  <RocketOutlined
                    style={{ fontSize: 18, color: "var(--omed-primary)" }}
                  />
                  <Title level={5} style={{ margin: 0 }}>
                    {t("fin.onboarding.modeTemplate")}
                  </Title>
                </div>
                <Paragraph
                  type="secondary"
                  style={{ marginBottom: 0, fontSize: "12px" }}
                >
                  {t("fin.onboarding.modeTemplateDesc")}
                </Paragraph>
              </div>
              <Radio
                checked={coaMode === "template"}
                onChange={() => onCoaModeChange("template")}
              />
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card
            hoverable
            className="cursor-pointer transition-all"
            style={{
              borderColor:
                coaMode === "blank"
                  ? "var(--omed-primary)"
                  : "var(--omed-border)",
              borderWidth: 2,
              backgroundColor:
                coaMode === "blank"
                  ? "var(--omed-tint)"
                  : "var(--omed-surface)",
            }}
            onClick={() => onCoaModeChange("blank")}
          >
            <div className="flex items-start justify-between gap-2">
              <div className="flex flex-col gap-1">
                <div className="flex items-center gap-2">
                  <EditOutlined
                    style={{ fontSize: 18, color: "var(--omed-primary)" }}
                  />
                  <Title level={5} style={{ margin: 0 }}>
                    {t("fin.onboarding.modeBlank")}
                  </Title>
                </div>
                <Paragraph
                  type="secondary"
                  style={{ marginBottom: 0, fontSize: "12px" }}
                >
                  {t("fin.onboarding.modeBlankDesc")}
                </Paragraph>
              </div>
              <Radio
                checked={coaMode === "blank"}
                onChange={() => onCoaModeChange("blank")}
              />
            </div>
          </Card>
        </Col>
      </Row>

      {/* When Template Mode is Active */}
      {coaMode === "template" && (
        <div className="flex flex-col gap-6">
          {/* Template Card Details */}
          <Card
            style={{
              borderColor: "var(--omed-border)",
              backgroundColor: "var(--omed-surface-muted)",
            }}
          >
            <div className="flex items-start justify-between gap-4">
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2">
                  <Title level={5} style={{ margin: 0 }}>
                    {t("fin.onboarding.templateFreelancer")}
                  </Title>
                  <Tag color="orange">
                    {t("fin.onboarding.accountsCount", {
                      count: previewData?.accounts?.length ?? 50,
                    })}
                  </Tag>
                </div>
                <Paragraph type="secondary" style={{ marginBottom: 8 }}>
                  {t("fin.onboarding.templateFreelancerDesc")}
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
            </div>
          </Card>

          {/* Language Selection */}
          <div className="flex flex-col gap-2">
            <Text strong>{t("fin.onboarding.language")}</Text>
            <Radio.Group
              value={lang}
              onChange={(e) => onLangChange(e.target.value)}
              className="flex flex-col gap-3"
            >
              <Radio value="id">
                <span className="font-medium">
                  {t("fin.onboarding.languages.id")}
                </span>
                <Text type="secondary" className="ml-2 text-xs">
                  (mis. Kas Tunai, Bank BCA, Utang PayLater, Beban Hidup &
                  Pribadi, Makanan & Minuman)
                </Text>
              </Radio>
              <Radio value="en">
                <span className="font-medium">
                  {t("fin.onboarding.languages.en")}
                </span>
                <Text type="secondary" className="ml-2 text-xs">
                  (e.g., Petty Cash, BCA Bank Account, PayLater Loans, Personal
                  & Living Expenses, Food & Groceries)
                </Text>
              </Radio>
            </Radio.Group>
          </div>

          {/* Account Preview Table */}
          <div className="flex flex-col gap-3">
            <Text strong>{t("fin.onboarding.preview")}</Text>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <Segmented
                value={selectedType}
                onChange={(val) => setSelectedType(String(val))}
                options={[
                  { label: t("fin.onboarding.filterAll"), value: "ALL" },
                  { label: t("fin.types.ASSET"), value: "ASSET" },
                  { label: t("fin.types.LIABILITY"), value: "LIABILITY" },
                  { label: t("fin.types.EQUITY"), value: "EQUITY" },
                  { label: t("fin.types.REVENUE"), value: "REVENUE" },
                  { label: t("fin.types.EXPENSE"), value: "EXPENSE" },
                ]}
              />
              <Input
                placeholder={t("fin.onboarding.previewSearchPlaceholder")}
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
        </div>
      )}
    </div>
  );
}
