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
  theme,
} from "antd";
import { type PropsWithChildren, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { clearAuthCache, signOut } from "../auth";
import { useAuth } from "../providers/AuthProvider";
import { useThemeMode } from "../providers/TeamProvider";
import { dashboardMenuItems, findParentKey } from "./menu-items";

const { Header, Sider, Content } = Layout;

const SIDER_WIDTH = 240;
const SIDER_WIDTH_COLLAPSED = 80;

export function DashboardLayout({ children }: PropsWithChildren) {
  const [collapsed, setCollapsed] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const {
    token: { colorBgContainer },
  } = theme.useToken();
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
        className="glass"
        collapsible
        collapsed={collapsed}
        trigger={null}
        width={SIDER_WIDTH}
        collapsedWidth={SIDER_WIDTH_COLLAPSED}
        style={{
          position: "fixed",
          insetInlineStart: 12,
          top: 12,
          bottom: 12,
          zIndex: 100,
          borderRadius: 20,
          overflow: "hidden",
        }}
      >
        <div
          style={{
            height: 56,
            display: "flex",
            alignItems: "center",
            justifyContent: collapsed ? "center" : "flex-start",
            paddingInline: collapsed ? 0 : 20,
            color: "#fff",
            fontWeight: 700,
            fontSize: 18,
            letterSpacing: 0.5,
          }}
        >
          <img src="/omed-mark.svg" alt="Omed" width={28} height={28} />
          {!collapsed && <div style={{ marginLeft: "8px" }}>Omed</div>}
        </div>
        <Menu
          theme={mode}
          mode="inline"
          selectedKeys={[location.pathname]}
          openKeys={collapsed ? undefined : openKeys}
          onOpenChange={(keys) => setOpenKeys(keys)}
          items={dashboardMenuItems}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>

      <Layout
        style={{
          marginInlineStart:
            (collapsed ? SIDER_WIDTH_COLLAPSED : SIDER_WIDTH) + 24,
          transition: "margin-inline-start 0.2s",
        }}
      >
        <Header
          className="glass"
          style={{
            position: "sticky",
            top: 12,
            zIndex: 99,
            margin: "12px 12px 0 0",
            padding: "0 16px",
            borderRadius: 16,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed((v) => !v)}
          />
          <Space size="middle">
            <Button
              type="text"
              shape="circle"
              icon={mode === "dark" ? <SunOutlined /> : <MoonOutlined />}
              onClick={toggle}
            />
            <Dropdown menu={userMenu} placement="bottomRight">
              {/* sama */}
            </Dropdown>
          </Space>
        </Header>

        <Content style={{ margin: "12px 12px 12px 0" }}>
          <div
            className="glass"
            style={{
              padding: 24,
              minHeight: "calc(100vh - 56px - 48px)",
              borderRadius: 20,
            }}
          >
            {children}
          </div>
        </Content>
      </Layout>
    </Layout>
  );
}
