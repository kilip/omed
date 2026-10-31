import { InputNumber, type InputNumberProps } from "antd";
import {
  type DecimalStyle,
  getActiveDecimalStyle,
  getCurrencyDecimals,
} from "~/lib/format";

export const MONEY_INPUT_MIN_WIDTH = 168;

export function formatMoneyInputValue(
  value: number | string,
  style: DecimalStyle,
  decimals: number,
): string {
  if (value === "" || value === null || value === undefined) {
    return "";
  }

  const thousandSep = style === "comma" ? "." : ",";
  const decimalSep = style === "comma" ? "," : ".";

  let rawInt = "";
  let rawDec: string | undefined;
  let hasTrailingDecimal = false;

  if (typeof value === "number") {
    if (Number.isNaN(value)) return "";
    const parts = String(value).split(".");
    rawInt = parts[0];
    rawDec = parts[1];
  } else {
    const str = String(value).trim();
    if (!str) return "";

    if (style === "comma") {
      if (str.includes(",")) {
        const parts = str.split(",");
        rawInt = parts[0].replaceAll(".", "");
        rawDec = parts.slice(1).join("");
        if (str.endsWith(",")) {
          hasTrailingDecimal = true;
        }
      } else if (str.includes(".")) {
        const parts = str.split(".");
        if (parts.length > 2 || (parts.length === 2 && parts[1].length === 3)) {
          rawInt = str.replaceAll(".", "");
        } else {
          rawInt = parts[0];
          rawDec = parts[1];
        }
      } else {
        rawInt = str;
      }
    } else {
      if (str.includes(".")) {
        const parts = str.split(".");
        rawInt = parts[0].replaceAll(",", "");
        rawDec = parts.slice(1).join("");
        if (str.endsWith(".")) {
          hasTrailingDecimal = true;
        }
      } else if (str.includes(",")) {
        const parts = str.split(",");
        if (parts.length > 2 || (parts.length === 2 && parts[1].length === 3)) {
          rawInt = str.replaceAll(",", "");
        } else {
          rawInt = parts[0];
          rawDec = parts[1];
        }
      } else {
        rawInt = str;
      }
    }
  }

  const isNeg = rawInt.startsWith("-");
  const digits = isNeg ? rawInt.slice(1) : rawInt;
  const formattedInt = digits.replace(/\B(?=(\d{3})+(?!\d))/g, thousandSep);
  const withSign = isNeg ? `-${formattedInt}` : formattedInt;

  if (decimals === 0) {
    return withSign;
  }

  if (rawDec !== undefined) {
    const slicedDec = rawDec.slice(0, decimals);
    if (slicedDec.length > 0) {
      return `${withSign}${decimalSep}${slicedDec}`;
    }
    if (hasTrailingDecimal) {
      return `${withSign}${decimalSep}`;
    }
    return withSign;
  }

  if (hasTrailingDecimal) {
    return `${withSign}${decimalSep}`;
  }

  return withSign;
}

export function parseMoneyInputValue(
  text: string,
  style: DecimalStyle,
): number | null {
  if (text === null || text === undefined) {
    return null;
  }
  const str = typeof text === "string" ? text.trim() : String(text).trim();
  if (!str) return null;

  let normalized: string;
  if (style === "comma") {
    const withoutThousands = str.replaceAll(".", "");
    if ((withoutThousands.match(/,/g) || []).length > 1) {
      return null;
    }
    normalized = withoutThousands.replace(",", ".");
  } else {
    const withoutThousands = str.replaceAll(",", "");
    if ((withoutThousands.match(/\./g) || []).length > 1) {
      return null;
    }
    normalized = withoutThousands;
  }

  const num = Number(normalized);
  if (Number.isNaN(num) || !Number.isFinite(num)) {
    return null;
  }
  return num;
}

export interface MoneyInputProps
  extends Omit<InputNumberProps<number>, "formatter" | "parser"> {
  currency?: string;
}

export function MoneyInput({ currency, style, ...props }: MoneyInputProps) {
  const decimalStyle = getActiveDecimalStyle() ?? "dot";
  const decimals = getCurrencyDecimals(currency);

  return (
    <InputNumber<number>
      {...props}
      style={{ minWidth: MONEY_INPUT_MIN_WIDTH, ...style }}
      formatter={(value) =>
        value !== undefined && value !== null
          ? formatMoneyInputValue(value, decimalStyle, decimals)
          : ""
      }
      parser={(displayValue) => {
        const parsed = parseMoneyInputValue(displayValue ?? "", decimalStyle);
        return (parsed ?? "") as unknown as number;
      }}
    />
  );
}
