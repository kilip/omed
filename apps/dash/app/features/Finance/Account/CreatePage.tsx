import { ArrowLeftOutlined } from "@ant-design/icons";
import { DEFAULT_CURRENCY } from "@omed/better-auth/schema";
import type { CreateAccountRequest } from "@omed/openapi/finance";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Card, message, Typography } from "antd";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router";
import { getCachedAuth } from "~/middleware/auth";
import { useApi } from "~/shared/hooks/useApi";
import { AccountForm, type AccountFormValues } from "./AccountForm";

export function meta() {
  return [{ title: "Create Account" }];
}

export default function CreateAccountPage() {
  const { t } = useTranslation(["finance", "common"]);
  const navigate = useNavigate();
  const { finance } = useApi();
  const queryClient = useQueryClient();

  const cachedUser = getCachedAuth()?.user;
  const initialCurrency =
    (cachedUser?.defaultCurrency as string) ?? DEFAULT_CURRENCY;

  // Fetch accounts to populate parent account select
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

  const createMutation = useMutation({
    mutationFn: async (values: AccountFormValues) => {
      const payload: CreateAccountRequest = {
        code: values.code.trim(),
        name: values.name.trim(),
        type: values.type,
        currency: values.currency,
        ...(values.description?.trim()
          ? { description: values.description.trim() }
          : {}),
        ...(values.parentId ? { parentId: values.parentId } : {}),
      };

      const res = await finance.POST("/accounts", {
        body: payload,
      });

      if (res.error) {
        const errorMsg =
          res.error.error?.message || t("finance:create.errorMessage");
        throw new Error(errorMsg);
      }

      return res.data?.data;
    },
    onSuccess: async () => {
      message.success(t("finance:create.successMessage"));
      await queryClient.invalidateQueries({ queryKey: ["accounts"] });
      navigate("/fin/accounts");
    },
    onError: (err: Error) => {
      message.error(err.message);
    },
  });

  const handleSubmit = (values: AccountFormValues) => {
    createMutation.mutate(values);
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
          {t("finance:create.backToAccounts")}
        </Button>
        <Typography.Title level={3} style={{ margin: 0 }}>
          {t("finance:create.title")}
        </Typography.Title>
        <Typography.Paragraph type="secondary" style={{ margin: "4px 0 0" }}>
          {t("finance:create.description")}
        </Typography.Paragraph>
      </div>

      <Card title={t("finance:create.cardTitle")}>
        <AccountForm
          mode="create"
          initialValues={{
            currency: initialCurrency,
            type: "asset",
          }}
          accounts={accounts}
          isLoadingAccounts={isLoadingAccounts}
          onSubmit={handleSubmit}
          isLoading={createMutation.isPending}
        />
      </Card>
    </div>
  );
}
