import { useTranslation } from "react-i18next";
import { UnderConstruction } from "~/shared/ui/UnderConstruction";

export function meta() {
  return [{ title: "Finance" }];
}

export default function FinancePage() {
  const { t } = useTranslation("finance");
  return <UnderConstruction pageTitle={t("title")} />;
}
