import {
  DEFAULT_CURRENCY,
  SUPPORTED_CURRENCIES,
} from "@omed/better-auth/schema";
import { Card, message, Radio, Select, Space, Typography } from "antd";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { authClient } from "~/lib/auth";
import { SUPPORTED_LANGUAGES } from "~/lib/i18n";
import { getCachedAuth, updateCachedAuthUser } from "~/middleware/auth";
import { useAppLanguage } from "~/shared/hooks/useAppLanguage";
import { useThemeMode } from "~/shared/ui/ThemeProvider";

export function meta() {
  return [{ title: "Settings" }];
}

export default function SettingsPage() {
  const { t } = useTranslation(["settings", "common"]);
  const { language: activeLang, changeLanguage } = useAppLanguage();
  const { mode, setMode } = useThemeMode();

  const cachedUser = getCachedAuth()?.user;
  const [currency, setCurrency] = useState<string>(
    (cachedUser?.defaultCurrency as string) ?? DEFAULT_CURRENCY,
  );
  const [savingCurrency, setSavingCurrency] = useState(false);

  useEffect(() => {
    if (cachedUser?.defaultCurrency) {
      setCurrency(cachedUser.defaultCurrency as string);
    }
  }, [cachedUser?.defaultCurrency]);

  async function handleCurrencyChange(value: string) {
    const prev = currency;
    setCurrency(value);
    setSavingCurrency(true);
    try {
      const { error } = await authClient.updateUser({ defaultCurrency: value });
      if (error) {
        setCurrency(prev);
        message.error(t("settings:currencySaveFailed"));
        return;
      }
      updateCachedAuthUser({ defaultCurrency: value });
      message.success(t("settings:currencySaved"));
    } catch {
      setCurrency(prev);
      message.error(t("settings:currencySaveFailed"));
    } finally {
      setSavingCurrency(false);
    }
  }

  return (
    <div>
      <Typography.Title level={3} style={{ marginTop: 0, marginBottom: 24 }}>
        {t("settings:title")}
      </Typography.Title>

      <Space orientation="vertical" size={20} style={{ width: "100%" }}>
        <Card title={t("settings:language")}>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
            {t("settings:languageDescription")}
          </Typography.Paragraph>
          <Radio.Group
            value={activeLang}
            onChange={(e) => changeLanguage(e.target.value)}
          >
            {SUPPORTED_LANGUAGES.map((lang) => (
              <Radio key={lang.code} value={lang.code}>
                {lang.label}
              </Radio>
            ))}
          </Radio.Group>
        </Card>

        <Card title={t("settings:currency")}>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
            {t("settings:currencyDescription")}
          </Typography.Paragraph>
          <Select
            aria-label={t("settings:currency")}
            value={currency}
            loading={savingCurrency}
            disabled={savingCurrency}
            style={{ width: 180 }}
            onChange={handleCurrencyChange}
            options={SUPPORTED_CURRENCIES.map((c) => ({
              label: c,
              value: c,
            }))}
          />
        </Card>

        <Card title={t("settings:appearance")}>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
            {t("settings:appearanceDescription")}
          </Typography.Paragraph>
          <Radio.Group value={mode} onChange={(e) => setMode(e.target.value)}>
            <Radio value="light">{t("common:theme.light")}</Radio>
            <Radio value="dark">{t("common:theme.dark")}</Radio>
          </Radio.Group>
        </Card>
      </Space>
    </div>
  );
}
