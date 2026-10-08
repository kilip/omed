import { CheckOutlined, GlobalOutlined } from "@ant-design/icons";
import { Button, Dropdown, type MenuProps } from "antd";
import { SUPPORTED_LANGUAGES } from "~/lib/i18n";
import { useAppLanguage } from "~/shared/hooks/useAppLanguage";

export interface LanguageSwitcherProps {
  /** Show full label or short code */
  showLabel?: boolean;
  size?: "small" | "medium" | "middle" | "large";
}

export function LanguageSwitcher({
  showLabel = false,
  size = "medium",
}: LanguageSwitcherProps) {
  const { language: activeLang, changeLanguage, t } = useAppLanguage();

  const items: MenuProps["items"] = SUPPORTED_LANGUAGES.map((lang) => ({
    key: lang.code,
    label: lang.label,
    icon:
      activeLang === lang.code ? (
        <CheckOutlined />
      ) : (
        <span style={{ width: 14, display: "inline-block" }} />
      ),
    onClick: () => {
      changeLanguage(lang.code);
    },
  }));

  const currentLabel =
    SUPPORTED_LANGUAGES.find((l) => l.code === activeLang)?.label ?? "English";

  return (
    <Dropdown
      menu={{
        items,
        selectedKeys: [activeLang],
      }}
      placement="bottomRight"
      trigger={["click"]}
    >
      <Button
        type="text"
        size={size}
        aria-label={t("language.switch")}
        icon={<GlobalOutlined />}
      >
        {showLabel ? currentLabel : activeLang.toUpperCase()}
      </Button>
    </Dropdown>
  );
}
