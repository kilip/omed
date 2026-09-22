"use client";
import { Layout, Menu } from "antd";
import { Content } from "antd/es/layout/layout";
import Sider from "antd/es/layout/Sider";
import { type PropsWithChildren, useState } from "react";
import { menuItems } from "./menu";

export default function DashboardLayout({ children }: PropsWithChildren) {
  const [collapsed, setCollapsed] = useState(false);
  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={(value) => setCollapsed(value)}
      >
        <Menu
          theme="dark"
          defaultSelectedKeys={["home"]}
          mode="inline"
          items={menuItems}
        />
      </Sider>
      <Layout>
        <Content style={{ margin: "0 16px" }}>
          <main>{children}</main>
        </Content>
      </Layout>
    </Layout>
  );
}
