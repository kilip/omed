import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
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
import { useLocation, useNavigate, useOutletContext } from "react-router";
import type { AppContext } from "~/context";
import { signOut, type User } from "~/shared/auth";
import { dashboardMenuItems, findParentKey } from "./menu-items";

const { Header, Sider, Content } = Layout;

const SIDER_WIDTH = 240;
const SIDER_WIDTH_COLLAPSED = 80;

export function DashboardLayout({
  children,
  user,
}: { user: User } & PropsWithChildren) {
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
          theme="dark"
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
          marginInlineStart: collapsed ? SIDER_WIDTH_COLLAPSED : SIDER_WIDTH,
          transition: "margin-inline-start 0.2s",
        }}
      >
        <Header
          style={{
            position: "sticky",
            top: 0,
            zIndex: 99,
            padding: "0 16px",
            background: colorBgContainer,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            borderBottom: "1px solid rgba(5, 5, 5, 0.06)",
          }}
        >
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed((v) => !v)}
          />

          <Dropdown menu={userMenu} placement="bottomRight">
            <Space style={{ cursor: "pointer" }}>
              <Avatar size="small" icon={<UserOutlined />} />
              <span>{user?.name}</span>
            </Space>
          </Dropdown>
        </Header>

        <Content style={{ margin: 16 }}>
          <div
            style={{
              padding: 24,
              minHeight: "calc(100vh - 56px - 32px)",
              background: colorBgContainer,
              borderRadius: 8,
            }}
          >
            {children}
          </div>
        </Content>
      </Layout>
    </Layout>
  );
}
