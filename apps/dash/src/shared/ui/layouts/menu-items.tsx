import {
  DashboardOutlined,
  FileTextOutlined,
  SettingOutlined,
  TeamOutlined,
  WalletOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";

export type MenuItem = Required<MenuProps>["items"][number];

// Static demo set — swap to build from role/permission data (Casbin)
// once auth context is wired in.
export const dashboardMenuItems: MenuItem[] = [
  {
    key: "/home",
    icon: <DashboardOutlined />,
    label: "Dashboard",
  },
  {
    key: "/finance",
    icon: <WalletOutlined />,
    label: "Finance",
    children: [
      { key: "/finance/transactions", label: "Transactions" },
      { key: "/finance/invoices", label: "Invoices" },
      { key: "/finance/reports", label: "Reports" },
    ],
  },
  {
    key: "/members",
    icon: <TeamOutlined />,
    label: "Members",
  },
  {
    key: "/documents",
    icon: <FileTextOutlined />,
    label: "Documents",
  },
  {
    key: "/settings",
    icon: <SettingOutlined />,
    label: "Settings",
  },
];
