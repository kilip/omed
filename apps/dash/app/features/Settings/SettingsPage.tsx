import { Card, Radio, Space, Typography } from "antd";
import { useTranslation } from "react-i18next";
import { SUPPORTED_LANGUAGES } from "~/lib/i18n";
import { useAppLanguage } from "~/shared/hooks/useAppLanguage";
import { useThemeMode } from "~/shared/ui/ThemeProvider";

export function meta() {
	return [{ title: "Settings" }];
}

export default function SettingsPage() {
	const { t } = useTranslation(["settings", "common"]);
	const { language: activeLang, changeLanguage } = useAppLanguage();
	const { mode, setMode } = useThemeMode();

	return (
		<div>
			<Typography.Title level={3} style={{ marginTop: 0, marginBottom: 24 }}>
				{t("settings:title")}
			</Typography.Title>

			<Space direction="vertical" size={20} style={{ width: "100%" }}>
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
