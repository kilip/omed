import {
	BellOutlined,
	BulbFilled,
	BulbOutlined,
	HomeOutlined,
	LogoutOutlined,
	MenuFoldOutlined,
	MenuUnfoldOutlined,
	ReadOutlined,
	SettingOutlined,
	UserOutlined,
	WalletOutlined,
} from "@ant-design/icons";
import {
	Avatar,
	Badge,
	Breadcrumb,
	Button,
	Drawer,
	Dropdown,
	Grid,
	Layout,
	Menu,
	type MenuProps,
	Space,
	Tooltip,
	Typography,
	theme,
} from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useLocation, useNavigate } from "react-router";
import { signOut } from "~/lib/auth";
import { LanguageSwitcher } from "./LanguageSwitcher";
import { useThemeMode } from "./ThemeProvider";

const { Sider, Header, Content } = Layout;

function Logo({ collapsed }: { collapsed: boolean }) {
	return (
		<Link
			to="/"
			style={{
				display: "flex",
				alignItems: "center",
				gap: 10,
				height: 64,
				padding: collapsed ? "0 0 0 22px" : "0 24px",
				textDecoration: "none",
			}}
		>
			<img src="logo-icon.svg" width={32} height={32} alt="O" />
			{!collapsed && (
				<Typography.Text strong style={{ fontSize: 18, letterSpacing: -0.2 }}>
					Omed
				</Typography.Text>
			)}
		</Link>
	);
}

function SideMenu({ onNavigate }: { onNavigate?: () => void }) {
	const { pathname } = useLocation();
	const { t } = useTranslation("nav");
	const selected = `/${pathname.split("/")[1] ?? ""}`;

	const navItems = [
		{ key: "/", label: t("home"), icon: <HomeOutlined />, to: "/" },
		{
			key: "/fin",
			label: t("finance"),
			icon: <WalletOutlined />,
			to: "/fin",
		},
		{ key: "/blog", label: t("blog"), icon: <ReadOutlined />, to: "/blog" },
		{
			key: "/settings",
			label: t("settings"),
			icon: <SettingOutlined />,
			to: "/settings",
		},
	];

	const menuItems: MenuProps["items"] = navItems.map((n) => ({
		key: n.key,
		icon: n.icon,
		label: <Link to={n.to}>{n.label}</Link>,
	}));

	return (
		<Menu
			mode="inline"
			items={menuItems}
			selectedKeys={[selected]}
			onClick={onNavigate}
			style={{ border: 0 }}
		/>
	);
}

function AppBreadcrumb() {
	const { pathname } = useLocation();
	const { t } = useTranslation("nav");
	const segments = pathname.split("/").filter(Boolean);

	const segmentLabels: Record<string, string> = {
		home: t("home"),
		fin: t("finance"),
		blog: t("blog"),
		settings: t("settings"),
	};

	const items = [
		{
			title: segments.length ? <Link to="/">{t("home")}</Link> : t("home"),
		},
		...segments.map((seg, i) => {
			const label =
				segmentLabels[seg.toLowerCase()] ??
				seg.charAt(0).toUpperCase() + seg.slice(1);
			const to = `/${segments.slice(0, i + 1).join("/")}`;
			return {
				title: i === segments.length - 1 ? label : <Link to={to}>{label}</Link>,
			};
		}),
	];
	return <Breadcrumb items={items} />;
}

export default function DashboardLayout() {
	const [collapsed, setCollapsed] = useState(false);
	const [drawerOpen, setDrawerOpen] = useState(false);
	const { t } = useTranslation(["common", "nav"]);
	const { mode, toggle } = useThemeMode();
	const { token } = theme.useToken();
	const navigate = useNavigate();
	const screens = Grid.useBreakpoint();
	const isMobile = screens.lg === false;

	const userMenu: MenuProps = {
		items: [
			{ key: "profile", icon: <UserOutlined />, label: t("nav:profile") },
			{ type: "divider" },
			{
				key: "logout",
				icon: <LogoutOutlined />,
				label: t("nav:logout"),
				danger: true,
			},
		],
		onClick: async ({ key }) => {
			if (key === "profile") navigate("/settings");
			if (key === "logout") {
				// TODO: authClient.signOut() ke apps/auth, lalu redirect ke login
				await signOut({
					fetchOptions: {
						onSuccess: () => {
							navigate("/login");
						},
					},
				});
			}
		},
	};

	const toggleSider = () =>
		isMobile ? setDrawerOpen(true) : setCollapsed((c) => !c);

	return (
		<Layout style={{ minHeight: "100vh" }}>
			{!isMobile && (
				<Sider
					width={248}
					collapsedWidth={76}
					collapsed={collapsed}
					trigger={null}
					style={{
						borderRight: `1px solid ${token.colorBorderSecondary}`,
						position: "sticky",
						top: 0,
						height: "100vh",
					}}
				>
					<Logo collapsed={collapsed} />
					<SideMenu />
				</Sider>
			)}

			{isMobile && (
				<Drawer
					placement="left"
					size={264}
					open={drawerOpen}
					onClose={() => setDrawerOpen(false)}
					closable={false}
					styles={{ body: { padding: 0 }, header: { display: "none" } }}
				>
					<Logo collapsed={false} />
					<SideMenu onNavigate={() => setDrawerOpen(false)} />
				</Drawer>
			)}

			<Layout>
				<Header
					style={{
						position: "sticky",
						top: 0,
						zIndex: 10,
						display: "flex",
						alignItems: "center",
						justifyContent: "space-between",
						borderBottom: `1px solid ${token.colorBorderSecondary}`,
						padding: isMobile ? "0 16px" : "0 24px",
					}}
				>
					<Space size={16}>
						<Button
							type="text"
							aria-label={t("common:toggleMenu")}
							icon={
								collapsed && !isMobile ? (
									<MenuUnfoldOutlined />
								) : (
									<MenuFoldOutlined />
								)
							}
							onClick={toggleSider}
						/>
						{!isMobile && <AppBreadcrumb />}
					</Space>

					<Space size={6}>
						<LanguageSwitcher />
						<Tooltip
							title={
								mode === "light"
									? t("common:theme.switchToDark")
									: t("common:theme.switchToLight")
							}
						>
							<Button
								type="text"
								shape="circle"
								aria-label={t("common:theme.toggle")}
								icon={mode === "light" ? <BulbOutlined /> : <BulbFilled />}
								onClick={toggle}
							/>
						</Tooltip>
						<Tooltip title={t("common:notifications")}>
							<Badge dot offset={[-6, 6]}>
								<Button
									type="text"
									shape="circle"
									aria-label={t("common:notifications")}
									icon={<BellOutlined />}
								/>
							</Badge>
						</Tooltip>
						<Dropdown
							menu={userMenu}
							trigger={["click"]}
							placement="bottomRight"
						>
							<Button
								type="text"
								shape="circle"
								aria-label={t("common:userMenu")}
								style={{ marginLeft: 8 }}
							>
								<Avatar
									size={32}
									style={{
										background: token.colorPrimaryBg,
										color: token.colorPrimary,
										fontWeight: 600,
									}}
								>
									T
								</Avatar>
							</Button>
						</Dropdown>
					</Space>
				</Header>

				<Content style={{ padding: isMobile ? 16 : 24 }}>
					{isMobile && (
						<div style={{ marginBottom: 16 }}>
							<AppBreadcrumb />
						</div>
					)}
					<div style={{ maxWidth: 1280, margin: "0 auto" }}>
						<Outlet />
					</div>
				</Content>
			</Layout>
		</Layout>
	);
}
