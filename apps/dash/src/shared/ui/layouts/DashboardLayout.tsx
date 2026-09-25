"use client";

import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { Avatar, Button, Dropdown, Layout, Menu, Space, theme } from "antd";
import Image from "next/image";
import { usePathname, useRouter } from "next/navigation";
import { type PropsWithChildren, useState } from "react";
import { useAuth } from "@/shared/providers/AuthProvider";
import { Providers } from "@/shared/providers/Providers";
import { dashboardMenuItems } from "./menu-items";

const { Header, Sider, Content } = Layout;

const SIDER_WIDTH = 240;
const SIDER_WIDTH_COLLAPSED = 80;

function DashboardLayout({ children }: { children: React.ReactNode }) {
  const [collapsed, setCollapsed] = useState(false);
  const router = useRouter();
  const pathname = usePathname();
  const {
    token: { colorBgContainer },
  } = theme.useToken();
  const { user } = useAuth();

  const userMenu = {
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
          <Image
            src="/omed-mark.svg"
            alt="Omed"
            width={28}
            height={28}
            priority
          />
          {!collapsed && "Omed"}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[pathname]}
          items={dashboardMenuItems}
          onClick={({ key }) => router.push(key)}
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

export default function Main({ children }: PropsWithChildren) {
  return (
    <Providers>
      <DashboardLayout>{children}</DashboardLayout>
    </Providers>
  );
}
