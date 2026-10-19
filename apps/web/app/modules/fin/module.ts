import {
  AccountBookOutlined,
  BankOutlined,
  BookOutlined,
  ClusterOutlined,
  DashboardOutlined,
  FileTextOutlined,
} from "@ant-design/icons";
import { defineModule } from "~/lib/modules";

export default defineModule({
  id: "fin",
  labelKey: "fin.label",
  descriptionKey: "fin.description",
  icon: BankOutlined,
  order: 30,
  menu: [
    {
      key: "dashboard",
      path: "",
      labelKey: "fin.menu.dashboard",
      icon: DashboardOutlined,
    },
    {
      key: "accounts",
      path: "accounts",
      labelKey: "fin.menu.accounts",
      icon: ClusterOutlined,
    },
    {
      key: "transactions",
      path: "transactions",
      labelKey: "fin.menu.transactions",
      icon: FileTextOutlined,
    },
    {
      key: "journal",
      path: "journal",
      labelKey: "fin.menu.journal",
      icon: BookOutlined,
    },
    {
      key: "ledger",
      path: "ledger",
      labelKey: "fin.menu.ledger",
      icon: BookOutlined,
    },
    {
      key: "trial-balance",
      path: "trial-balance",
      labelKey: "fin.menu.trialBalance",
      icon: AccountBookOutlined,
    },
  ],
});
