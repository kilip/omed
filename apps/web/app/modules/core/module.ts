import {
  ApartmentOutlined,
  DashboardOutlined,
  KeyOutlined,
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
      key: "dashboard",
      path: "",
      labelKey: "core.menu.dashboard",
      icon: DashboardOutlined,
    },
    {
      key: "members",
      path: "members",
      labelKey: "core.menu.members",
      icon: TeamOutlined,
      hideInPersonalWorkspace: true,
    },
    {
      key: "invitations",
      path: "invitations",
      labelKey: "core.menu.invitations",
      icon: LinkOutlined,
      hideInPersonalWorkspace: true,
    },
    {
      key: "access-tokens",
      path: "access-tokens",
      labelKey: "core.menu.accessTokens",
      icon: KeyOutlined,
    },
  ],
});
