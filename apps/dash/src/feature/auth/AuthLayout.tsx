import { Layout } from "antd";
import { Content } from "antd/es/layout/layout";
import type { PropsWithChildren } from "react";
export default function AuthLayout({ children }: PropsWithChildren) {
  return (
    <Layout style={{ minHeight: "100vh", backgroundColor: "#f5f5f5" }}>
      <Content
        style={{
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          padding: "24px",
        }}
      >
        <main>{children}</main>
      </Content>
    </Layout>
  );
}
