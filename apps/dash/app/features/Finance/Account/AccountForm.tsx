import { PlusOutlined, SaveOutlined } from "@ant-design/icons";
import { SUPPORTED_CURRENCIES } from "@omed/better-auth/schema";
import type {
  Account,
  AccountStatus,
  AccountType,
} from "@omed/openapi/finance";
import { Button, Col, Form, Input, Row, Select, Space } from "antd";
import type React from "react";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";

export interface AccountFormValues {
  code: string;
  name: string;
  type: AccountType;
  currency: string;
  description?: string;
  parentId?: string;
  status?: AccountStatus;
}

export interface AccountFormProps {
  mode: "create" | "edit";
  initialValues?: Partial<AccountFormValues>;
  currentAccountId?: string;
  accounts?: Account[];
  isLoadingAccounts?: boolean;
  onSubmit: (values: AccountFormValues) => Promise<void> | void;
  isLoading?: boolean;
  extraActions?: React.ReactNode;
}

export const AccountForm: React.FC<AccountFormProps> = ({
  mode,
  initialValues,
  currentAccountId,
  accounts = [],
  isLoadingAccounts = false,
  onSubmit,
  isLoading = false,
  extraActions,
}) => {
  const { t } = useTranslation(["finance", "common"]);
  const navigate = useNavigate();
  const [form] = Form.useForm<AccountFormValues>();

  const isEdit = mode === "edit";
  const watchedType = Form.useWatch("type", form);

  useEffect(() => {
    if (initialValues) {
      form.setFieldsValue(initialValues);
    }
  }, [form, initialValues]);

  // Filter available parent accounts matching selected type and exclude current account
  const parentOptions = useMemo(() => {
    return accounts
      .filter((a) => {
        if (currentAccountId && a.id === currentAccountId) {
          return false;
        }
        return !watchedType || a.type === watchedType;
      })
      .map((a) => ({
        label: `${a.code} - ${a.name}`,
        value: a.id,
      }));
  }, [accounts, watchedType, currentAccountId]);

  return (
    <Form
      form={form}
      layout="vertical"
      onFinish={onSubmit}
      initialValues={initialValues}
    >
      <Row gutter={16}>
        <Col xs={24} sm={12}>
          <Form.Item
            name="code"
            label={t("finance:create.code")}
            rules={[
              {
                required: true,
                message: t("finance:create.codeRequired"),
              },
            ]}
          >
            <Input
              placeholder={t("finance:create.codePlaceholder")}
              allowClear={!isEdit}
              disabled={isEdit}
            />
          </Form.Item>
        </Col>
        <Col xs={24} sm={12}>
          <Form.Item
            name="name"
            label={t("finance:create.name")}
            rules={[
              {
                required: true,
                message: t("finance:create.nameRequired"),
              },
            ]}
          >
            <Input
              placeholder={t("finance:create.namePlaceholder")}
              allowClear
            />
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col xs={24} sm={12}>
          <Form.Item
            name="type"
            label={t("finance:create.type")}
            rules={[
              {
                required: true,
                message: t("finance:create.typeRequired"),
              },
            ]}
          >
            <Select
              placeholder={t("finance:create.typePlaceholder")}
              disabled={isEdit}
              options={[
                {
                  label: t("finance:seed.types.asset"),
                  value: "asset",
                },
                {
                  label: t("finance:seed.types.liability"),
                  value: "liability",
                },
                {
                  label: t("finance:seed.types.equity"),
                  value: "equity",
                },
                {
                  label: t("finance:seed.types.revenue"),
                  value: "revenue",
                },
                {
                  label: t("finance:seed.types.expense"),
                  value: "expense",
                },
              ]}
              onChange={() => {
                // Reset parentId if the type changes
                form.setFieldValue("parentId", undefined);
              }}
            />
          </Form.Item>
        </Col>
        <Col xs={24} sm={12}>
          <Form.Item
            name="currency"
            label={t("finance:create.currency")}
            rules={[
              {
                required: true,
                message: t("finance:create.currencyRequired"),
              },
            ]}
          >
            <Select
              placeholder={t("finance:create.currencyPlaceholder")}
              disabled={isEdit}
              options={SUPPORTED_CURRENCIES.map((c) => ({
                label: c,
                value: c,
              }))}
            />
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col xs={24} sm={isEdit ? 12 : 24}>
          <Form.Item name="parentId" label={t("finance:create.parentAccount")}>
            <Select
              allowClear
              placeholder={t("finance:create.parentAccountPlaceholder")}
              options={parentOptions}
              loading={isLoadingAccounts}
              disabled={parentOptions.length === 0}
            />
          </Form.Item>
        </Col>
        {isEdit && (
          <Col xs={24} sm={12}>
            <Form.Item
              name="status"
              label={t("finance:edit.status")}
              rules={[
                {
                  required: true,
                },
              ]}
            >
              <Select
                options={[
                  {
                    label: t("finance:edit.statusActive"),
                    value: "active",
                  },
                  {
                    label: t("finance:edit.statusArchived"),
                    value: "archived",
                  },
                ]}
              />
            </Form.Item>
          </Col>
        )}
      </Row>

      <Form.Item
        name="description"
        label={t("finance:create.descriptionLabel")}
      >
        <Input.TextArea
          rows={3}
          placeholder={t("finance:create.descriptionPlaceholder")}
          allowClear
        />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0, marginTop: 12 }}>
        <div
          style={{
            display: "flex",
            justifyContent: extraActions ? "space-between" : "flex-end",
            alignItems: "center",
            flexWrap: "wrap",
            gap: 12,
          }}
        >
          {extraActions ? <div>{extraActions}</div> : <div />}
          <Space>
            <Button onClick={() => navigate("/fin/accounts")}>
              {t("finance:create.cancel")}
            </Button>
            <Button
              type="primary"
              htmlType="submit"
              icon={isEdit ? <SaveOutlined /> : <PlusOutlined />}
              loading={isLoading}
            >
              {isEdit ? t("finance:edit.submit") : t("finance:create.submit")}
            </Button>
          </Space>
        </div>
      </Form.Item>
    </Form>
  );
};
