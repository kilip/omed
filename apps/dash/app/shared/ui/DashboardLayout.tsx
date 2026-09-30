import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  MoonOutlined,
  SunOutlined,
  UserOutlined,
} from "@ant-design/icons";
import {
  Avatar,
  Button,
  Dropdown,
  Layout,
  Menu,
  type MenuProps,
  Space,
} from "antd";
import { type PropsWithChildren, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { clearAuthCache, signOut } from "../auth";
import { useAuth } from "../providers/AuthProvider";
import { useThemeMode } from "../providers/TeamProvider";
import { dashboardMenuItems, findParentKey } from "./menu-items";

const { Header, Sider, Content } = Layout;

const SIDER_WIDTH = 248;
const SIDER_WIDTH_COLLAPSED = 64;

export function DashboardLayout({ children }: PropsWithChildren) {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const [openKeys, setOpenKeys] = useState<string[]>(() => {
    const parent = findParentKey(location.pathname);
    return parent ? [parent] : [];
  });
  const { user } = useAuth();
  const { mode, toggle } = useThemeMode();

  const userMenu: MenuProps = {
    items: [
      { key: "profile", icon: <UserOutlined />, label: "Profile" },
      { type: "divider" as const },
      {
        key: "logout",
        icon: <LogoutOutlined />,
        danger: true,
        label: "Log out",
      },
    ],
    onClick: async (e) => {
      if (e.key === "logout") {
        await signOut({
          fetchOptions: {
            onSuccess() {
              clearAuthCache();
              navigate("/login", { replace: true });
            },
          },
        });
      }
    },
  };

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider
        className="app-sider"
        collapsible
        collapsed={collapsed}
        trigger={null}
        width={SIDER_WIDTH}
        collapsedWidth={SIDER_WIDTH_COLLAPSED}
        style={{
          position: "fixed",
          insetInlineStart: 0,
          top: 0,
          bottom: 0,
          zIndex: 100,
          overflow: "auto",
        }}
      >
        <div
          style={{
            height: 56,
            display: "flex",
            alignItems: "center",
            justifyContent: collapsed ? "center" : "flex-start",
            paddingInline: collapsed ? 0 : 20,
            gap: 10,
            fontWeight: 600,
            fontSize: 16,
            color: "var(--text)",
          }}
        >
          <img src="/omed-mark.svg" alt="Omed" width={24} height={24} />
          {!collapsed && <span>Omed</span>}
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          openKeys={collapsed ? undefined : openKeys}
          onOpenChange={(keys) => setOpenKeys(keys)}
          items={dashboardMenuItems}
          onClick={({ key }) => navigate(key)}
          style={{ background: "transparent", borderInlineEnd: 0 }}
        />
      </Sider>

      <Layout
        style={{
          marginInlineStart: collapsed ? SIDER_WIDTH_COLLAPSED : SIDER_WIDTH,
          transition: "margin-inline-start 0.2s",
          background: "transparent",
        }}
      >
        <Header
          className="app-header"
          style={{
            position: "sticky",
            top: 0,
            zIndex: 99,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <Button
            type="text"
            aria-label="Toggle sidebar"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed((v) => !v)}
          />
          <Space size="small">
            <Button
              type="text"
              shape="circle"
              aria-label="Toggle theme"
              icon={mode === "dark" ? <SunOutlined /> : <MoonOutlined />}
              onClick={toggle}
            />
            <Dropdown
              menu={userMenu}
              placement="bottomRight"
              trigger={["click"]}
            >
              <Avatar
                size={32}
                src={user.image ?? undefined}
                style={{ cursor: "pointer" }}
              >
                {user.name?.[0]?.toUpperCase()}
              </Avatar>
            </Dropdown>
          </Space>
        </Header>

        <Content>
          <div className="app-content">{children}</div>
        </Content>
      </Layout>
    </Layout>
  );
}
