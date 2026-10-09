import {
  BankOutlined,
  PlusOutlined,
  RightOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import {
  Button,
  Card,
  Col,
  Empty,
  Row,
  Spin,
  Statistic,
  Typography,
} from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { useApi } from "~/shared/hooks/useApi";

export function meta() {
  return [{ title: "Finance" }];
}

export default function FinancePage() {
  const { t } = useTranslation(["finance", "common"]);
  const navigate = useNavigate();
  const { finance } = useApi();

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
          imageStyle={{ height: 72 }}
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
          <Button
            type="primary"
            size="large"
            icon={<PlusOutlined />}
            onClick={() => navigate("/fin/seed-coa")}
          >
            {t("finance:seed.onboarding.seedAction")}
          </Button>
        </Empty>
      </Card>
    );
  }

  const primaryCurrency = accounts[0]?.currency ?? "IDR";

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Typography.Title level={3} style={{ margin: 0 }}>
          {t("finance:title")}
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ margin: 0 }}>
          {t("finance:metaDescription")}
        </Typography.Paragraph>
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12}>
          <Card>
            <Statistic
              title={t("finance:accounts.totalAccounts")}
              value={accounts.length}
              prefix={
                <BankOutlined style={{ color: "#1677ff", marginRight: 8 }} />
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12}>
          <Card>
            <Statistic
              title={t("finance:seed.currency")}
              value={primaryCurrency}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} md={12}>
          <Card
            hoverable
            style={{ cursor: "pointer", height: "100%" }}
            onClick={() => navigate("/fin/accounts")}
          >
            <div style={{ display: "flex", alignItems: "flex-start", gap: 16 }}>
              <div
                style={{
                  background: "#e6f4ff",
                  borderRadius: 8,
                  padding: 12,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                }}
              >
                <BankOutlined style={{ fontSize: 28, color: "#1677ff" }} />
              </div>
              <div style={{ flex: 1 }}>
                <Typography.Title
                  level={4}
                  style={{ marginTop: 0, marginBottom: 4 }}
                >
                  {t("finance:modules.accountsTitle")}
                </Typography.Title>
                <Typography.Paragraph
                  type="secondary"
                  style={{ marginBottom: 12 }}
                >
                  {t("finance:modules.accountsDesc")}
                </Typography.Paragraph>
                <Button
                  type="primary"
                  onClick={(e) => {
                    e.stopPropagation();
                    navigate("/fin/accounts");
                  }}
                  icon={<RightOutlined />}
                  iconPosition="end"
                >
                  {t("finance:accounts.viewAccounts")}
                </Button>
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} md={12}>
          <Card
            hoverable
            style={{ cursor: "pointer", height: "100%" }}
            onClick={() => navigate("/fin/seed-coa")}
          >
            <div style={{ display: "flex", alignItems: "flex-start", gap: 16 }}>
              <div
                style={{
                  background: "#f6ffed",
                  borderRadius: 8,
                  padding: 12,
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "center",
                }}
              >
                <SettingOutlined style={{ fontSize: 28, color: "#52c41a" }} />
              </div>
              <div style={{ flex: 1 }}>
                <Typography.Title
                  level={4}
                  style={{ marginTop: 0, marginBottom: 4 }}
                >
                  {t("finance:modules.seedTitle")}
                </Typography.Title>
                <Typography.Paragraph
                  type="secondary"
                  style={{ marginBottom: 12 }}
                >
                  {t("finance:modules.seedDesc")}
                </Typography.Paragraph>
                <Button
                  onClick={(e) => {
                    e.stopPropagation();
                    navigate("/fin/seed-coa");
                  }}
                  icon={<RightOutlined />}
                  iconPosition="end"
                >
                  {t("finance:seed.title")}
                </Button>
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
