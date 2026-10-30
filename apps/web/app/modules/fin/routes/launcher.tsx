import {
  AccountBookOutlined,
  ArrowDownOutlined,
  ArrowUpOutlined,
  BookOutlined,
  CalendarOutlined,
  ClusterOutlined,
  DashboardOutlined,
  DollarOutlined,
  FileTextOutlined,
} from "@ant-design/icons";
import { createElement } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";

const APP_COLORS = 3;

export const finLauncherItems = [
  {
    key: "overview",
    to: "/fin/overview",
    labelKey: "fin.menu.overview",
    icon: DashboardOutlined,
  },
  {
    key: "income",
    to: "/fin/income",
    labelKey: "fin.menu.income",
    icon: ArrowDownOutlined,
  },
  {
    key: "expense",
    to: "/fin/expense",
    labelKey: "fin.menu.expense",
    icon: ArrowUpOutlined,
  },
  {
    key: "accounts",
    to: "/fin/accounts",
    labelKey: "fin.menu.accounts",
    icon: ClusterOutlined,
  },
  {
    key: "periods",
    to: "/fin/periods",
    labelKey: "fin.menu.periods",
    icon: CalendarOutlined,
  },
  {
    key: "opening-balance",
    to: "/fin/opening-balance",
    labelKey: "fin.menu.openingBalance",
    icon: DollarOutlined,
  },
  {
    key: "transactions",
    to: "/fin/transactions",
    labelKey: "fin.menu.transactions",
    icon: FileTextOutlined,
  },
  {
    key: "journal",
    to: "/fin/journal",
    labelKey: "fin.menu.journal",
    icon: BookOutlined,
  },
  {
    key: "ledger",
    to: "/fin/ledger",
    labelKey: "fin.menu.ledger",
    icon: BookOutlined,
  },
  {
    key: "trial-balance",
    to: "/fin/trial-balance",
    labelKey: "fin.menu.trialBalance",
    icon: AccountBookOutlined,
  },
];

/** Finance module sub-launcher: tiles on a dotted backdrop linking to all financial features. */
export default function FinLauncher() {
  const { t } = useTranslation();

  return (
    <div className="home-backdrop flex flex-1 justify-center">
      <div className="flex w-full max-w-[880px] flex-col gap-6 px-4 py-10">
        <div className="text-center">
          <h1 className="m-0 text-2xl font-medium">
            {t("fin.launcher.title")}
          </h1>
          <p style={{ color: "var(--omed-text-secondary)" }}>
            {t("fin.launcher.subtitle")}
          </p>
        </div>
        <div className="grid justify-center gap-2 [grid-template-columns:repeat(auto-fill,minmax(132px,1fr))]">
          {finLauncherItems.map((item, index) => (
            <Link
              key={item.key}
              to={item.to}
              className={`app-tile app-tile--${(index % APP_COLORS) + 1}`}
            >
              <span className="app-tile__icon">{createElement(item.icon)}</span>
              <span className="min-w-0">
                <span className="block font-medium">{t(item.labelKey)}</span>
              </span>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
