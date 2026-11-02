import { ApiOutlined, RobotOutlined, SettingOutlined } from "@ant-design/icons";
import { defineModule } from "~/lib/modules";

export default defineModule({
  id: "agent",
  labelKey: "agent.label",
  descriptionKey: "agent.description",
  icon: RobotOutlined,
  order: 50,
  menu: [
    {
      key: "providers",
      path: "providers",
      labelKey: "agent.menu.providers",
      icon: ApiOutlined,
    },
    {
      key: "settings",
      path: "settings",
      labelKey: "agent.menu.settings",
      icon: SettingOutlined,
    },
  ],
});
