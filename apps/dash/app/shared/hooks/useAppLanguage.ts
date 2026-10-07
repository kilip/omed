import { useTranslation } from "react-i18next";
import { authClient } from "~/lib/auth";
import type { SupportedLanguage } from "~/lib/i18n";
import { getCachedAuth, updateCachedAuthUser } from "~/middleware/auth";

export function useAppLanguage() {
	const { i18n, t } = useTranslation();

	const activeLang = (i18n.resolvedLanguage ?? i18n.language ?? "en").split(
		"-",
	)[0] as SupportedLanguage;

	const changeLanguage = async (lang: SupportedLanguage) => {
		await i18n.changeLanguage(lang);
		const cached = getCachedAuth();
		if (cached) {
			updateCachedAuthUser({ locale: lang });
			try {
				await authClient.updateUser({ locale: lang });
			} catch (e) {
				console.error("Failed to sync locale to user:", e);
			}
		}
	};

	return {
		language: activeLang,
		changeLanguage,
		t,
		i18n,
	};
}
