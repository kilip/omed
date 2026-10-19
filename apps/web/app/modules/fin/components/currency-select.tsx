import type { components } from "@omed/openapi";
import { Select, type SelectProps } from "antd";
import { useMemo } from "react";

type Currency = components["schemas"]["model.CurrencyResponse"];

export type CurrencySelectProps = Omit<SelectProps<string>, "options"> & {
  currencies: Currency[];
};

export function CurrencySelect({
  currencies,
  placeholder = "Select Currency",
  ...props
}: CurrencySelectProps) {
  const options = useMemo(() => {
    return currencies.map((c) => ({
      value: c.code as string,
      label: `${c.code} - ${c.name} (${c.symbol})`,
      currency: c,
    }));
  }, [currencies]);

  return (
    <Select
      showSearch
      placeholder={placeholder}
      optionFilterProp="label"
      filterOption={(input, option) =>
        (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
      }
      options={options}
      {...props}
    />
  );
}
