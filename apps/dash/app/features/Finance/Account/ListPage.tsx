import {
  BankOutlined,
  EditOutlined,
  PlusOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import type { Account } from "@omed/openapi/finance";
import { useQuery } from "@tanstack/react-query";
import {
  Button,
  Card,
  Col,
  Empty,
  Input,
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
  return [{ title: "Chart of Accounts" }];
}

export default function AccountsPage() {
  const { t } = useTranslation(["finance", "common"]);
  const navigate = useNavigate();
  const { finance } = useApi();

  const [search, setSearch] = useState<string>("");
  const [selectedType, setSelectedType] = useState<string>("all");

  const { data: accounts = [], isLoading } = useQuery({
    queryKey: ["accounts"],
    queryFn: async () => {
      const res = await finance.GET("/accounts");
      if (res.error) {
        throw new Error("Failed to load accounts");
      }
      return res.data?.data ?? [];
    },
  });

  const filteredAccounts = useMemo(() => {
    return accounts.filter((a) => {
      const matchesSearch =
        !search.trim() ||
        (a.code ?? "").toLowerCase().includes(search.toLowerCase()) ||
        (a.name ?? "").toLowerCase().includes(search.toLowerCase());

      const matchesType =
        selectedType === "all" ||
        (a.type ?? "").toLowerCase() === selectedType.toLowerCase();

      return matchesSearch && matchesType;
    });
  }, [accounts, search, selectedType]);

  const columns: ColumnsType<Account> = [
    {
      title: t("finance:seed.table.code"),
      dataIndex: "code",
      key: "code",
      width: 140,
      render: (code: string, record) => (
        <Button
          type="link"
          style={{ padding: 0, height: "auto" }}
          onClick={() => navigate(`/fin/accounts/${record.id}`)}
        >
          <Typography.Text strong code>
            {code}
          </Typography.Text>
        </Button>
      ),
      sorter: (a, b) => (a.code ?? "").localeCompare(b.code ?? ""),
    },
    {
      title: t("finance:seed.table.name"),
      dataIndex: "name",
      key: "name",
      render: (name: string, record) => (
        <Button
          type="link"
          style={{ padding: 0, height: "auto", color: "inherit" }}
          onClick={() => navigate(`/fin/accounts/${record.id}`)}
        >
          {name}
        </Button>
      ),
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
    {
      title: t("finance:seed.currency"),
      dataIndex: "currency",
      key: "currency",
      width: 110,
      render: (currency: string) => <Tag>{currency}</Tag>,
    },
    {
      title: t("finance:accounts.action"),
      key: "action",
      width: 80,
      align: "center",
      render: (_, record) => (
        <Button
          type="text"
          icon={<EditOutlined />}
          onClick={() => navigate(`/fin/accounts/${record.id}`)}
          title={t("finance:accounts.edit")}
        />
      ),
    },
  ];

  if (isLoading) {
    return (
      <div style={{ textAlign: "center", padding: "80px 0" }}>
        <Spin size="large" />
      </div>
    );
  }

  if (accounts.length === 0) {
    return (
      <Card
        style={{ margin: "24px 0", textAlign: "center", padding: "40px 20px" }}
      >
        <Empty
          image={<BankOutlined style={{ fontSize: 64, color: "#1677ff" }} />}
          styles={{ image: { height: 72 } }}
          description={
            <div style={{ maxWidth: 480, margin: "0 auto" }}>
              <Typography.Title level={4} style={{ marginBottom: 8 }}>
                {t("finance:seed.onboarding.emptyTitle")}
              </Typography.Title>
              <Typography.Paragraph type="secondary">
                {t("finance:seed.onboarding.emptyDescription")}
              </Typography.Paragraph>
            </div>
          }
        >
          <Space>
            <Button
              type="primary"
              size="large"
              icon={<PlusOutlined />}
              onClick={() => navigate("/fin/seed-coa")}
            >
              {t("finance:seed.onboarding.seedAction")}
            </Button>
            <Button
              size="large"
              icon={<PlusOutlined />}
              onClick={() => navigate("/fin/accounts/create")}
            >
              {t("finance:seed.onboarding.manualCreateAction")}
            </Button>
          </Space>
        </Empty>
      </Card>
    );
  }

  return (
    <div>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 20,
          flexWrap: "wrap",
          gap: 12,
        }}
      >
        <div>
          <Typography.Title level={3} style={{ margin: 0 }}>
            {t("finance:accounts.title")}
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ margin: 0 }}>
            {t("finance:accounts.description")}
          </Typography.Paragraph>
        </div>
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate("/fin/accounts/create")}
          >
            {t("finance:create.title")}
          </Button>
          <Button
            icon={<SettingOutlined />}
            onClick={() => navigate("/fin/seed-coa")}
          >
            {t("finance:seed.title")}
          </Button>
        </Space>
      </div>

      <Card>
        <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
          <Col xs={24} sm={12} md={8}>
            <Input.Search
              placeholder={t("finance:accounts.searchPlaceholder")}
              allowClear
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              style={{ width: "100%" }}
              value={selectedType}
              onChange={(val) => setSelectedType(val)}
              options={[
                { label: t("finance:accounts.allTypes"), value: "all" },
                { label: t("finance:seed.types.asset"), value: "asset" },
                {
                  label: t("finance:seed.types.liability"),
                  value: "liability",
                },
                { label: t("finance:seed.types.equity"), value: "equity" },
                { label: t("finance:seed.types.revenue"), value: "revenue" },
                { label: t("finance:seed.types.expense"), value: "expense" },
              ]}
            />
          </Col>
        </Row>

        <Table
          dataSource={filteredAccounts}
          columns={columns}
          rowKey="id"
          pagination={{ pageSize: 15, showSizeChanger: true }}
        />
      </Card>
    </div>
  );
}
