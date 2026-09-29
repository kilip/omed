import { Layout, Spin } from "antd";

export default function Spinner() {
  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Spin fullscreen spinning={true} size="large" />
    </Layout>
  );
}
