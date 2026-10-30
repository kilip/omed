import {
  AccountBookOutlined,
  ArrowDownOutlined,
  ArrowUpOutlined,
  AuditOutlined,
  BankOutlined,
  BookOutlined,
  CalendarOutlined,
  ClusterOutlined,
  DashboardOutlined,
  DollarOutlined,
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
      key: "overview",
      path: "overview",
      labelKey: "fin.menu.overview",
      icon: DashboardOutlined,
    },
    {
      key: "income",
      path: "income",
      labelKey: "fin.menu.income",
      icon: ArrowDownOutlined,
    },
    {
      key: "expense",
      path: "expense",
      labelKey: "fin.menu.expense",
      icon: ArrowUpOutlined,
    },
    {
      key: "accounting",
      labelKey: "fin.menu.accounting",
      icon: AuditOutlined,
      children: [
        {
          key: "accounts",
          path: "accounts",
          labelKey: "fin.menu.accounts",
          icon: ClusterOutlined,
        },
        {
          key: "periods",
          path: "periods",
          labelKey: "fin.menu.periods",
          icon: CalendarOutlined,
        },
        {
          key: "opening-balance",
          path: "opening-balance",
          labelKey: "fin.menu.openingBalance",
          icon: DollarOutlined,
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
    },
  ],
});
