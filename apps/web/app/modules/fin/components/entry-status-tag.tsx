import { Tag, type TagProps } from "antd";
import { useTranslation } from "react-i18next";

export type EntryStatus = "DRAFT" | "POSTED" | "VOID";

export type EntryStatusTagProps = Omit<TagProps, "color" | "children"> & {
  status?: string;
};

export function EntryStatusTag({ status, ...props }: EntryStatusTagProps) {
  const { t } = useTranslation();

  switch (status) {
    case "POSTED":
      return (
        <Tag color="success" {...props}>
          {t("fin.journal.POSTED", "Posted")}
        </Tag>
      );
    case "VOID":
      return (
        <Tag color="error" {...props}>
          {t("fin.journal.VOID", "Void")}
        </Tag>
      );
    default:
      return (
        <Tag color="default" {...props}>
          {t("fin.journal.DRAFT", "Draft")}
        </Tag>
      );
  }
}
