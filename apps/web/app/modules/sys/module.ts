import {
  ApartmentOutlined,
  DashboardOutlined,
  LinkOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { defineModule } from "~/lib/modules";

export default defineModule({
  id: "sys",
  labelKey: "sys.label",
  descriptionKey: "sys.description",
  icon: ApartmentOutlined,
  order: 10,
  menu: [
    {
      key: "overview",
      path: "",
      labelKey: "sys.menu.overview",
      icon: DashboardOutlined,
    },
    {
      key: "workspaces",
      path: "workspaces",
      labelKey: "sys.menu.workspaces",
      icon: ApartmentOutlined,
    },
    {
      key: "members",
      path: "members",
      labelKey: "sys.menu.members",
      icon: TeamOutlined,
    },
    {
      key: "invitations",
      path: "invitations",
      labelKey: "sys.menu.invitations",
      icon: LinkOutlined,
    },
  ],
});
