import { GithubOutlined, GoogleOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Divider, Space, Typography, theme } from "antd";
import { useState } from "react";
import { useSearchParams } from "react-router";
import { signIn } from "~/lib/auth";

type Provider = "google" | "github";

const PROVIDERS: { id: Provider; label: string; icon: React.ReactNode }[] = [
	{ id: "google", label: "Lanjut dengan Google", icon: <GoogleOutlined /> },
	{ id: "github", label: "Lanjut dengan GitHub", icon: <GithubOutlined /> },
];

// Hanya terima path internal supaya tidak jadi open redirect
function safeRedirect(value: string | null) {
	return value?.startsWith("/") && !value?.startsWith("//") ? value : "/";
}

export default function LoginPage() {
	const { token } = theme.useToken();
	const [params] = useSearchParams();
	const [loading, setLoading] = useState<Provider | null>(null);
	const [error, setError] = useState<string | null>(
		params.get("error")
			? "Gagal masuk. Coba lagi, atau pakai akun lain."
			: null,
	);

	const redirectTo = safeRedirect(params.get("redirect"));

	async function doSignIn(provider: Provider) {
		setError(null);
		setLoading(provider);
		try {
			const { error } = await signIn.social({
				provider,
				callbackURL: `${window.location.origin}${redirectTo}`,
				errorCallbackURL: `${window.location.origin}/login?error=1`,
			});
			if (error) throw error;
			// sukses: browser di-redirect ke provider, jadi loading dibiarkan
		} catch {
			setError(
				"Tidak bisa terhubung ke layanan login. Cek koneksi, lalu coba lagi.",
			);
			setLoading(null);
		}
	}

	return (
		<main
			style={{
				minHeight: "100vh",
				display: "flex",
				flexDirection: "column",
				alignItems: "center",
				justifyContent: "center",
				padding: 24,
				background: token.colorBgLayout,
			}}
		>
			<Card
				style={{ width: "100%", maxWidth: 400 }}
				styles={{ body: { padding: 40 } }}
			>
				<Space orientation="vertical" size={24} style={{ width: "100%" }}>
					<div style={{ textAlign: "center" }}>
						<img
							src="/logo-icon.svg"
							alt="O"
							width={48}
							height={48}
							style={{ display: "block", margin: "0 auto 20px" }}
						/>
						<Typography.Title level={3} style={{ margin: 0 }}>
							Masuk ke Omed
						</Typography.Title>
						<Typography.Text type="secondary">
							Pilih akun untuk lanjut ke dashboard.
						</Typography.Text>
					</div>

					{error && <Alert type="error" showIcon message={error} />}

					<Space direction="vertical" size={12} style={{ width: "100%" }}>
						{PROVIDERS.map((p) => (
							<Button
								key={p.id}
								block
								size="large"
								icon={p.icon}
								loading={loading === p.id}
								disabled={loading !== null && loading !== p.id}
								onClick={() => doSignIn(p.id)}
								style={{ height: 48 }}
							>
								{p.label}
							</Button>
						))}
					</Space>

					<Divider style={{ margin: 0 }} />

					<Typography.Paragraph
						type="secondary"
						style={{ margin: 0, textAlign: "center", fontSize: 12 }}
					>
						Dengan masuk, kamu setuju dengan <a href="/terms">Syarat Layanan</a>{" "}
						dan <a href="/privacy">Kebijakan Privasi</a>.
					</Typography.Paragraph>
				</Space>
			</Card>

			<Typography.Text type="secondary" style={{ marginTop: 24, fontSize: 13 }}>
				Ada kendala masuk? <a href="mailto:support@omed.app">Hubungi kami</a>
			</Typography.Text>
		</main>
	);
}
