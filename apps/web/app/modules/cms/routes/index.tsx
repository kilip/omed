import {
  FileOutlined,
  FileTextOutlined,
  PictureOutlined,
} from "@ant-design/icons";
import { createElement } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";

const APP_COLORS = 3;

export const cmsLauncherItems = [
  {
    key: "posts",
    to: "/cms/posts",
    labelKey: "cms.menu.posts",
    icon: FileTextOutlined,
  },
  {
    key: "pages",
    to: "/cms/pages",
    labelKey: "cms.menu.pages",
    icon: FileOutlined,
  },
  {
    key: "media",
    to: "/cms/media",
    labelKey: "cms.menu.media",
    icon: PictureOutlined,
  },
];

export default function CmsIndex() {
  const { t } = useTranslation();

  return (
    <div className="home-backdrop flex flex-1 justify-center">
      <div className="flex w-full max-w-[880px] flex-col gap-6 px-4 py-10">
        <div className="text-center">
          <h1 className="m-0 text-2xl font-medium">
            {t("cms.launcher.title", { defaultValue: "Content" })}
          </h1>
          <p style={{ color: "var(--omed-text-secondary)" }}>
            {t("cms.launcher.subtitle", {
              defaultValue: "Posts, pages, and media library",
            })}
          </p>
        </div>
        <div className="grid justify-center gap-2 [grid-template-columns:repeat(auto-fill,minmax(132px,1fr))]">
          {cmsLauncherItems.map((item, index) => (
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
