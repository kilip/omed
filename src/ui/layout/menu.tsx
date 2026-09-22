import { HomeFilled } from "@ant-design/icons";
import type { MenuProps } from "antd";

export type MenuItem = Required<MenuProps>["items"][number];

export const menuItems: MenuItem[] = [
  {
    label: "Home",
    key: "/home",
    icon: <HomeFilled />,
  },
];
