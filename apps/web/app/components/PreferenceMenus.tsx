import { BulbFilled, BulbOutlined, GlobalOutlined } from "@ant-design/icons";
import { Button, Dropdown } from "antd";
import { useTranslation } from "react-i18next";
import { setLanguage } from "~/i18n";
import type { Lang } from "~/lib/format";
import { type ThemePreference, useTheme } from "~/theme/theme-context";

export function LanguageMenu() {
  const { i18n, t } = useTranslation();
  const current = (i18n.language === "id" ? "id" : "en") as Lang;

  return (
    <Dropdown
      trigger={["click"]}
      menu={{
        selectedKeys: [current],
        items: [
          { key: "en", label: "English" },
          { key: "id", label: "Bahasa Indonesia" },
        ],
        onClick: ({ key }) => setLanguage(key as Lang),
      }}
    >
      <Button
        type="text"
        icon={<GlobalOutlined />}
        aria-label={t("shell.language")}
      >
        {current.toUpperCase()}
      </Button>
    </Dropdown>
  );
}

export function ThemeMenu() {
  const { t } = useTranslation();
  const { mode, preference, setPreference } = useTheme();

  return (
    <Dropdown
      trigger={["click"]}
      menu={{
        selectedKeys: [preference],
        items: [
          { key: "light", label: t("shell.themeLight") },
          { key: "dark", label: t("shell.themeDark") },
          { key: "system", label: t("shell.themeSystem") },
        ],
        onClick: ({ key }) => setPreference(key as ThemePreference),
      }}
    >
      <Button
        type="text"
        aria-label={t("shell.theme")}
        icon={mode === "dark" ? <BulbFilled /> : <BulbOutlined />}
      />
    </Dropdown>
  );
}
