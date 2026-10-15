import {
  ApartmentOutlined,
  DashboardOutlined,
  LinkOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { defineModule } from "~/lib/modules";

export default defineModule({
  id: "core",
  labelKey: "core.label",
  descriptionKey: "core.description",
  icon: ApartmentOutlined,
  order: 10,
  menu: [
    {
      key: "overview",
      path: "",
      labelKey: "core.menu.overview",
      icon: DashboardOutlined,
    },
    {
      key: "workspaces",
      path: "workspaces",
      labelKey: "core.menu.workspaces",
      icon: ApartmentOutlined,
    },
    {
      key: "members",
      path: "members",
      labelKey: "core.menu.members",
      icon: TeamOutlined,
    },
    {
      key: "invitations",
      path: "invitations",
      labelKey: "core.menu.invitations",
      icon: LinkOutlined,
    },
  ],
});
