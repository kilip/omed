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
import { Link, Outlet, useLocation, useNavigate } from "react-router";
import { signOut } from "~/lib/auth";
import { useThemeMode } from "./ThemeProvider";

const { Sider, Header, Content } = Layout;

const NAV = [
	{ key: "/", label: "Home", icon: <HomeOutlined />, to: "/" },
	{
		key: "/fin",
		label: "Finance",
		icon: <WalletOutlined />,
		to: "/fin",
	},
	{ key: "/blog", label: "Blog", icon: <ReadOutlined />, to: "/blog" },
	{
		key: "/settings",
		label: "Settings",
		icon: <SettingOutlined />,
		to: "/settings",
	},
];

const menuItems: MenuProps["items"] = NAV.map((n) => ({
	key: n.key,
	icon: n.icon,
	label: <Link to={n.to}>{n.label}</Link>,
}));

function Logo({ collapsed }: { collapsed: boolean }) {
	const { token } = theme.useToken();
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
	const selected = `/${pathname.split("/")[1] ?? ""}`;
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
	const segments = pathname.split("/").filter(Boolean);
	const items = [
		{ title: segments.length ? <Link to="/"></Link> : "Home" },
		...segments.map((seg, i) => {
			const label = seg.charAt(0).toUpperCase() + seg.slice(1);
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
	const { mode, toggle } = useThemeMode();
	const { token } = theme.useToken();
	const navigate = useNavigate();
	const screens = Grid.useBreakpoint();
	const isMobile = screens.lg === false;

	const userMenu: MenuProps = {
		items: [
			{ key: "profile", icon: <UserOutlined />, label: "Profile" },
			{ type: "divider" },
			{
				key: "logout",
				icon: <LogoutOutlined />,
				label: "Log out",
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
							aria-label="Toggle menu"
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

					<Space size={4}>
						<Tooltip
							title={mode === "light" ? "Switch to dark" : "Switch to light"}
						>
							<Button
								type="text"
								shape="circle"
								aria-label="Toggle theme"
								icon={mode === "light" ? <BulbOutlined /> : <BulbFilled />}
								onClick={toggle}
							/>
						</Tooltip>
						<Tooltip title="Notifications">
							<Badge dot offset={[-6, 6]}>
								<Button
									type="text"
									shape="circle"
									aria-label="Notifications"
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
								aria-label="User menu"
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
