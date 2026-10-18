import {
  AppstoreOutlined,
  FormOutlined,
  UnorderedListOutlined,
} from "@ant-design/icons";
import { Segmented, Tooltip } from "antd";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

export type ViewType = "list" | "kanban" | "form";

const icons: Record<ViewType, ReactNode> = {
  list: <UnorderedListOutlined />,
  kanban: <AppstoreOutlined />,
  form: <FormOutlined />,
};

export type ViewSwitcherProps = {
  value: ViewType;
  onChange: (view: ViewType) => void;
  /** Views the screen offers, in display order. */
  views: ViewType[];
};

/** Odoo-style view switcher: one icon per available view. */
export function ViewSwitcher({ value, onChange, views }: ViewSwitcherProps) {
  const { t } = useTranslation();
  if (views.length < 2) return null;

  return (
    <Segmented<ViewType>
      aria-label={t("common.views")}
      value={value}
      onChange={onChange}
      options={views.map((view) => ({
        value: view,
        label: (
          <Tooltip title={t(`common.view.${view}`)}>
            <span aria-label={t(`common.view.${view}`)} role="img">
              {icons[view]}
            </span>
          </Tooltip>
        ),
      }))}
    />
  );
}
