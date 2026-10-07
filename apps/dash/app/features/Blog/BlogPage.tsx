import { useTranslation } from "react-i18next";
import { UnderConstruction } from "~/shared/ui/UnderConstruction";

export function meta() {
  return [{ title: "Blog" }];
}

export default function BlogPage() {
  const { t } = useTranslation("blog");
  return <UnderConstruction pageTitle={t("title")} />;
}
