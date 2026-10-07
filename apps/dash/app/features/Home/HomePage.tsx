import { useTranslation } from "react-i18next";
import { UnderConstruction } from "~/shared/ui/UnderConstruction";

export function meta() {
	return [
		{ title: "Omed | Home" },
		{ name: "description", content: "Welcome to Omed Dashboard" },
	];
}

export default function Home() {
	const { t } = useTranslation("home");
	return <UnderConstruction pageTitle={t("title")} />;
}
