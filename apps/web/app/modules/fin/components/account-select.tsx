import type { components } from "@omed/openapi";
import { Select, type SelectProps } from "antd";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";

type Account = components["schemas"]["model.AccountResponse"];

export type AccountSelectProps = Omit<SelectProps<number>, "options"> & {
  accounts: Account[];
};

export function AccountSelect({
  accounts,
  placeholder,
  ...props
}: AccountSelectProps) {
  const { t } = useTranslation();

  const options = useMemo(() => {
    return accounts
      .filter((a) => a.isPostable && a.isActive)
      .map((a) => ({
        value: a.id as number,
        label: `${a.code} - ${a.name}`,
        account: a,
      }));
  }, [accounts]);

  return (
    <Select
      showSearch
      allowClear
      placeholder={placeholder ?? t("fin.journal.account")}
      optionFilterProp="label"
      filterOption={(input, option) =>
        (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
      }
      options={options}
      {...props}
    />
  );
}
