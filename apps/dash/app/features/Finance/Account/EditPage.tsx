import { ArrowLeftOutlined, DeleteOutlined } from "@ant-design/icons";
import type { UpdateAccountRequest } from "@omed/openapi/finance";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Alert,
  Button,
  Card,
  message,
  Popconfirm,
  Spin,
  Typography,
} from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router";
import { useApi } from "~/shared/hooks/useApi";
import { AccountForm, type AccountFormValues } from "./AccountForm";

export function meta() {
  return [{ title: "Edit Account" }];
}

export default function EditAccountPage() {
  const { id } = useParams<{ id: string }>();
  const { t } = useTranslation(["finance", "common"]);
  const navigate = useNavigate();
  const { finance } = useApi();
  const queryClient = useQueryClient();

  // Fetch single account detail
  const {
    data: account,
    isLoading: isLoadingAccount,
    error: accountError,
  } = useQuery({
    queryKey: ["accounts", id],
    queryFn: async () => {
      if (!id) throw new Error("Account ID is required");
      const res = await finance.GET("/accounts/{id}", {
        params: {
          path: { id },
        },
      });
      if (res.error) {
        throw new Error(res.error.error?.message || t("finance:edit.notFound"));
      }
      return res.data?.data;
    },
    enabled: !!id,
  });

  // Fetch accounts list for parent account select
  const { data: accounts = [], isLoading: isLoadingAccounts } = useQuery({
    queryKey: ["accounts"],
    queryFn: async () => {
      const res = await finance.GET("/accounts");
      if (res.error) {
        throw new Error("Failed to load accounts");
      }
      return res.data?.data ?? [];
    },
  });

  const updateMutation = useMutation({
    mutationFn: async (values: AccountFormValues) => {
      if (!id) throw new Error("Account ID is required");
      const payload: UpdateAccountRequest = {
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
        parentId: values.parentId || undefined,
        status: values.status,
      };

      const res = await finance.PUT("/accounts/{id}", {
        params: {
          path: { id },
        },
        body: payload,
      });

      if (res.error) {
        const errorMsg =
          res.error.error?.message || t("finance:edit.errorMessage");
        throw new Error(errorMsg);
      }

      return res.data?.data;
    },
    onSuccess: async () => {
      message.success(t("finance:edit.successMessage"));
      await queryClient.invalidateQueries({ queryKey: ["accounts"] });
      navigate("/fin/accounts");
    },
    onError: (err: Error) => {
      message.error(err.message);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async () => {
      if (!id) throw new Error("Account ID is required");
      const res = await finance.DELETE("/accounts/{id}", {
        params: {
          path: { id },
        },
      });

      if (res.error) {
        const errorMsg =
          res.error.error?.message || t("finance:delete.errorMessage");
        throw new Error(errorMsg);
      }

      return res.data;
    },
    onSuccess: async () => {
      message.success(t("finance:delete.successMessage"));
      await queryClient.invalidateQueries({ queryKey: ["accounts"] });
      navigate("/fin/accounts");
    },
    onError: (err: Error) => {
      message.error(err.message);
    },
  });

  const handleSubmit = (values: AccountFormValues) => {
    updateMutation.mutate(values);
  };

  if (isLoadingAccount) {
    return (
      <div style={{ textAlign: "center", padding: "80px 0" }}>
        <Spin size="large" />
      </div>
    );
  }

  if (accountError || !account) {
    return (
      <div style={{ maxWidth: 800, margin: "0 auto" }}>
        <Button
          type="link"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate("/fin/accounts")}
          style={{ paddingLeft: 0, marginBottom: 16 }}
        >
          {t("finance:edit.backToAccounts")}
        </Button>
        <Alert
          type="error"
          message={t("finance:edit.notFound")}
          description={
            accountError instanceof Error ? accountError.message : undefined
          }
          showIcon
        />
      </div>
    );
  }

  const initialValues: Partial<AccountFormValues> = {
    code: account.code,
    name: account.name,
    type: account.type,
    currency: account.currency,
    description: account.description ?? undefined,
    parentId: account.parentId ?? undefined,
    status: account.status,
  };

  return (
    <div style={{ maxWidth: 800, margin: "0 auto" }}>
      <div style={{ marginBottom: 20 }}>
        <Button
          type="link"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate("/fin/accounts")}
          style={{ paddingLeft: 0, marginBottom: 8 }}
        >
          {t("finance:edit.backToAccounts")}
        </Button>
        <Typography.Title level={3} style={{ margin: 0 }}>
          {t("finance:edit.title")}
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ margin: "4px 0 0" }}>
          {t("finance:edit.description")}
        </Typography.Paragraph>
      </div>

      <Card title={t("finance:edit.cardTitle")}>
        <AccountForm
          mode="edit"
          initialValues={initialValues}
          currentAccountId={account.id}
          accounts={accounts}
          isLoadingAccounts={isLoadingAccounts}
          onSubmit={handleSubmit}
          isLoading={updateMutation.isPending}
          extraActions={
            <Popconfirm
              title={t("finance:delete.confirmTitle")}
              description={t("finance:delete.confirmDescription")}
              onConfirm={() => deleteMutation.mutate()}
              okText={t("finance:delete.confirmOk")}
              cancelText={t("finance:delete.confirmCancel")}
              okButtonProps={{
                danger: true,
                loading: deleteMutation.isPending,
              }}
            >
              <Button
                danger
                icon={<DeleteOutlined />}
                loading={deleteMutation.isPending}
              >
                {t("finance:delete.button")}
              </Button>
            </Popconfirm>
          }
        />
      </Card>
    </div>
  );
}
