import {
  AccountBookOutlined,
  BarChartOutlined,
  BookOutlined,
  DollarCircleOutlined,
  EditOutlined,
  FileTextOutlined,
  GlobalOutlined,
  HomeOutlined,
  RedEnvelopeOutlined,
  SettingOutlined,
  WalletOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";

export type MenuItem = Required<MenuProps>["items"][number];

// Static demo set — swap to build from role/permission data (Casbin)
// once auth context is wired in.
export const dashboardMenuItems: MenuItem[] = [
  {
    key: "/home",
    icon: <HomeOutlined />,
    label: "Home",
  },
  {
    key: "blog",
    icon: <GlobalOutlined />,
    label: "Blog",
    children: [
      { key: "/blog/articles", label: "Articles", icon: <EditOutlined /> },
      { key: "/blog/pages", label: "Pages", icon: <BookOutlined /> },
    ],
  },
  {
    key: "finance",
    icon: <WalletOutlined />,
    label: "Finance",
    children: [
      {
        key: "/finance/accounts",
        label: "Accounts",
        icon: <AccountBookOutlined />,
      },
      {
        key: "/finance/transactions",
        label: "Transactions",
        icon: <DollarCircleOutlined />,
      },
      {
        key: "/finance/invoices",
        label: "Invoices",
        icon: <RedEnvelopeOutlined />,
      },
      { key: "/finance/reports", label: "Reports", icon: <BarChartOutlined /> },
    ],
  },
  {
    key: "/settings",
    icon: <SettingOutlined />,
    label: "Settings",
  },
];

// Given the current pathname, find which top-level submenu key (if any)
// contains it — used to auto-open the right submenu on hard refresh /
// direct navigation, since antd Menu doesn't infer this itself.
export function findParentKey(pathname: string): string | null {
  for (const item of dashboardMenuItems) {
    if (item && "children" in item && item.children) {
      const match = item.children.some(
        (child) => child && "key" in child && child.key === pathname,
      );
      if (match) return item.key as string;
    }
  }
  return null;
}
