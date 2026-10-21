import { Tag, type TagProps } from "antd";
import { useTranslation } from "react-i18next";

export type ContentStatus = "DRAFT" | "SCHEDULED" | "PUBLISHED";

export type StatusTagProps = Omit<TagProps, "color" | "children"> & {
  status?: string;
  effectiveStatus?: string;
};

export function StatusTag({
  status,
  effectiveStatus,
  ...props
}: StatusTagProps) {
  const { t } = useTranslation();
  const current = (effectiveStatus || status || "DRAFT").toUpperCase();

  switch (current) {
    case "PUBLISHED":
      return (
        <Tag color="success" {...props}>
          {t("cms.status.PUBLISHED", "Published")}
        </Tag>
      );
    case "SCHEDULED":
      return (
        <Tag color="warning" {...props}>
          {t("cms.status.SCHEDULED", "Scheduled")}
        </Tag>
      );
    default:
      return (
        <Tag color="default" {...props}>
          {t("cms.status.DRAFT", "Draft")}
        </Tag>
      );
  }
}
