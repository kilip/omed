import { Layout } from "antd";
import { Content } from "antd/es/layout/layout";
import type { PropsWithChildren } from "react";

export default function AuthLayout({ children }: PropsWithChildren) {
  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Content
        style={{
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
        }}
      >
        <main>{children}</main>
      </Content>
    </Layout>
  );
}
