"use client";

import { ToolOutlined } from "@ant-design/icons";
import { Result, Typography } from "antd";

const { Text } = Typography;

export function UnderConstruction({
  title = "Halaman ini masih dalam pengembangan",
  description = "Fitur ini lagi kita kerjain. Balik lagi nanti ya.",
}: {
  title?: string;
  description?: string;
}) {
  return (
    <Result
      icon={<ToolOutlined style={{ color: "#2B4C7E" }} />}
      title={title}
      subTitle={<Text type="secondary">{description}</Text>}
    />
  );
}
