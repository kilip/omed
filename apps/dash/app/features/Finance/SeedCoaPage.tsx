import {
  DEFAULT_CURRENCY,
  SUPPORTED_CURRENCIES,
} from "@omed/better-auth/schema";
import type { SeedPreviewAccount } from "@omed/openapi/finance";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Col,
  Input,
  message,
  Radio,
  Result,
  Row,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { getCachedAuth } from "~/middleware/auth";
import { useApi } from "~/shared/hooks/useApi";

const TYPE_COLORS: Record<string, string> = {
  asset: "blue",
  liability: "orange",
  equity: "purple",
  revenue: "green",
  expense: "magenta",
};

const TYPE_NAMES: Record<string, string> = {
  asset: "finance:seed.types.asset",
  liability: "finance:seed.types.liability",
  equity: "finance:seed.types.equity",
  revenue: "finance:seed.types.revenue",
  expense: "finance:seed.types.expense",
};

export function meta() {
  return [{ title: "Seed Chart of Accounts" }];
}

export default function SeedCoaPage() {
  const { t, i18n } = useTranslation(["finance", "common"]);
  const navigate = useNavigate();
  const { finance } = useApi();
  const queryClient = useQueryClient();

  const cachedUser = getCachedAuth()?.user;
  const initialCurrency =
    (cachedUser?.defaultCurrency as string) ?? DEFAULT_CURRENCY;
  const initialLang = (
    i18n.resolvedLanguage ??
    i18n.language ??
    "en"
  ).startsWith("id")
    ? "id"
    : "en";

  const [profile, setProfile] = useState<string>("freelancer");
  const [currency, setCurrency] = useState<string>(initialCurrency);
  const [lang, setLang] = useState<string>(initialLang);
  const [search, setSearch] = useState<string>("");
  const [confirmOverwrite, setConfirmOverwrite] = useState<boolean>(false);
  const [seededCount, setSeededCount] = useState<number | null>(null);

  // 1. Fetch available templates
  const { data: templates = [], isLoading: isLoadingTemplates } = useQuery({
    queryKey: ["seed-templates"],
    queryFn: async () => {
      const res = await finance.GET("/accounts/seed/templates");
      if (res.error) {
        throw new Error("Failed to load templates");
      }
      return res.data?.data ?? [];
    },
  });

  // 2. Fetch preview for selected profile & language
  const {
    data: preview,
    isLoading: isLoadingPreview,
    error: previewError,
  } = useQuery({
    queryKey: ["seed-preview", profile, lang],
    queryFn: async () => {
      const res = await finance.GET("/accounts/seed/preview", {
        params: {
          query: { profile, lang },
        },
      });
      if (res.error) {
        throw new Error("Failed to load preview");
      }
      return res.data?.data;
    },
    enabled: Boolean(profile && lang),
  });

  // 3. Mutation to seed accounts
  const seedMutation = useMutation({
    mutationFn: async () => {
      const force = Boolean(preview && (preview.accountCount ?? 0) > 0);
      const res = await finance.POST("/accounts/seed", {
        body: {
          profile,
          lang,
          currency,
          force,
        },
      });
      if (res.error) {
        const errorMsg =
          res.error.error?.message || t("finance:seed.warning.accountsExist");
        throw new Error(errorMsg);
      }
      return res.data?.data ?? [];
    },
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: ["accounts"] });
      void queryClient.invalidateQueries({ queryKey: ["seed-preview"] });
      setSeededCount(data.length);
    },
    onError: (err: Error) => {
      message.error(err.message);
    },
  });

  const accounts = preview?.accounts ?? [];
  const filteredAccounts = useMemo(() => {
    if (!search.trim()) return accounts;
    const term = search.toLowerCase();
    return accounts.filter(
      (a) =>
        (a.code ?? "").toLowerCase().includes(term) ||
        (a.name ?? "").toLowerCase().includes(term),
    );
  }, [accounts, search]);

  const columns: ColumnsType<SeedPreviewAccount> = [
    {
      title: t("finance:seed.table.code"),
      dataIndex: "code",
      key: "code",
      width: 140,
      render: (code: string) => (
        <Typography.Text strong code>
          {code}
        </Typography.Text>
      ),
      sorter: (a, b) => (a.code ?? "").localeCompare(b.code ?? ""),
    },
    {
      title: t("finance:seed.table.name"),
      dataIndex: "name",
      key: "name",
      sorter: (a, b) => (a.name ?? "").localeCompare(b.name ?? ""),
    },
    {
      title: t("finance:seed.table.type"),
      dataIndex: "type",
      key: "type",
      width: 160,
      render: (type: string) => {
        const typeKey = type as keyof typeof TYPE_COLORS;
        const color = TYPE_COLORS[typeKey] ?? "default";
        const translationKey = (TYPE_NAMES[type] ??
          "finance:seed.types.asset") as "finance:seed.types.asset";
        const label = t(translationKey, type);
        return <Tag color={color}>{label}</Tag>;
      },
    },
  ];

  if (seededCount !== null) {
    return (
      <Card style={{ margin: "24px 0" }}>
        <Result
          status="success"
          title={t("finance:seed.success.title")}
          subTitle={t("finance:seed.success.description", {
            count: seededCount,
          })}
          extra={[
            <Button
              type="primary"
              key="finance"
              size="large"
              onClick={() => navigate("/fin")}
            >
              {t("finance:seed.success.goToFinance")}
            </Button>,
          ]}
        />
      </Card>
    );
  }

  const hasEntries = Boolean(preview && (preview.entryCount ?? 0) > 0);
  const hasAccounts = Boolean(preview && (preview.accountCount ?? 0) > 0);
  const isSubmitDisabled =
    isLoadingPreview ||
    hasEntries ||
    (hasAccounts && !confirmOverwrite) ||
    seedMutation.isPending;

  return (
    <div>
      <Typography.Title level={3} style={{ marginTop: 0, marginBottom: 8 }}>
        {t("finance:seed.title")}
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 24 }}>
        {t("finance:seed.description")}
      </Typography.Paragraph>

      <Space
        orientation="vertical"
        orientation-gap={20}
        size={20}
        style={{ width: "100%" }}
      >
        {hasEntries && (
          <Alert
            type="error"
            showIcon
            message={t("finance:seed.warning.entriesExist")}
          />
        )}

        {!hasEntries && hasAccounts && (
          <Alert
            type="warning"
            showIcon
            message={t("finance:seed.warning.accountsExist")}
            description={
              <div style={{ marginTop: 8 }}>
                <Checkbox
                  checked={confirmOverwrite}
                  onChange={(e) => setConfirmOverwrite(e.target.checked)}
                >
                  <Typography.Text strong>
                    {t("finance:seed.warning.confirmOverwrite")}
                  </Typography.Text>
                </Checkbox>
              </div>
            }
          />
        )}

        <Card>
          <Row gutter={[24, 16]}>
            <Col xs={24} sm={8}>
              <Typography.Text
                strong
                style={{ display: "block", marginBottom: 8 }}
              >
                {t("finance:seed.profile")}
              </Typography.Text>
              <Select
                style={{ width: "100%" }}
                value={profile}
                loading={isLoadingTemplates}
                onChange={(val) => setProfile(val)}
                options={
                  templates.length > 0
                    ? templates.map((t) => ({ label: t.name, value: t.id }))
                    : [{ label: "Freelancer", value: "freelancer" }]
                }
              />
            </Col>
            <Col xs={24} sm={8}>
              <Typography.Text
                strong
                style={{ display: "block", marginBottom: 8 }}
              >
                {t("finance:seed.currency")}
              </Typography.Text>
              <Select
                style={{ width: "100%" }}
                value={currency}
                onChange={(val) => setCurrency(val)}
                options={SUPPORTED_CURRENCIES.map((c) => ({
                  label: c,
                  value: c,
                }))}
              />
            </Col>
            <Col xs={24} sm={8}>
              <Typography.Text
                strong
                style={{ display: "block", marginBottom: 8 }}
              >
                {t("finance:seed.language")}
              </Typography.Text>
              <Radio.Group
                value={lang}
                onChange={(e) => setLang(e.target.value)}
                optionType="button"
                buttonStyle="solid"
              >
                <Radio.Button value="en">English</Radio.Button>
                <Radio.Button value="id">Bahasa Indonesia</Radio.Button>
              </Radio.Group>
            </Col>
          </Row>
        </Card>

        <Card
          title={
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
                flexWrap: "wrap",
                gap: 12,
              }}
            >
              <span>{t("finance:seed.preview")}</span>
              <Input.Search
                placeholder={t("finance:seed.searchPlaceholder")}
                allowClear
                style={{ width: 260 }}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
          }
        >
          {isLoadingPreview ? (
            <div style={{ textAlign: "center", padding: "40px 0" }}>
              <Spin size="large" />
            </div>
          ) : previewError ? (
            <Alert
              type="error"
              message={(previewError as Error).message}
              showIcon
            />
          ) : (
            <Table
              dataSource={filteredAccounts}
              columns={columns}
              rowKey="code"
              pagination={{
                pageSize: 15,
                showSizeChanger: true,
                pageSizeOptions: ["10", "15", "25", "50"],
              }}
              size="middle"
            />
          )}

          <div
            style={{
              display: "flex",
              justifyContent: "flex-end",
              marginTop: 20,
              paddingTop: 16,
              borderTop: "1px solid #f0f0f0",
            }}
          >
            <Button
              type="primary"
              size="large"
              loading={seedMutation.isPending}
              disabled={isSubmitDisabled}
              onClick={() => seedMutation.mutate()}
            >
              {seedMutation.isPending
                ? t("finance:seed.submitting")
                : t("finance:seed.submit")}
            </Button>
          </div>
        </Card>
      </Space>
    </div>
  );
}
